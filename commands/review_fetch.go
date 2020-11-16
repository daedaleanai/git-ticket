package commands

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/bug"
	_select "github.com/daedaleanai/git-ticket/commands/select"
)

func newReviewFetchCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:   "fetch DIFF-ID [ID]",
		Short: "Get Differential Revision data from Phabricator and store in a ticket.",
		Long: `fetch stores Phabricator Differential Revision data in a ticket.

The command takes a Phabricator Differential Revision ID (e.g. D1234) and queries the
Phabricator server for any associated comments or status changes, any resulting data
is stored with the selected ticket. Subsequent calls with the same ID will fetch and
store any updates since the previous call. Multiple Revisions can be stored with a
ticket by running the command with different IDs.

`,
		PreRunE: loadRepoEnsureUser(env),
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
	var lastUpdate string
	if existingReview, ok := b.Snapshot().Reviews[diffId]; ok {
		lastUpdate = existingReview.LastTransaction
	}

	review, err := bug.FetchReviewInfo(diffId, lastUpdate)
	if err != nil {
		return err
	}

	if len(review.Comments) == 0 && len(review.Statuses) == 0 {
		fmt.Printf("No updates to save for %s, aborting\n", diffId)
		return nil
	}

	_, err = b.SetReview(review)
	if err != nil {
		return err
	}

	return b.Commit()
}
