package commands

import (
	"errors"
	"fmt"

	_select "github.com/daedaleanai/git-ticket/commands/select"
	"github.com/spf13/cobra"
)

func newReviewClearCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "clear <DiffID> [<id>]",
		Short:    "Clear the Differential Revision data associated with a ticket.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReviewClear(env, args)
		},
	}

	return cmd
}

func runReviewClear(env *Env, args []string) error {
	if len(args) < 1 {
		return errors.New("no DiffID supplied")
	}

	diffId := args[0]
	args = args[1:]

	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	if _, ok := b.Snapshot().Reviews[diffId]; !ok {
		return fmt.Errorf("ticket %s does not have a review %s", b.Id().Human(), diffId)
	}

	_, err = b.RmReview(diffId)
	if err != nil {
		return err
	}

	return b.Commit()
}
