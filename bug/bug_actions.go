package bug

import (
	"fmt"
	"io"
	"path"

	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/repository"
	"github.com/pkg/errors"
)

// Fetch retrieve updates from a remote
// This does not change the local bugs state
func Fetch(repo repository.Repo, remote string) (string, error) {
	remoteRefSpec := fmt.Sprintf(bugsRemoteRefPattern, remote)
	fetchRefSpec := fmt.Sprintf("%s*:%s*", bugsRefPattern, remoteRefSpec)

	return repo.FetchRefs(remote, fetchRefSpec)
}

// Push update a remote with all the local changes
func Push(repo repository.Repo, remote string, out io.Writer) error {
	remoteRefSpec := fmt.Sprintf(bugsRemoteRefPattern, remote)
	localRefs, err := repo.ListRefs(bugsRefPattern)

	if err != nil {
		return err
	}

	refSpecs := []string{}
	for _, localRef := range localRefs {
		hashes, err := repo.CommitsBetween(remoteRefSpec+path.Base(localRef), localRef)
		if err == nil && hashes == nil {
			continue
		}

		refSpecs = append(refSpecs, localRef)
	}

	if len(refSpecs) > 0 {
		fmt.Fprintf(out, "Pushing tickets:\n")
		for _, localRef := range refSpecs {
			fmt.Fprintf(out, "%s\n", localRef)
		}

		stdout, err := repo.PushAllRefs(remote, refSpecs)
		fmt.Fprintln(out, stdout)
		if err != nil {
			return err
		}

		fmt.Fprintln(out, "Updating local references")
		for _, localRef := range refSpecs {
			// Need to update the remote ref manually because push doesn't do it automatically
			// for bug references
			err = repo.UpdateRef(remoteRefSpec+path.Base(localRef), repository.Hash(localRef))
			if err != nil {
				return err
			}
		}
		fmt.Fprintln(out, "Everything sync'd with remote")
	} else {
		fmt.Fprintln(out, "Everything up-to-date")
	}

	return nil
}

// PushRef update a remote with a local change
func PushRef(repo repository.Repo, remote string, ref string, out io.Writer) error {
	fmt.Fprintf(out, "Pushing selected ticket: %s\n", ref)
	stdout, err := repo.PushRefs(remote, bugsRefPattern+ref)
	fmt.Fprintln(out, stdout)
	if err != nil {
		return err
	}

	remoteRefSpec := fmt.Sprintf(bugsRemoteRefPattern, remote)
	// Need to update the remote ref manually because push doesn't do it automatically
	// for bug references
	err = repo.UpdateRef(remoteRefSpec+ref, repository.Hash(bugsRefPattern+ref))
	if err != nil {
		return err
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
			return errors.Errorf("merge failure for ticket %s: %s", merge.Id.Human(), merge.Reason)
		}
	}

	return nil
}

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
