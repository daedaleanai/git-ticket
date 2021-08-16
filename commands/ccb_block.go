package commands

import (
	"errors"
	"fmt"

	"github.com/daedaleanai/git-ticket/bug"
	_select "github.com/daedaleanai/git-ticket/commands/select"
	"github.com/spf13/cobra"
)

func newCcbBlockCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "block [<id>]",
		Short:    "Block a ticket.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCcbBlock(env, args)
		},
	}

	return cmd
}

func runCcbBlock(env *Env, args []string) error {

	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	// Perform some checks before blocking the CCB of the ticket:
	//   is the current user in the CCB group of the ticket?
	//   has the current user already blocked the ticket?

	currentUserIdentity, err := env.backend.GetUserIdentity()

	currentUserState := b.Snapshot().GetCcbState(currentUserIdentity.Id())

	if currentUserState == bug.RemovedCcbState {
		return errors.New("you are not in the ticket CCB group")
	}
	if currentUserState == bug.BlockedCcbState {
		return errors.New("you have already blocked this ticket")
	}

	// Everything looks ok, block

	_, err = b.CcbBlock()
	if err != nil {
		return err
	}

	fmt.Printf("Blocking ticket %s\n", b.Id().Human())

	return b.Commit()
}
