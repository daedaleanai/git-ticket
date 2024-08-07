package cache

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/pkg/errors"
	"github.com/thought-machine/gonduit/requests"

	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/repository"
)

const identityCacheFile = "identity-cache"

func identityCacheFilePath(repo repository.Repo) string {
	return path.Join(repo.GetPath(), "git-bug", identityCacheFile)
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

	if aux.Version != formatVersion {
		return fmt.Errorf("unknown cache format version %v", aux.Version)
	}

	c.identitiesExcerpts = aux.Excerpts
	return nil
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
func (c *RepoCache) ResolveIdentityPhabID(phabID string) (identity.Interface, error) {
	user, err := c.ResolveIdentityMatcher(func(excerpt *IdentityExcerpt) bool {
		return excerpt.PhabID == phabID
	})
	if err != nil {
		return nil, err
	}
	return user.Identity, nil
}

// ResolveIdentityGiteaID retrieve an Identity matching a Gitea ID.
// It fails if multiple identities match.
func (c *RepoCache) ResolveIdentityGiteaID(giteaID int64) (identity.Interface, error) {
	user, err := c.ResolveIdentityMatcher(func(excerpt *IdentityExcerpt) bool {
		return excerpt.GiteaID == giteaID
	})
	if err != nil {
		return nil, err
	}
	return user.Identity, nil
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

func (c *RepoCache) AllIdentityExcerpts() []*IdentityExcerpt {
	var vals []*IdentityExcerpt
	for _, v := range c.identitiesExcerpts {
		vals = append(vals, v)
	}

	return vals
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
func (c *RepoCache) NewIdentity(name string, email string, skipPhabId, skipGiteaId bool, giteaUserName string) (*IdentityCache, error) {
	return c.NewIdentityRaw(name, email, "", "", nil, skipPhabId, skipGiteaId, giteaUserName)
}

// NewIdentityFull create a new identity
// The new identity is written in the repository (commit)
func (c *RepoCache) NewIdentityFull(name string, email string, login string, avatarUrl string, skipPhabId, skipGiteaId bool, giteaUserName string) (*IdentityCache, error) {
	return c.NewIdentityRaw(name, email, login, avatarUrl, nil, skipPhabId, skipGiteaId, giteaUserName)
}

func (c *RepoCache) NewIdentityRaw(name string, email string, login string, avatarUrl string, metadata map[string]string, skipPhabId, skipGiteaId bool, giteaUserName string) (*IdentityCache, error) {
	return c.NewIdentityWithKeyRaw(name, email, login, avatarUrl, metadata, nil, skipPhabId, skipGiteaId, giteaUserName)
}

func (c *RepoCache) NewIdentityWithKeyRaw(name string, email string, login string, avatarUrl string, metadata map[string]string, key *identity.Key, skipPhabId, skipGiteaId bool, giteaUserName string) (*IdentityCache, error) {

	var phabId string
	if skipPhabId == false {
		var err error
		// attempt to populate the phabricator ID
		phabId, err = c.getPhabId(email)
		if err != nil {
			return nil, errors.Wrap(err, "failed to retrieve users Phabricator ID")
		}
	}

	var giteaID int64 = -1
	if skipGiteaId == false {
		var err error
		// attempt to populate the gitea ID
		giteaID, err = c.getGiteaId(giteaUserName)
		if err != nil {
			return nil, errors.Wrap(err, "failed to retrieve users Gitea ID")
		}
	}

	i := identity.NewIdentityFull(name, email, login, avatarUrl, phabId, giteaID, key)
	return c.finishIdentity(i, metadata)
}

// UpdateIdentityWithGiteaId updates an existing identity in the repository and cache using the provided giteaID
func (c *RepoCache) UpdateIdentityWithGiteaId(i *IdentityCache, name string, email string, login string, avatarUrl string, skipPhabId bool, giteaID int64) error {

	var phabId string
	if skipPhabId == false {
		var err error
		// attempt to populate the phabricator ID
		phabId, err = c.getPhabId(email)
		if err != nil {
			return errors.Wrap(err, "failed to retrieve users Phabricator ID")
		}
	}

	err := i.Mutate(func(mutator identity.Mutator) identity.Mutator {
		mutator.Name = name
		mutator.Email = email
		mutator.AvatarUrl = avatarUrl
		mutator.PhabID = phabId
		mutator.GiteaID = giteaID
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

// UpdateIdentity updates an existing identity in the repository and cache
func (c *RepoCache) UpdateIdentity(i *IdentityCache, name string, email string, login string, avatarUrl string, skipPhabId, skipGiteaId bool, giteaUserName string) error {
	var giteaID int64 = -1
	if skipGiteaId == false {
		var err error
		// attempt to populate the gitea ID
		giteaID, err = c.getGiteaId(giteaUserName)
		if err != nil {
			return err
		}
	}

	return c.UpdateIdentityWithGiteaId(i, name, email, login, avatarUrl, skipGiteaId, giteaID)
}

func (c *RepoCache) getPhabId(email string) (string, error) {
	// Assuming that the e-mail prefix is username on Phabricator
	user := email[0:strings.Index(email, "@")]

	phabClient, err := repository.GetPhabClient()
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

type GiteaIdErrorType uint

const (
	GiteaIdNotFound GiteaIdErrorType = iota
	GiteaIdMultipleMatches
)

type GiteaIdError struct {
	error
	GiteaIdErrorType GiteaIdErrorType
}

func newGiteaIdError(errorType GiteaIdErrorType, err error) GiteaIdError {
	return GiteaIdError{err, errorType}
}

func (c *RepoCache) getGiteaId(userName string) (int64, error) {
	giteaClient, err := repository.GetGiteaClient()
	if err != nil {
		return -1, err
	}

	users, _, err := giteaClient.SearchUsers(gitea.SearchUsersOption{
		KeyWord: userName,
	})
	if err != nil {
		return -1, err
	}

	if len(users) == 0 {
		return -1, newGiteaIdError(GiteaIdNotFound, fmt.Errorf("no Gitea users matching %s", userName))
	}
	if len(users) > 1 {
		for _, user := range users {
			fmt.Println("matched user", user.FullName, "id", user.ID)
		}
		return -1, newGiteaIdError(GiteaIdMultipleMatches, fmt.Errorf("too many Gitea users matching %s", userName))
	}

	return users[0].ID, nil
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
