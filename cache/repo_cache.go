package cache

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/thought-machine/gonduit"
	"github.com/thought-machine/gonduit/core"
	"github.com/thought-machine/gonduit/requests"

	"github.com/daedaleanai/git-ticket/bug"
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/repository"
	"github.com/daedaleanai/git-ticket/util/git"
	"github.com/daedaleanai/git-ticket/util/process"
)

const bugCacheFile = "bug-cache"
const identityCacheFile = "identity-cache"

const configRefPrefix = "refs/configs/"
const configConflictRefPattern = "refs/conflicts/config-%s-%s"
const configRemoteRefPattern = "refs/remotes/%s/configs/"

// 1: original format
// 2: added cache for identities with a reference in the bug cache
const formatVersion = 2

type ErrInvalidCacheFormat struct {
	message string
}

func (e ErrInvalidCacheFormat) Error() string {
	return e.message
}

var _ repository.RepoCommon = &RepoCache{}

// RepoCache is a cache for a Repository. This cache has multiple functions:
//
// 1. After being loaded, a Bug is kept in memory in the cache, allowing for fast
// 		access later.
// 2. The cache maintain in memory and on disk a pre-digested excerpt for each bug,
// 		allowing for fast querying the whole set of bugs without having to load
//		them individually.
// 3. The cache guarantee that a single instance of a Bug is loaded at once, avoiding
// 		loss of data that we could have with multiple copies in the same process.
// 4. The same way, the cache maintain in memory a single copy of the loaded identities.
//
// The cache also protect the on-disk data by locking the git repository for its
// own usage, by writing a lock file. Of course, normal git operations are not
// affected, only git-bug related one.
type RepoCache struct {
	// the underlying repo
	repo repository.ClockedRepo

	// the name of the repository, as defined in the MultiRepoCache
	name string

	muBug sync.RWMutex
	// excerpt of bugs data for all bugs
	bugExcerpts map[entity.Id]*BugExcerpt
	// bug loaded in memory
	bugs map[entity.Id]*BugCache

	muIdentity sync.RWMutex
	// excerpt of identities data for all identities
	identitiesExcerpts map[entity.Id]*IdentityExcerpt
	// identities loaded in memory
	identities map[entity.Id]*IdentityCache

	// the user identity's id, if known
	userIdentityId entity.Id

	muConfig sync.RWMutex
}

func NewRepoCache(r repository.ClockedRepo) (*RepoCache, error) {
	return NewNamedRepoCache(r, "")
}

func NewNamedRepoCache(r repository.ClockedRepo, name string) (*RepoCache, error) {
	c := &RepoCache{
		repo:       r,
		name:       name,
		bugs:       make(map[entity.Id]*BugCache),
		identities: make(map[entity.Id]*IdentityCache),
	}

	err := c.lock()
	if err != nil {
		return &RepoCache{}, err
	}

	err = c.load()
	if err == nil {
		return c, nil
	}
	if _, ok := err.(ErrInvalidCacheFormat); ok {
		return nil, err
	}

	err = c.buildCache()
	if err != nil {
		return nil, err
	}

	return c, c.write()
}

func (c *RepoCache) Name() string {
	return c.name
}

// LocalConfig give access to the repository scoped configuration
func (c *RepoCache) LocalConfig() repository.Config {
	return c.repo.LocalConfig()
}

// GlobalConfig give access to the git global configuration
func (c *RepoCache) GlobalConfig() repository.Config {
	return c.repo.GlobalConfig()
}

// GetPath returns the path to the repo.
func (c *RepoCache) GetPath() string {
	return c.repo.GetPath()
}

// GetCoreEditor returns the name of the editor that the user has used to configure git.
func (c *RepoCache) GetCoreEditor() (string, error) {
	return c.repo.GetCoreEditor()
}

// GetRemotes returns the configured remotes repositories.
func (c *RepoCache) GetRemotes() (map[string]string, error) {
	return c.repo.GetRemotes()
}

// GetUserName returns the name the the user has used to configure git
func (c *RepoCache) GetUserName() (string, error) {
	return c.repo.GetUserName()
}

// GetUserEmail returns the email address that the user has used to configure git.
func (c *RepoCache) GetUserEmail() (string, error) {
	return c.repo.GetUserEmail()
}

func (c *RepoCache) lock() error {
	lockPath := repoLockFilePath(c.repo)

	err := repoIsAvailable(c.repo)
	if err != nil {
		return err
	}

	f, err := os.Create(lockPath)
	if err != nil {
		return err
	}

	pid := fmt.Sprintf("%d", os.Getpid())
	_, err = f.WriteString(pid)
	if err != nil {
		return err
	}

	return f.Close()
}

func (c *RepoCache) Close() error {
	c.muBug.Lock()
	defer c.muBug.Unlock()
	c.muIdentity.Lock()
	defer c.muIdentity.Unlock()

	c.identities = make(map[entity.Id]*IdentityCache)
	c.identitiesExcerpts = nil
	c.bugs = make(map[entity.Id]*BugCache)
	c.bugExcerpts = nil

	lockPath := repoLockFilePath(c.repo)
	return os.Remove(lockPath)
}

