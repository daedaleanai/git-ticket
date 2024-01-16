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
		Use:      "approve status [ticket_id]",
		Short:    "Approve a ticket status.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCcbApprove(env, args)
		},
	}

	return cmd
}

func runCcbApprove(env *Env, args []string) error {
	if len(args) < 1 {
		return errors.New("no status supplied")
	}

	status, err := bug.StatusFromString(args[0])
	if err != nil {
		return err
	}

	args = args[1:]

	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	// Perform some checks before approving the status of the ticket:
	//   is the current user an approver of the ticket status?
	//   has the current user already approved the ticket?

	currentUserIdentity, err := env.backend.GetUserIdentity()
	if err != nil {
		return err
	}

	currentUserState := b.Snapshot().GetCcbState(currentUserIdentity.Id(), status)

	if currentUserState == bug.RemovedCcbState {
		return fmt.Errorf("you are not an approver of the ticket status %s", status)
	}
	if currentUserState == bug.ApprovedCcbState {
		fmt.Printf("you have already approved this ticket status %s\n", status)
		return nil
	}

	// Everything looks ok, approve

	_, err = b.CcbApprove(status)
	if err != nil {
		return err
	}

	fmt.Printf("Approving ticket %s\n", b.Id().Human())

	return b.Commit()
}
