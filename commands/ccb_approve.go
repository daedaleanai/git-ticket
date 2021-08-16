package commands

import (
	"errors"
	"fmt"

	"github.com/daedaleanai/git-ticket/bug"
	_select "github.com/daedaleanai/git-ticket/commands/select"
	"github.com/spf13/cobra"
)

func newCcbApproveCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "approve [<id>]",
		Short:    "Approve a ticket.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCcbApprove(env, args)
		},
	}

	return cmd
}

func runCcbApprove(env *Env, args []string) error {

	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	// Perform some checks before approving the CCB of the ticket:
	//   is the current user in the CCB group of the ticket?
	//   has the current user already approved the ticket?

	currentUserIdentity, err := env.backend.GetUserIdentity()

	currentUserState := b.Snapshot().GetCcbState(currentUserIdentity.Id())

	if currentUserState == bug.RemovedCcbState {
		return errors.New("you are not in the ticket CCB group")
	}
	if currentUserState == bug.ApprovedCcbState {
		return errors.New("you have already approved this ticket")
	}

	// Everything looks ok, approve

	_, err = b.CcbApprove()
	if err != nil {
		return err
	}

	fmt.Printf("Approving ticket %s\n", b.Id().Human())

	return b.Commit()
}