// bugUpdated is a callback to trigger when the excerpt of a bug changed,
// that is each time a bug is updated
func (c *RepoCache) bugUpdated(id entity.Id) error {
	c.muBug.Lock()

	b, ok := c.bugs[id]
	if !ok {
		c.muBug.Unlock()
		panic("missing bug in the cache")
	}

	c.bugExcerpts[id] = NewBugExcerpt(b.bug, b.Snapshot())
	c.muBug.Unlock()

	// we only need to write the bug cache
	return c.writeBugCache()
}

// identityUpdated is a callback to trigger when the excerpt of an identity
// changed, that is each time an identity is updated
func (c *RepoCache) identityUpdated(id entity.Id) error {
	c.muIdentity.Lock()

	i, ok := c.identities[id]
	if !ok {
		c.muIdentity.Unlock()
		panic("missing identity in the cache")
	}

	c.identitiesExcerpts[id] = NewIdentityExcerpt(i.Identity)
	c.muIdentity.Unlock()

	// we only need to write the identity cache
	return c.writeIdentityCache()
}

// load will try to read from the disk all the cache files
func (c *RepoCache) load() error {
	err := c.loadBugCache()
	if err != nil {
		return err
	}
	return c.loadIdentityCache()
}

// load will try to read from the disk the bug cache file
func (c *RepoCache) loadBugCache() error {
	c.muBug.Lock()
	defer c.muBug.Unlock()

	f, err := os.Open(bugCacheFilePath(c.repo))
	if err != nil {
		return err
	}

	decoder := gob.NewDecoder(f)

	aux := struct {
		Version  uint
		Excerpts map[entity.Id]*BugExcerpt
	}{}

	err = decoder.Decode(&aux)
	if err != nil {
		return err
	}

	if aux.Version != 2 {
		return ErrInvalidCacheFormat{
			message: fmt.Sprintf("unknown cache format version %v", aux.Version),
		}
	}

	c.bugExcerpts = aux.Excerpts
	return nil
}

// load will try to read from the disk the identity cache file
func (c *RepoCache) loadIdentityCache() error {
	c.muIdentity.Lock()
	defer c.muIdentity.Unlock()

	f, err := os.Open(identityCacheFilePath(c.repo))
	if err != nil {
		return err
	}

	decoder := gob.NewDecoder(f)

	aux := struct {
		Version  uint
		Excerpts map[entity.Id]*IdentityExcerpt
	}{}

	err = decoder.Decode(&aux)
	if err != nil {
		return err
	}

	if aux.Version != 2 {
		return ErrInvalidCacheFormat{
			message: fmt.Sprintf("unknown cache format version %v", aux.Version),
		}
	}

	c.identitiesExcerpts = aux.Excerpts
	return nil
}

// write will serialize on disk all the cache files
func (c *RepoCache) write() error {
	err := c.writeBugCache()
	if err != nil {
		return err
	}
	return c.writeIdentityCache()
}

// write will serialize on disk the bug cache file
func (c *RepoCache) writeBugCache() error {
	c.muBug.RLock()
	defer c.muBug.RUnlock()

	var data bytes.Buffer

	aux := struct {
		Version  uint
		Excerpts map[entity.Id]*BugExcerpt
	}{
		Version:  formatVersion,
		Excerpts: c.bugExcerpts,
	}

	encoder := gob.NewEncoder(&data)

	err := encoder.Encode(aux)
	if err != nil {
		return err
	}

	f, err := os.Create(bugCacheFilePath(c.repo))
	if err != nil {
		return err
	}

	_, err = f.Write(data.Bytes())
	if err != nil {
		return err
	}

	return f.Close()
}

// write will serialize on disk the identity cache file
func (c *RepoCache) writeIdentityCache() error {
	c.muIdentity.RLock()
	defer c.muIdentity.RUnlock()

	var data bytes.Buffer

	aux := struct {
		Version  uint
		Excerpts map[entity.Id]*IdentityExcerpt
	}{
		Version:  formatVersion,
		Excerpts: c.identitiesExcerpts,
	}

	encoder := gob.NewEncoder(&data)

	err := encoder.Encode(aux)
	if err != nil {
		return err
	}

	f, err := os.Create(identityCacheFilePath(c.repo))
	if err != nil {
		return err
	}

	_, err = f.Write(data.Bytes())
	if err != nil {
		return err
	}

	return f.Close()
}

func bugCacheFilePath(repo repository.Repo) string {
	return path.Join(repo.GetPath(), "git-bug", bugCacheFile)
}

func identityCacheFilePath(repo repository.Repo) string {
	return path.Join(repo.GetPath(), "git-bug", identityCacheFile)
}

