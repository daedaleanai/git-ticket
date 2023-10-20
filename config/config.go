package config

import (
	"fmt"
	"io"
	"path"

	"github.com/daedaleanai/git-ticket/repository"
)

const configRefPattern = "refs/configs/"
const configConflictRefPattern = "refs/conflicts/config-%s-%s"
const configRemoteRefPattern = "refs/remotes/%s/configs/"

// Fetch retrieve updates from a remote
// This does not change the local bugs state
func Fetch(repo repository.Repo, remote string) (string, error) {
	remoteRefSpec := fmt.Sprintf(configRemoteRefPattern, remote)
	fetchRefSpec := fmt.Sprintf("%s*:%s*", configRefPattern, remoteRefSpec)

	return repo.FetchRefs(remote, fetchRefSpec)
}

// Push update a remote with all the local changes
func Push(repo repository.Repo, remote string, out io.Writer) error {
	remoteRefSpec := fmt.Sprintf(configRemoteRefPattern, remote)
	localRefs, err := repo.ListRefs(configRefPattern)

	if err != nil {
		return err
	}

	pushed := 0

	for _, localRef := range localRefs {
		hashes, err := repo.CommitsBetween(remoteRefSpec+path.Base(localRef), localRef)
		if err == nil && hashes == nil {
			continue
		}

		fmt.Fprintf(out, "Pushing config: %s\n", path.Base(localRef))
		stdout, err := repo.PushRefs(remote, localRef)
		fmt.Fprintln(out, stdout)
		if err != nil {
			return err
		}

		// Need to update the remote ref manually because push doesn't do it automatically
		// for config references
		err = repo.UpdateRef(remoteRefSpec+path.Base(localRef), repository.Hash(localRef))
		if err != nil {
			return err
		}

		pushed++
	}

	if pushed == 0 {
		fmt.Fprintln(out, "Everything up-to-date")
	} else {
		fmt.Fprintln(out, "Everything sync'd with remote")
	}

	return nil
}

// List configurations stored in git
func ListConfigs(repo repository.ClockedRepo) ([]string, error) {
	refs, err := repo.ListRefs(configRefPattern)
	if err != nil {
		return nil, fmt.Errorf("cache: failed to get a list of config refs: %s", err)
	}

	configs := make([]string, 0)
	for _, ref := range refs {
		configs = append(configs, ref[len(configRefPattern):])
	}

	return configs, nil
}

// Store the configuration data under the given name
func SetConfig(repo repository.ClockedRepo, name string, config []byte) error {
	// Store the blob in the repository
	blobHash, err := repo.StoreData(config)
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
	treeHash, err := repo.StoreTree(tree)
	if err != nil {
		return fmt.Errorf("cache: failed to store the config tree in git: %s", err)
	}

	refName := configRefPattern + name
	exists, err := repo.RefExist(refName)
	if err != nil {
		return fmt.Errorf("cache: failed to determine if ref %s exists: %s", refName, err)
	}

	var commitHash repository.Hash
	if exists {
		oldCommitHash, err := repo.ResolveRef(refName)
		if err != nil {
			return fmt.Errorf("cache: failed to resolve ref %s: %s", refName, err)
		}

		oldTreeHash, err := repo.GetTreeHash(oldCommitHash)
		if err != nil {
			return fmt.Errorf("cache: failed to get the tree for commit %s (ref: %s): %s", oldCommitHash, refName, err)
		}

		// The same tree has as in the previous commit means that nothing has actually changed.
		// This means success
		if oldTreeHash == treeHash {
			return nil
		}

		// Otherwise we reference the old commit as the parent commit
		commitHash, err = repo.StoreCommitWithParent(treeHash, oldCommitHash)
		if err != nil {
			return fmt.Errorf("cache: failed to make a git commit: %s", err)
		}
	} else {
		commitHash, err = repo.StoreCommit(treeHash)
		if err != nil {
			return fmt.Errorf("cache: failed to make a git commit: %s", err)
		}
	}

	err = repo.UpdateRef(refName, commitHash)
	if err != nil {
		return fmt.Errorf("cache: failed to update git ref %s: %s", refName, err)
	}

	return nil
}

