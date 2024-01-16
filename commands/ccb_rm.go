package commands

import (
	"errors"
	"fmt"

	"github.com/daedaleanai/git-ticket/bug"
	_select "github.com/daedaleanai/git-ticket/commands/select"
	"github.com/spf13/cobra"
)

func newCcbRmCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "rm {user_name | user_id} status [ticket_id]",
		Short:    "Remove a CCB member as an approver of a ticket status.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCcbRm(env, args)
		},
	}

	return cmd
}

func runCcbRm(env *Env, args []string) error {
	if len(args) < 2 {
		return errors.New("no user and/or status supplied")
	}

	userToRemoveString := args[0]

	status, err := bug.StatusFromString(args[1])
	if err != nil {
		return err
	}

	args = args[2:]

	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	// Check if the user to remove is an approver of the ticket status

	userToRemove, _, err := ResolveUser(env.backend, []string{userToRemoveString})
	if err != nil {
		return err
	}

	if b.Snapshot().GetCcbState(userToRemove.Id(), status) == bug.RemovedCcbState {
		fmt.Printf("%s is not an approver of the ticket status %s\n", userToRemove.DisplayName(), status)
		return nil
	}

	// Everything looks ok, remove the user

	_, err = b.CcbRm(userToRemove, status)
	if err != nil {
		return err
	}

	fmt.Printf("Removing %s as an approver of the ticket %s status %s\n", userToRemove.DisplayName(), b.Id().Human(), status)

	return b.Commit()
}