func (c *RepoCache) buildCache() error {
	c.muBug.Lock()
	defer c.muBug.Unlock()
	c.muIdentity.Lock()
	defer c.muIdentity.Unlock()

	_, _ = fmt.Fprintf(os.Stderr, "Building identity cache... ")

	c.identitiesExcerpts = make(map[entity.Id]*IdentityExcerpt)

	allIdentities := identity.ReadAllLocalIdentities(c.repo)

	for i := range allIdentities {
		if i.Err != nil {
			return i.Err
		}

		c.identitiesExcerpts[i.Identity.Id()] = NewIdentityExcerpt(i.Identity)
	}

	_, _ = fmt.Fprintln(os.Stderr, "Done.")

	_, _ = fmt.Fprintf(os.Stderr, "Building bug cache... ")

	c.bugExcerpts = make(map[entity.Id]*BugExcerpt)

	allBugs := bug.ReadAllLocalBugs(c.repo)

	for b := range allBugs {
		if b.Err != nil {
			return b.Err
		}

		snap := b.Bug.Compile()
		c.bugExcerpts[b.Bug.Id()] = NewBugExcerpt(b.Bug, &snap)
	}

	_, _ = fmt.Fprintln(os.Stderr, "Done.")
	return nil
}

// ResolveBugExcerpt retrieve a BugExcerpt matching the exact given id
func (c *RepoCache) ResolveBugExcerpt(id entity.Id) (*BugExcerpt, error) {
	c.muBug.RLock()
	defer c.muBug.RUnlock()

	e, ok := c.bugExcerpts[id]
	if !ok {
		return nil, bug.ErrBugNotExist
	}

	return e, nil
}

// ResolveBug retrieve a bug matching the exact given id
func (c *RepoCache) ResolveBug(id entity.Id) (*BugCache, error) {
	c.muBug.RLock()
	cached, ok := c.bugs[id]
	c.muBug.RUnlock()
	if ok {
		return cached, nil
	}

	b, err := bug.ReadLocalBug(c.repo, id)
	if err != nil {
		return nil, err
	}

	cached = NewBugCache(c, b)

	c.muBug.Lock()
	c.bugs[id] = cached
	c.muBug.Unlock()

	return cached, nil
}

// ResolveBugExcerptPrefix retrieve a BugExcerpt matching an id prefix. It fails if multiple
// bugs match.
func (c *RepoCache) ResolveBugExcerptPrefix(prefix string) (*BugExcerpt, error) {
	return c.ResolveBugExcerptMatcher(func(excerpt *BugExcerpt) bool {
		return excerpt.Id.HasPrefix(prefix)
	})
}

// ResolveBugPrefix retrieve a bug matching an id prefix. It fails if multiple
// bugs match.
func (c *RepoCache) ResolveBugPrefix(prefix string) (*BugCache, error) {
	return c.ResolveBugMatcher(func(excerpt *BugExcerpt) bool {
		return excerpt.Id.HasPrefix(prefix)
	})
}

// ResolveBugCreateMetadata retrieve a bug that has the exact given metadata on
// its Create operation, that is, the first operation. It fails if multiple bugs
// match.
func (c *RepoCache) ResolveBugCreateMetadata(key string, value string) (*BugCache, error) {
	return c.ResolveBugMatcher(func(excerpt *BugExcerpt) bool {
		return excerpt.CreateMetadata[key] == value
	})
}

func (c *RepoCache) ResolveBugExcerptMatcher(f func(*BugExcerpt) bool) (*BugExcerpt, error) {
	id, err := c.resolveBugMatcher(f)
	if err != nil {
		return nil, err
	}
	return c.ResolveBugExcerpt(id)
}

func (c *RepoCache) ResolveBugMatcher(f func(*BugExcerpt) bool) (*BugCache, error) {
	id, err := c.resolveBugMatcher(f)
	if err != nil {
		return nil, err
	}
	return c.ResolveBug(id)
}

func (c *RepoCache) resolveBugMatcher(f func(*BugExcerpt) bool) (entity.Id, error) {
	c.muBug.RLock()
	defer c.muBug.RUnlock()

	// preallocate but empty
	matching := make([]entity.Id, 0, 5)

	for _, excerpt := range c.bugExcerpts {
		if f(excerpt) {
			matching = append(matching, excerpt.Id)
		}
	}

	if len(matching) > 1 {
		return entity.UnsetId, bug.NewErrMultipleMatchBug(matching)
	}

	if len(matching) == 0 {
		return entity.UnsetId, bug.ErrBugNotExist
	}

	return matching[0], nil
}

// QueryBugs return the id of all Bug matching the given Query
func (c *RepoCache) QueryBugs(query *Query) []entity.Id {
	c.muBug.RLock()
	defer c.muBug.RUnlock()

	if query == nil {
		return c.AllBugsIds()
	}

	var filtered []*BugExcerpt

	for _, excerpt := range c.bugExcerpts {
		if query.Match(excerpt, c) {
			filtered = append(filtered, excerpt)
		}
	}

	var sorter sort.Interface

	switch query.OrderBy {
	case OrderById:
		sorter = BugsById(filtered)
	case OrderByCreation:
		sorter = BugsByCreationTime(filtered)
	case OrderByEdit:
		sorter = BugsByEditTime(filtered)
	default:
		panic("missing sort type")
	}

	if query.OrderDirection == OrderDescending {
		sorter = sort.Reverse(sorter)
	}

	sort.Sort(sorter)

	result := make([]entity.Id, len(filtered))

	for i, val := range filtered {
		result[i] = val.Id
	}

	return result
}

