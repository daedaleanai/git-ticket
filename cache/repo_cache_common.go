package cache

import (
	"fmt"
	"io"
	"time"

	"github.com/daedaleanai/git-ticket/config"
	"github.com/pkg/errors"

	"github.com/daedaleanai/git-ticket/bug"
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/repository"
)

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

// ReadData will attempt to read arbitrary data from the given hash
func (c *RepoCache) ReadData(hash repository.Hash) ([]byte, error) {
	return c.repo.ReadData(hash)
}

// StoreData will store arbitrary data and return the corresponding hash
func (c *RepoCache) StoreData(data []byte) (repository.Hash, error) {
	return c.repo.StoreData(data)
}

// Fetch retrieve updates from a remote
// This does not change the local bugs or identities state
func (c *RepoCache) Fetch(remote string) (string, error) {
	return c.repo.FetchRefs(remote, identity.Namespace, bug.Namespace, config.Namespace)
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

// RefreshResult holds the state of a bug that is new or has been updated
type RefreshResult struct {
	Id   entity.Id
	From time.Time
	To   time.Time
}

// RefreshCache synchronizes the local cache with the bugs and identity commits
func (c *RepoCache) RefreshCache() ([]RefreshResult, error) {
	var results []RefreshResult

	// Bugs. Compare the last edit time of each bug in cache with the last edit time of
	// the bug in the repo. Refresh the cache if it's out of date.
	localBugIds, err := bug.ListLocalIds(c.repo)
	if err != nil {
		return nil, err
	}

	for _, bugId := range localBugIds {
		updateCache := false
		result := RefreshResult{Id: bugId}

		cachedBug, present := c.bugExcerpts[bugId]
		if !present {
			// local bug not in the cache!
			updateCache = true
		} else {
			localBugEditTime, err := bug.PeekLocalBugEditTime(c.repo, bugId)
			if err != nil {
				return nil, err
			}

			if cachedBug.EditTime().Before(localBugEditTime) {
				// local bug has been updated!
				result.From = cachedBug.EditTime()
				result.To = localBugEditTime
				updateCache = true
			}
		}

		if updateCache {
			bug, err := bug.ReadLocalBug(c.repo, bugId)
			if err != nil {
				return nil, err
			}

			snap := bug.Compile()
			c.muBug.Lock()
			c.bugExcerpts[bugId] = NewBugExcerpt(bug, &snap)
			c.muBug.Unlock()

			results = append(results, result)
		}
	}

	// Identities. Nothing clever, just load all the identities again into cache.
	// Note, if at some point this takes too long then the Identity Excerpts need to be
	// updated to include the last edit time, then we could do something clever like with
	// the bugs.
	for i := range identity.ReadAllLocalIdentities(c.repo) {
		if i.Err != nil {
			return nil, i.Err
		}

		c.muIdentity.Lock()
		c.identitiesExcerpts[i.Identity.Id()] = NewIdentityExcerpt(i.Identity)
		c.muIdentity.Unlock()
	}

	err = c.write()

	return results, err
}

// UpdateConfigs will update all the configs from the remote
func (c *RepoCache) UpdateConfigs(remote string) (string, error) {
	return config.UpdateConfigs(c.repo, remote)
}

// Push update a remote with the local changes
func (c *RepoCache) Push(remote string) (string, error) {
	return c.repo.PushRefs(remote, bug.Namespace, identity.Namespace, config.Namespace)
}

// PushTicket updates a remote with the given ticket id
func (c *RepoCache) PushTicket(remote string, id string) (string, error) {
	return c.repo.PushSingleRef(remote, fmt.Sprintf("bugs/%s", id))
}

// Pull will do a Fetch + MergeAll
// This function will return an error if a merge fail
func (c *RepoCache) Pull(remote string, out io.Writer) error {
	fmt.Fprintln(out, "Fetching remote...")
	stdout, err := c.Fetch(remote)
	fmt.Fprintln(out, stdout)
	if err != nil {
		return err
	}

	fmt.Fprintln(out, "Merging data...")
	for merge := range c.MergeAll(remote) {
		if merge.Err != nil {
			return merge.Err
		}
		if merge.Status == entity.MergeStatusInvalid {
			return errors.Errorf("merge failure for ticket %s: %s", merge.Id.Human(), merge.Reason)
		}

		if merge.Status != entity.MergeStatusNothing {
			fmt.Fprintf(out, "%s: %s\n", merge.Id.Human(), merge)
		}
	}

	fmt.Fprintln(out, "Updating configs...")
	stdout, err = c.UpdateConfigs(remote)
	fmt.Fprintln(out, stdout)
	if err != nil {
		return err
	}

	return nil
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

// List configurations stored in git
func (c *RepoCache) ListConfigs() ([]string, error) {
	c.muConfig.RLock()
	defer c.muConfig.RUnlock()

	return config.ListConfigs(c.repo)
}

// Store the configuration data under the given name
func (c *RepoCache) SetConfig(name string, configData []byte) error {
	c.muConfig.Lock()
	defer c.muConfig.Unlock()

	return config.SetConfig(c.repo, name, configData)
}

// Get the named configuration data
func (c *RepoCache) GetConfig(name string) ([]byte, error) {
	c.muConfig.RLock()
	defer c.muConfig.RUnlock()

	return config.GetConfig(c.repo, name)
}
