package commands

import (
	"errors"
	"fmt"

	"github.com/daedaleanai/git-ticket/bug"
	_select "github.com/daedaleanai/git-ticket/commands/select"
	"github.com/spf13/cobra"
)

func newCcbAddCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "add user_name/id status [ticket id]",
		Short:    "Add a CCB member as an approver of a ticket status.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCcbAdd(env, args)
		},
	}

	return cmd
}

func runCcbAdd(env *Env, args []string) error {
	if len(args) < 2 {
		return errors.New("no user and/or status supplied")
	}

	userToAddString := args[0]

	status, err := bug.StatusFromString(args[1])
	if err != nil {
		return err
	}

	args = args[2:]

	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	// Perform some checks before adding the user as an approver of the ticket status:
	//   is the user to add a CCB member?
	//   is the user to add already an approver of the ticket status?

	userToAdd, _, err := ResolveUser(env.backend, []string{userToAddString})
	if err != nil {
		return err
	}

	ok, err := bug.IsCcbMember(userToAdd.Identity)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New(userToAdd.DisplayName() + " is not a CCB member")
	}

	if b.Snapshot().GetCcbState(userToAdd.Id(), status) != bug.RemovedCcbState {
		fmt.Printf("%s is already an approver of the ticket status %s\n", userToAdd.DisplayName(), status)
		return nil
	}

	// Everything looks ok, add the user

	_, err = b.CcbAdd(userToAdd, status)
	if err != nil {
		return err
	}

	fmt.Printf("Adding %s as an approver of the ticket %s status %s\n", userToAdd.DisplayName(), b.Id().Human(), status)

	return b.Commit()
}