// AllBugsIds return all known bug ids
func (c *RepoCache) AllBugsIds() []entity.Id {
	c.muBug.RLock()
	defer c.muBug.RUnlock()

	result := make([]entity.Id, len(c.bugExcerpts))

	i := 0
	for _, excerpt := range c.bugExcerpts {
		result[i] = excerpt.Id
		i++
	}

	return result
}

// ValidLabels list valid labels
//
// Note: in the future, a proper label policy could be implemented where valid
// labels are defined in a configuration file. Until that, the default behavior
// is to return the list of labels already used.
func (c *RepoCache) ValidLabels() []bug.Label {
	c.muBug.RLock()
	defer c.muBug.RUnlock()

	set := map[bug.Label]interface{}{}

	for _, excerpt := range c.bugExcerpts {
		for _, l := range excerpt.Labels {
			set[l] = nil
		}
	}

	result := make([]bug.Label, len(set))

	i := 0
	for l := range set {
		result[i] = l
		i++
	}

	// Sort
	sort.Slice(result, func(i, j int) bool {
		return string(result[i]) < string(result[j])
	})

	return result
}

// NewBug create a new bug
// The new bug is written in the repository (commit)
func (c *RepoCache) NewBug(title string, message string) (*BugCache, *bug.CreateOperation, error) {
	return c.NewBugWithFiles(title, message, nil)
}

// NewBugWithFiles create a new bug with attached files for the message
// The new bug is written in the repository (commit)
func (c *RepoCache) NewBugWithFiles(title string, message string, files []git.Hash) (*BugCache, *bug.CreateOperation, error) {
	author, err := c.GetUserIdentity()
	if err != nil {
		return nil, nil, err
	}

	return c.NewBugRaw(author, time.Now().Unix(), title, message, files, nil)
}

// NewBugWithFilesMeta create a new bug with attached files for the message, as
// well as metadata for the Create operation.
// The new bug is written in the repository (commit)
func (c *RepoCache) NewBugRaw(author *IdentityCache, unixTime int64, title string, message string, files []git.Hash, metadata map[string]string) (*BugCache, *bug.CreateOperation, error) {
	b, op, err := bug.CreateWithFiles(author.Identity, unixTime, title, message, files)
	if err != nil {
		return nil, nil, err
	}

	for key, value := range metadata {
		op.SetMetadata(key, value)
	}

	err = b.Commit(c.repo)
	if err != nil {
		return nil, nil, err
	}

	c.muBug.Lock()
	if _, has := c.bugs[b.Id()]; has {
		c.muBug.Unlock()
		return nil, nil, fmt.Errorf("bug %s already exist in the cache", b.Id())
	}

	cached := NewBugCache(c, b)
	c.bugs[b.Id()] = cached
	c.muBug.Unlock()

	// force the write of the excerpt
	err = c.bugUpdated(b.Id())
	if err != nil {
		return nil, nil, err
	}

	return cached, op, nil
}

// Fetch retrieve updates from a remote
// This does not change the local bugs or identities state
func (c *RepoCache) Fetch(remote string) (string, error) {
	stdout1, err := identity.Fetch(c.repo, remote)
	if err != nil {
		return stdout1, err
	}

	stdout2, err := bug.Fetch(c.repo, remote)
	if err != nil {
		return stdout2, err
	}

	// Config
	remoteRefSpec := fmt.Sprintf(configRemoteRefPattern, remote)
	fetchRefSpec := fmt.Sprintf("%s*:%s*", configRefPrefix, remoteRefSpec)

	stdout3, err := c.repo.FetchRefs(remote, fetchRefSpec)
	if err != nil {
		return stdout3, err
	}

	return stdout1 + stdout2 + stdout3, nil
}

// MergeAll will merge all the available remote bug and identities
func (c *RepoCache) MergeAll(remote string) <-chan entity.MergeResult {
	out := make(chan entity.MergeResult)

	// Intercept merge results to update the cache properly
	go func() {
		defer close(out)

		results := identity.MergeAll(c.repo, remote)
		for result := range results {
			out <- result

			if result.Err != nil {
				continue
			}

			switch result.Status {
			case entity.MergeStatusNew, entity.MergeStatusUpdated:
				i := result.Entity.(*identity.Identity)
				c.muIdentity.Lock()
				c.identitiesExcerpts[result.Id] = NewIdentityExcerpt(i)
				c.muIdentity.Unlock()
			}
		}

		results = bug.MergeAll(c.repo, remote)
		for result := range results {
			out <- result

			if result.Err != nil {
				continue
			}

			switch result.Status {
			case entity.MergeStatusNew, entity.MergeStatusUpdated:
				b := result.Entity.(*bug.Bug)
				snap := b.Compile()
				c.muBug.Lock()
				c.bugExcerpts[result.Id] = NewBugExcerpt(b, &snap)
				c.muBug.Unlock()
			}
		}

		err := c.write()

		// No easy way out here ..
		if err != nil {
			panic(err)
		}
	}()

	return out
}

