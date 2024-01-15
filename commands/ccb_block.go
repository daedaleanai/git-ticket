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
		Use:      "block status [ticket id]",
		Short:    "Block a ticket status.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCcbBlock(env, args)
		},
	}

	return cmd
}

func runCcbBlock(env *Env, args []string) error {
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

	// Perform some checks before blocking the status of the ticket:
	//   is the current user an approver of the ticket status?
	//   has the current user already blocked the ticket?

	currentUserIdentity, err := env.backend.GetUserIdentity()
	if err != nil {
		return err
	}

	currentUserState := b.Snapshot().GetCcbState(currentUserIdentity.Id(), status)

	if currentUserState == bug.RemovedCcbState {
		return fmt.Errorf("you are not an approver of the ticket status %s", status)
	}
	if currentUserState == bug.BlockedCcbState {
		fmt.Printf("you have already blocked this ticket status %s\n", status)
		return nil
	}

	// Everything looks ok, block

	_, err = b.CcbBlock(status)
	if err != nil {
		return err
	}

	fmt.Printf("Blocking ticket %s\n", b.Id().Human())

	return b.Commit()
}
