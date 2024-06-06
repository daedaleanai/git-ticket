package bug

import (
	"fmt"
	"path"

	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/repository"
	"github.com/pkg/errors"
)

const Namespace = "bugs"

// MergeAll will merge all the available remote bug:
//
//   - If the remote has new commit, the local bug is updated to match the same history
//     (fast-forward update)
//   - if the local bug has new commits but the remote don't, nothing is changed
//   - if both local and remote bug have new commits (that is, we have a concurrent edition),
//     new local commits are rewritten at the head of the remote history (that is, a rebase)
func MergeAll(repo repository.ClockedRepo, remote string) <-chan entity.MergeResult {
	out := make(chan entity.MergeResult)

	go func() {
		defer close(out)

		remoteRefSpec := fmt.Sprintf(bugsRemoteRefPattern, remote)
		remoteRefs, err := repo.ListRefs(remoteRefSpec)

		if err != nil {
			out <- entity.MergeResult{Err: err}
			return
		}

		for _, remoteRef := range remoteRefs {
			hashes, err := repo.CommitsBetween(bugsRefPattern+path.Base(remoteRef), remoteRef)
			if err == nil && hashes == nil {
				// If the command succeeded and there are no commits between the remote and local ref then we're
				// up to date. Don't bother with the merge, continue to the next bug. If the command failed then
				// it could be because there is no local ref.
				continue
			}

			id := entity.Id(path.Base(remoteRef))

			if err := id.Validate(); err != nil {
				out <- entity.NewMergeInvalidStatus(id, errors.Wrap(err, "invalid ref").Error())
				continue
			}

			remoteBug, err := readBug(repo, remoteRef)

			if err != nil {
				out <- entity.NewMergeInvalidStatus(id, errors.Wrap(err, "remote bug is not readable").Error())
				continue
			}

			// Check for error in remote data
			if err := remoteBug.Validate(); err != nil {
				out <- entity.NewMergeInvalidStatus(id, errors.Wrap(err, "remote bug is invalid").Error())
				continue
			}

			localRef := bugsRefPattern + remoteBug.Id().String()
			localExist, err := repo.RefExist(localRef)

			if err != nil {
				out <- entity.NewMergeError(err, id)
				continue
			}

			// the bug is not local yet, simply create the reference
			if !localExist {
				err := repo.CopyRef(remoteRef, localRef)

				if err != nil {
					out <- entity.NewMergeError(err, id)
					return
				}

				out <- entity.NewMergeStatus(entity.MergeStatusNew, id, remoteBug)
				continue
			}

			localBug, err := readBug(repo, localRef)

			if err != nil {
				out <- entity.NewMergeError(errors.Wrap(err, "local bug is not readable"), id)
				return
			}

			updated, err := localBug.Merge(repo, remoteBug)

			if err != nil {
				out <- entity.NewMergeInvalidStatus(id, errors.Wrap(err, "merge failed").Error())
				return
			}

			if updated {
				out <- entity.NewMergeStatus(entity.MergeStatusUpdated, id, localBug)
			} else {
				out <- entity.NewMergeStatus(entity.MergeStatusNothing, id, localBug)
			}
		}
	}()

	return out
}