// Push update a remote with the local changes
func (c *RepoCache) Push(remote string) (string, error) {
	stdout1, err := identity.Push(c.repo, remote)
	if err != nil {
		return stdout1, err
	}

	stdout2, err := bug.Push(c.repo, remote)
	if err != nil {
		return stdout2, err
	}

	stdout3, err := c.repo.PushRefs(remote, configRefPrefix+"*")
	if err != nil {
		return stdout3, err
	}

	return stdout1 + stdout2 + stdout3, nil
}

// Pull will do a Fetch + MergeAll
// This function will return an error if a merge fail
func (c *RepoCache) Pull(remote string) error {
	_, err := c.Fetch(remote)
	if err != nil {
		return err
	}

	for merge := range c.MergeAll(remote) {
		if merge.Err != nil {
			return merge.Err
		}
		if merge.Status == entity.MergeStatusInvalid {
			return errors.Errorf("merge failure: %s", merge.Reason)
		}
	}

	err = c.MergeConfigs(remote)
	if err != nil {
		return err
	}

	return nil
}

func repoLockFilePath(repo repository.Repo) string {
	return path.Join(repo.GetPath(), "git-bug", lockfile)
}

// repoIsAvailable check is the given repository is locked by a Cache.
// Note: this is a smart function that will cleanup the lock file if the
// corresponding process is not there anymore.
// If no error is returned, the repo is free to edit.
func repoIsAvailable(repo repository.Repo) error {
	lockPath := repoLockFilePath(repo)

	// Todo: this leave way for a racey access to the repo between the test
	// if the file exist and the actual write. It's probably not a problem in
	// practice because using a repository will be done from user interaction
	// or in a context where a single instance of git-bug is already guaranteed
	// (say, a server with the web UI running). But still, that might be nice to
	// have a mutex or something to guard that.

	// Todo: this will fail if somehow the filesystem is shared with another
	// computer. Should add a configuration that prevent the cleaning of the
	// lock file

	f, err := os.Open(lockPath)

	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if err == nil {
		// lock file already exist
		buf, err := ioutil.ReadAll(io.LimitReader(f, 10))
		if err != nil {
			return err
		}
		if len(buf) == 10 {
			return fmt.Errorf("the lock file should be < 10 bytes")
		}

		pid, err := strconv.Atoi(string(buf))
		if err != nil {
			return err
		}

		if process.IsRunning(pid) {
			return fmt.Errorf("the repository you want to access is already locked by the process pid %d", pid)
		}

		// The lock file is just laying there after a crash, clean it

		fmt.Println("A lock file is present but the corresponding process is not, removing it.")
		err = f.Close()
		if err != nil {
			return err
		}

		err = os.Remove(lockPath)
		if err != nil {
			return err
		}
	}

	return nil
}

// ResolveIdentityExcerpt retrieve a IdentityExcerpt matching the exact given id
func (c *RepoCache) ResolveIdentityExcerpt(id entity.Id) (*IdentityExcerpt, error) {
	c.muIdentity.RLock()
	defer c.muIdentity.RUnlock()

	e, ok := c.identitiesExcerpts[id]
	if !ok {
		return nil, identity.ErrIdentityNotExist
	}

	return e, nil
}

// ResolveIdentity retrieve an identity matching the exact given id
func (c *RepoCache) ResolveIdentity(id entity.Id) (*IdentityCache, error) {
	c.muIdentity.RLock()
	cached, ok := c.identities[id]
	c.muIdentity.RUnlock()
	if ok {
		return cached, nil
	}

	i, err := identity.ReadLocal(c.repo, id)
	if err != nil {
		return nil, err
	}

	cached = NewIdentityCache(c, i)

	c.muIdentity.Lock()
	c.identities[id] = cached
	c.muIdentity.Unlock()

	return cached, nil
}

// ResolveIdentityExcerptPrefix retrieve a IdentityExcerpt matching an id prefix.
// It fails if multiple identities match.
func (c *RepoCache) ResolveIdentityExcerptPrefix(prefix string) (*IdentityExcerpt, error) {
	return c.ResolveIdentityExcerptMatcher(func(excerpt *IdentityExcerpt) bool {
		return excerpt.Id.HasPrefix(prefix)
	})
}

// ResolveIdentityPhabID retrieve an Identity matching a Phabricator ID.
// It fails if multiple identities match.
func (c *RepoCache) ResolveIdentityPhabID(phabID string) (*IdentityCache, error) {
	return c.ResolveIdentityMatcher(func(excerpt *IdentityExcerpt) bool {
		return excerpt.PhabID == phabID
	})
}

