package commands

import (
	"errors"
	"fmt"
	"github.com/daedaleanai/git-ticket/bug/review"
	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/bug"
	_select "github.com/daedaleanai/git-ticket/commands/select"
)

func newReviewFetchCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:   "fetch <revision id or pull request ref> [<ticket id>]",
		Short: "Get Differential Revision data from Phabricator or Gitea and store in a ticket.",
		Long: `fetch stores Phabricator Differential Revision data in a ticket.

The command takes a Phabricator Differential Revision ID (e.g. D1234) or Gitea Pull Request (e.g. daedalean-github/git-ticket#9) and queries the 
server for any associated comments or status changes, any resulting data
is stored with the selected ticket. Subsequent calls with the same ID will fetch and
store any updates since the previous call. Multiple Revisions can be stored with a
ticket by running the command with different IDs.

`,
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReviewFetch(env, args)
		},
	}

	return cmd
}

func runReviewFetch(env *Env, args []string) error {
	if len(args) < 1 {
		return errors.New("no DiffID supplied")
	}

	diffId := args[0]
	args = args[1:]

	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	// If we already have review data for this Differential then just get any updates
	// since then
	var lastUpdate review.PullRequest
	if existingReview, ok := b.Snapshot().Reviews[diffId]; ok {
		lastUpdate = existingReview
	}

	review, err := bug.FetchReviewInfo(diffId, lastUpdate)
	if err != nil {
		return fmt.Errorf("failed to fetch review info: %s", err)
	}

	if review.IsEmpty() {
		fmt.Printf("No updates to save for %s, aborting\n", diffId)
		return nil
	}

	_, err = b.SetReview(review)
	if err != nil {
		return fmt.Errorf("failed to store review info: %s", err)
	}

	return b.Commit()
}
