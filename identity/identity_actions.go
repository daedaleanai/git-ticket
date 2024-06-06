package identity

import (
	"fmt"
	"path"

	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/repository"
	"github.com/pkg/errors"
)

const Namespace = "identities"

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