// ResolveIdentityPrefix retrieve an Identity matching an id prefix.
// It fails if multiple identities match.
func (c *RepoCache) ResolveIdentityPrefix(prefix string) (*IdentityCache, error) {
	return c.ResolveIdentityMatcher(func(excerpt *IdentityExcerpt) bool {
		return excerpt.Id.HasPrefix(prefix)
	})
}

// ResolveIdentityImmutableMetadata retrieve an Identity that has the exact given metadata on
// one of it's version. If multiple version have the same key, the first defined take precedence.
func (c *RepoCache) ResolveIdentityImmutableMetadata(key string, value string) (*IdentityCache, error) {
	return c.ResolveIdentityMatcher(func(excerpt *IdentityExcerpt) bool {
		return excerpt.ImmutableMetadata[key] == value
	})
}

func (c *RepoCache) ResolveIdentityExcerptMatcher(f func(*IdentityExcerpt) bool) (*IdentityExcerpt, error) {
	id, err := c.resolveIdentityMatcher(f)
	if err != nil {
		return nil, err
	}
	return c.ResolveIdentityExcerpt(id)
}

func (c *RepoCache) ResolveIdentityMatcher(f func(*IdentityExcerpt) bool) (*IdentityCache, error) {
	id, err := c.resolveIdentityMatcher(f)
	if err != nil {
		return nil, err
	}
	return c.ResolveIdentity(id)
}

func (c *RepoCache) resolveIdentityMatcher(f func(*IdentityExcerpt) bool) (entity.Id, error) {
	c.muIdentity.RLock()
	defer c.muIdentity.RUnlock()

	// preallocate but empty
	matching := make([]entity.Id, 0, 5)

	for _, excerpt := range c.identitiesExcerpts {
		if f(excerpt) {
			matching = append(matching, excerpt.Id)
		}
	}

	if len(matching) > 1 {
		return entity.UnsetId, identity.NewErrMultipleMatch(matching)
	}

	if len(matching) == 0 {
		return entity.UnsetId, identity.ErrIdentityNotExist
	}

	return matching[0], nil
}

// AllIdentityIds return all known identity ids
func (c *RepoCache) AllIdentityIds() []entity.Id {
	c.muIdentity.RLock()
	defer c.muIdentity.RUnlock()

	result := make([]entity.Id, len(c.identitiesExcerpts))

	i := 0
	for _, excerpt := range c.identitiesExcerpts {
		result[i] = excerpt.Id
		i++
	}

	return result
}

func (c *RepoCache) SetUserIdentity(i *IdentityCache) error {
	err := identity.SetUserIdentity(c.repo, i.Identity)
	if err != nil {
		return err
	}

	c.muIdentity.RLock()
	defer c.muIdentity.RUnlock()

	// Make sure that everything is fine
	if _, ok := c.identities[i.Id()]; !ok {
		panic("SetUserIdentity while the identity is not from the cache, something is wrong")
	}

	c.userIdentityId = i.Id()

	return nil
}

func (c *RepoCache) GetUserIdentity() (*IdentityCache, error) {
	if c.userIdentityId != "" {
		i, ok := c.identities[c.userIdentityId]
		if ok {
			return i, nil
		}
	}

	c.muIdentity.Lock()
	defer c.muIdentity.Unlock()

	i, err := identity.GetUserIdentity(c.repo)
	if err != nil {
		return nil, err
	}

	cached := NewIdentityCache(c, i)
	c.identities[i.Id()] = cached
	c.userIdentityId = i.Id()

	return cached, nil
}

func (c *RepoCache) GetUserIdentityExcerpt() (*IdentityExcerpt, error) {
	if c.userIdentityId == "" {
		id, err := identity.GetUserIdentityId(c.repo)
		if err != nil {
			return nil, err
		}
		c.userIdentityId = id
	}

	c.muIdentity.RLock()
	defer c.muIdentity.RUnlock()

	excerpt, ok := c.identitiesExcerpts[c.userIdentityId]
	if !ok {
		return nil, fmt.Errorf("cache: missing identity excerpt %v", c.userIdentityId)
	}
	return excerpt, nil
}

func (c *RepoCache) IsUserIdentitySet() (bool, error) {
	return identity.IsUserIdentitySet(c.repo)
}

func (c *RepoCache) NewIdentityFromGitUser() (*IdentityCache, error) {
	return c.NewIdentityFromGitUserRaw(nil)
}

func (c *RepoCache) NewIdentityFromGitUserRaw(metadata map[string]string) (*IdentityCache, error) {
	i, err := identity.NewFromGitUser(c.repo)
	if err != nil {
		return nil, err
	}
	return c.finishIdentity(i, metadata)
}

// NewIdentity create a new identity
// The new identity is written in the repository (commit)
func (c *RepoCache) NewIdentity(name string, email string) (*IdentityCache, error) {
	return c.NewIdentityRaw(name, email, "", "", nil)
}

