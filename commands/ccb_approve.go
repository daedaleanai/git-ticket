package commands

import (
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
	//   is the current user in the CCB group of the ticket status?
	//   has the current user already approved the ticket?

	currentUserIdentity, err := env.backend.GetUserIdentity()

	nextStates, err := b.Snapshot().NextStates()
	if err != nil {
		return err
	}

	for _, s := range nextStates {
		currentUserState := b.Snapshot().GetCcbState(currentUserIdentity.Id(), s)

		if currentUserState == bug.RemovedCcbState {
			return fmt.Errorf("you are not in the ticket %s CCB group", s)
		}
		if currentUserState == bug.ApprovedCcbState {
			fmt.Println("you have already approved this ticket")
			return nil
		}

		// Everything looks ok, approve

		_, err = b.CcbApprove(s)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Approving ticket %s\n", b.Id().Human())

	return b.Commit()
}
