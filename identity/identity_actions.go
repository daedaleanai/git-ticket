package identity

import (
	"fmt"
	"io"
	"path"

	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/repository"
	"github.com/pkg/errors"
)

// Fetch retrieve updates from a remote
// This does not change the local identities state
func Fetch(repo repository.Repo, remote string) (string, error) {
	remoteRefSpec := fmt.Sprintf(identityRemoteRefPattern, remote)
	fetchRefSpec := fmt.Sprintf("%s*:%s*", identityRefPattern, remoteRefSpec)

	return repo.FetchRefs(remote, fetchRefSpec)
}

// Push update a remote with the local changes
func Push(repo repository.Repo, remote string, out io.Writer) error {
	remoteRefSpec := fmt.Sprintf(identityRemoteRefPattern, remote)
	localRefs, err := repo.ListRefs(identityRefPattern)

	if err != nil {
		return err
	}

	pushed := 0

	for _, localRef := range localRefs {
		hashes, err := repo.CommitsBetween(remoteRefSpec+path.Base(localRef), localRef)
		if err == nil && hashes == nil {
			continue
		}

		fmt.Fprintf(out, "Pushing identity: %s\n", path.Base(localRef))
		stdout, err := repo.PushRefs(remote, localRef)
		fmt.Fprintln(out, stdout)
		if err != nil {
			return err
		}

		// Need to update the remote ref manually because push doesn't do it automatically
		// for identity references
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

// Pull will do a Fetch + MergeAll
// This function will return an error if a merge fail
func Pull(repo repository.ClockedRepo, remote string) error {
	_, err := Fetch(repo, remote)
	if err != nil {
		return err
	}

	for merge := range MergeAll(repo, remote) {
		if merge.Err != nil {
			return merge.Err
		}
		if merge.Status == entity.MergeStatusInvalid {
			return errors.Errorf("merge failure: %s", merge.Reason)
		}
	}

	return nil
}

// MergeAll will merge all the available remote identity
func MergeAll(repo repository.ClockedRepo, remote string) <-chan entity.MergeResult {
	out := make(chan entity.MergeResult)

	go func() {
		defer close(out)

		remoteRefSpec := fmt.Sprintf(identityRemoteRefPattern, remote)
		remoteRefs, err := repo.ListRefs(remoteRefSpec)

		if err != nil {
			out <- entity.MergeResult{Err: err}
			return
		}

		for _, remoteRef := range remoteRefs {
			hashes, err := repo.CommitsBetween(identityRefPattern+path.Base(remoteRef), remoteRef)
			if err == nil && hashes == nil {
				// If the command succeeded and there are no commits between the remote and local ref then we're
				// up to date. Don't bother with the merge, continue to the next identity. If the command failed then
				// it could be because there is no local ref.
				continue
			}

			id := entity.Id(path.Base(remoteRef))

			if err := id.Validate(); err != nil {
				out <- entity.NewMergeInvalidStatus(id, errors.Wrap(err, "invalid ref").Error())
				continue
			}

			remoteIdentity, err := read(repo, remoteRef)

			if err != nil {
				out <- entity.NewMergeInvalidStatus(id, errors.Wrap(err, "remote identity is not readable").Error())
				continue
			}

			// Check for error in remote data
			if err := remoteIdentity.Validate(); err != nil {
				out <- entity.NewMergeInvalidStatus(id, errors.Wrap(err, "remote identity is invalid").Error())
				continue
			}

			localRef := identityRefPattern + remoteIdentity.Id().String()
			localExist, err := repo.RefExist(localRef)

			if err != nil {
				out <- entity.NewMergeError(err, id)
				continue
			}

			// the identity is not local yet, simply create the reference
			if !localExist {
				err := repo.CopyRef(remoteRef, localRef)

				if err != nil {
					out <- entity.NewMergeError(err, id)
					return
				}

				out <- entity.NewMergeStatus(entity.MergeStatusNew, id, remoteIdentity)
				continue
			}

			localIdentity, err := read(repo, localRef)

			if err != nil {
				out <- entity.NewMergeError(errors.Wrap(err, "local identity is not readable"), id)
				return
			}

			updated, err := localIdentity.Merge(repo, remoteIdentity)

			if err != nil {
				out <- entity.NewMergeInvalidStatus(id, errors.Wrap(err, "merge failed").Error())
				return
			}

			if updated {
				out <- entity.NewMergeStatus(entity.MergeStatusUpdated, id, localIdentity)
			} else {
				out <- entity.NewMergeStatus(entity.MergeStatusNothing, id, localIdentity)
			}
		}
	}()

	return out
}