// NewIdentityFull create a new identity
// The new identity is written in the repository (commit)
func (c *RepoCache) NewIdentityFull(name string, email string, login string, avatarUrl string) (*IdentityCache, error) {
	return c.NewIdentityRaw(name, email, login, avatarUrl, nil)
}

func (c *RepoCache) NewIdentityRaw(name string, email string, login string, avatarUrl string, metadata map[string]string) (*IdentityCache, error) {
	// attempt to populate the phabricator ID, for now it's not fatal if it fails
	phabId, _ := c.getPhabId(email)

	i := identity.NewIdentityFull(name, email, login, avatarUrl, phabId)
	return c.finishIdentity(i, metadata)
}

// UpdatedIdentity updates an existing identity in the repository and cache
func (c *RepoCache) UpdateIdentity(i *IdentityCache, name string, email string, login string, avatarUrl string) error {
	// attempt to populate the phabricator ID, for now it's not fatal if it fails
	phabId, _ := c.getPhabId(email)

	err := i.Mutate(func(mutator identity.Mutator) identity.Mutator {
		mutator.Name = name
		mutator.Email = email
		mutator.AvatarUrl = avatarUrl
		mutator.PhabID = phabId
		return mutator
	})

	if err != nil {
		return err
	}

	err = i.CommitAsNeeded()
	if err != nil {
		return err
	}

	c.muIdentity.Lock()
	c.identities[i.Id()] = i
	c.muIdentity.Unlock()

	// force the write of the excerpt
	return c.identityUpdated(i.Id())
}

func (c *RepoCache) getPhabId(email string) (string, error) {
	// Assuming that the e-mail prefix is username on Phabricator
	user := email[0:strings.Index(email, "@")]

	// We need an API token to access Phabricator through conduit
	var apiToken string
	var err error
	if apiToken, err = c.LocalConfig().ReadString("daedalean.taskmgr-api-token"); err != nil {
		if apiToken, err = c.GlobalConfig().ReadString("daedalean.taskmgr-api-token"); err != nil {
			msg := `No Phabricator API token set. Please go to
	https://p.daedalean.ai/settings/user/<YOUR_USERNAME_HERE>/page/apitokens/
click on <Generate API Token>, and then paste the token into this command
	git config --global --replace-all daedalean.taskmgr-api-token <PASTE_TOKEN_HERE>`
			return "", errors.New(msg)
		}
	}

	phabClient, err := gonduit.Dial("https://p.daedalean.ai", &core.ClientOptions{APIToken: apiToken})
	if err != nil {
		return "", err
	}

	request := requests.SearchRequest{Constraints: map[string]interface{}{"usernames": []string{user}}}

	response, err := phabClient.UserSearch(request)
	if err != nil {
		return "", err
	}

	if len(response.Data) == 0 {
		return "", fmt.Errorf("no Phabricator users matching %s", user)
	}

	return response.Data[0].PHID, nil
}

func (c *RepoCache) finishIdentity(i *identity.Identity, metadata map[string]string) (*IdentityCache, error) {
	for key, value := range metadata {
		i.SetMetadata(key, value)
	}

	err := i.Commit(c.repo)
	if err != nil {
		return nil, err
	}

	c.muIdentity.Lock()
	if _, has := c.identities[i.Id()]; has {
		return nil, fmt.Errorf("identity %s already exist in the cache", i.Id())
	}

	cached := NewIdentityCache(c, i)
	c.identities[i.Id()] = cached
	c.muIdentity.Unlock()

	// force the write of the excerpt
	err = c.identityUpdated(i.Id())
	if err != nil {
		return nil, err
	}

	return cached, nil
}

// List configurations stored in git
func (c *RepoCache) ListConfigs() ([]string, error) {
	c.muConfig.RLock()
	defer c.muConfig.RUnlock()

	refs, err := c.repo.ListRefs(configRefPrefix)
	if err != nil {
		return nil, fmt.Errorf("cache: failed to get a list of config refs: %s", err)
	}

	configs := make([]string, 0)
	for _, ref := range refs {
		configs = append(configs, ref[len(configRefPrefix):])
	}

	return configs, nil
}