// Get the named configuration data
func GetConfig(repo repository.ClockedRepo, name string) ([]byte, error) {
	refName := configRefPattern + name
	commitHash, err := repo.ResolveRef(refName)
	if err != nil {
		return nil, fmt.Errorf("cache: failed to resolve ref %s: %s", refName, err)
	}

	treeHash, err := repo.GetTreeHash(commitHash)
	if err != nil {
		return nil, fmt.Errorf("cache: failed to get the tree for commit %s (ref: %s): %s", commitHash, refName, err)
	}

	tree, err := repo.ReadTree(treeHash)
	if err != nil {
		return nil, fmt.Errorf("cache: failed to list entries in tree %s (ref: %s): %s", treeHash, refName, err)
	}

	for _, entry := range tree {
		if entry.ObjectType != repository.Blob {
			continue
		}

		if entry.Name == "config.json" {
			data, err := repo.ReadData(entry.Hash)
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

// UpdateConfigs fetches config data from the remote and updates the local references.
// If a config has been updated locally and on the server the local one is backed
// up and replaced
func UpdateConfigs(repo repository.ClockedRepo, remote string) (string, error) {
	var out string

	remoteRefSpec := fmt.Sprintf(configRemoteRefPattern, remote)
	remoteRefs, err := repo.ListRefs(remoteRefSpec)

	if err != nil {
		return "", fmt.Errorf("failed to list refs for spec %s: %s", remoteRefSpec, err)
	}

	for _, remoteRef := range remoteRefs {
		refName := remoteRef[len(remoteRefSpec):]
		localRef := configRefPattern + refName

		exist, err := repo.RefExist(localRef)
		if err != nil {
			return out, fmt.Errorf("failed to check if ref exists %s: %s", localRef, err)
		}

		if !exist {
			// New config
			err = repo.CopyRef(remoteRef, localRef)
			if err != nil {
				return out, fmt.Errorf("failed to copy ref %s: %s", remoteRef, err)
			}

			out = out + fmt.Sprintf("%s: new\n", localRef)
			continue
		}

		localCommit, err := repo.ResolveRef(localRef)
		if err != nil {
			return out, fmt.Errorf("failed to resolve local ref %s: %s", localRef, err)
		}
		remoteCommit, err := repo.ResolveRef(remoteRef)
		if err != nil {
			return out, fmt.Errorf("failed to resolve remote ref %s: %s", remoteRef, err)
		}

		// Refs are the same
		if localCommit == remoteCommit {
			continue
		}

		ancestor, err := repo.FindCommonAncestor(localCommit, remoteCommit)
		if err != nil {
			return out, fmt.Errorf("failed to get common ancestor %s %s: %s", localCommit, remoteCommit, err)
		}

		// Local commit is a child of the remote commit
		if ancestor == remoteCommit {
			continue
		}

		// Remote commit is a child of the local commit, local ref can be fast-forwarded to the remote commit
		if ancestor == localCommit {
			err = repo.UpdateRef(localRef, remoteCommit)
			if err != nil {
				return out, fmt.Errorf("failed to update ref %s: %s", localRef, err)
			}

			out = out + fmt.Sprintf("%s: updated to %s\n", localRef, remoteCommit)
		} else {
			// Local and remote configurations diverged, making a backup of the local version, adopting the remote version
			localBak := fmt.Sprintf(configConflictRefPattern, refName, localCommit)
			err = repo.CopyRef(localRef, localBak)
			if err != nil {
				return out, fmt.Errorf("failed to create a backup of ref %s: %s", localRef, err)
			}

			err = repo.UpdateRef(localRef, remoteCommit)
			if err != nil {
				return out, fmt.Errorf("failed to update ref %s: %s", localRef, err)
			}

			out = out + fmt.Sprintf("%s: updated to %s\n", localRef, remoteCommit)

			// This sucks, but I am not sure how to do it better
			out = out + fmt.Sprintf("Warning: Changes to your local config (%s) conflict with the remote config\n", refName)
			out = out + fmt.Sprintf("Warning: and therefore have been discarded, backed up at: %s\n", localBak)
		}
	}

	return out, nil
}