// Store the configuration data under the given name
func (c *RepoCache) SetConfig(name string, config []byte) error {
	c.muConfig.Lock()
	defer c.muConfig.Unlock()

	// Store the blob in the repository
	blobHash, err := c.repo.StoreData(config)
	if err != nil {
		return fmt.Errorf("cache: failed to store config data in git: %s", err)
	}

	// Make a tree referencing the blob
	tree := []repository.TreeEntry{
		{
			ObjectType: repository.Blob,
			Hash:       blobHash,
			Name:       "config.json",
		},
	}

	// Store the tree in git
	treeHash, err := c.repo.StoreTree(tree)
	if err != nil {
		return fmt.Errorf("cache: failed to store the config tree in git: %s", err)
	}

	refName := configRefPrefix + name
	exists, err := c.repo.RefExist(refName)
	if err != nil {
		return fmt.Errorf("cache: failed to determine if ref %s exists: %s", refName, err)
	}

	var commitHash git.Hash
	if exists {
		oldCommitHash, err := c.repo.ResolveRef(refName)
		if err != nil {
			return fmt.Errorf("cache: failed to resolve ref %s: %s", refName, err)
		}

		oldTreeHash, err := c.repo.GetTreeHash(oldCommitHash)
		if err != nil {
			return fmt.Errorf("cache: failed to get the tree for commit %s (ref: %s): %s", oldCommitHash, refName, err)
		}

		// The same tree has as in the previous commit means that nothing has actually changed.
		// This means success
		if oldTreeHash == treeHash {
			return nil
		}

		// Otherwise we reference the old commit as the parent commit
		commitHash, err = c.repo.StoreCommitWithParent(treeHash, oldCommitHash)
	} else {
		commitHash, err = c.repo.StoreCommit(treeHash)
	}

	if err != nil {
		return fmt.Errorf("cache: failed to make a git commit: %s", err)
	}

	err = c.repo.UpdateRef(refName, commitHash)
	if err != nil {
		return fmt.Errorf("cache: failed to update git ref %s: %s", refName, err)
	}

	return nil
}

// Get the named configuration data
func (c *RepoCache) GetConfig(name string) ([]byte, error) {
	c.muConfig.RLock()
	defer c.muConfig.RUnlock()

	refName := configRefPrefix + name
	commitHash, err := c.repo.ResolveRef(refName)
	if err != nil {
		return nil, fmt.Errorf("cache: failed to resolve ref %s: %s", refName, err)
	}

	treeHash, err := c.repo.GetTreeHash(commitHash)
	if err != nil {
		return nil, fmt.Errorf("cache: failed to get the tree for commit %s (ref: %s): %s", commitHash, refName, err)
	}

	tree, err := c.repo.ListEntries(treeHash)
	if err != nil {
		return nil, fmt.Errorf("cache: failed to list entries in tree %s (ref: %s): %s", treeHash, refName, err)
	}

	for _, entry := range tree {
		if entry.ObjectType != repository.Blob {
			continue
		}

		if entry.Name == "config.json" {
			data, err := c.repo.ReadData(entry.Hash)
			if err != nil {
				return nil, fmt.Errorf("cache: failed to get data for blob %s (ref: %s): %s", entry.Hash, refName, err)
			}
			return data, nil
		}
	}
	return nil, fmt.Errorf(
		`cache: failed to find "config.json" blob in the tree corresponding to the HEAD of %s`,
		refName)
}

// Merge the configuration data fetched from the remote
func (c *RepoCache) MergeConfigs(remote string) error {
	c.muConfig.RLock()
	defer c.muConfig.RUnlock()

	remoteRefSpec := fmt.Sprintf(configRemoteRefPattern, remote)
	remoteRefs, err := c.repo.ListRefs(remoteRefSpec)

	if err != nil {
		return fmt.Errorf("cache: failed to list refs for spec %s: %s", remoteRefSpec, err)
	}

	for _, remoteRef := range remoteRefs {
		refName := remoteRef[len(remoteRefSpec):]
		localRef := configRefPrefix + refName

		exist, err := c.repo.RefExist(localRef)
		if err != nil {
			return fmt.Errorf("cache: failed to check if ref exists %s: %s", refName, err)
		}

		if !exist {
			err = c.repo.CopyRef(remoteRef, localRef)
			if err != nil {
				return fmt.Errorf("cache: failed to copy %s to %s: %s", remoteRef, localRef, err)
			}
			return nil
		}

		localCommit, err := c.repo.ResolveRef(localRef)
		if err != nil {
			return fmt.Errorf("cache: failed to resolve %s: %s", localRef, err)
		}
		remoteCommit, err := c.repo.ResolveRef(remoteRef)
		if err != nil {
			return fmt.Errorf("cache: failed to resolve %s: %s", remoteRef, err)
		}

		// Refs are the same
		if localCommit == remoteCommit {
			continue
		}

		ancestor, err := c.repo.FindCommonAncestor(localCommit, remoteCommit)
		if err != nil {
			return fmt.Errorf("cache: failed to get common ancestor of merge commits for %s: %s", refName, err)
		}

		// Local commit is a child of the remote commit
		if ancestor == remoteCommit {
			continue
		}

		// Local and remote configurations diverged, making a backup of the local version, adopting the remote version
		localBak := fmt.Sprintf(configConflictRefPattern, refName, localCommit)
		err = c.repo.CopyRef(localRef, localBak)
		if err != nil {
			return fmt.Errorf("cache: failed to create a backup ref %s: %s", localBak, err)
		}

		err = c.repo.UpdateRef(localRef, remoteCommit)
		if err != nil {
			return fmt.Errorf("cache: failed to update ref %s: %s", refName, err)
		}

		// This sucks, but I am not sure how to do it better
		fmt.Printf("Warning: Changes to your local config (%s) are not based on based on remote config\n", refName)
		fmt.Printf("Warning: and therefore have been discarded, backed up at: %s", localBak)
	}
	return nil
}
