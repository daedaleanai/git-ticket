package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/daedaleanai/git-ticket/bug"
	_select "github.com/daedaleanai/git-ticket/commands/select"
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/spf13/cobra"
)

func newCcbAddCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "add <user> <status> [<id>]",
		Short:    "Add a CCB member to a ticket status.",
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

	// Perform some checks before adding the user to the CCB of the ticket:
	//   is the current user a CCB member?
	//   is the user to add a CCB member?
	//   is the user to add already in the CCB group of the ticket status?

	currentUserIdentity, err := env.backend.GetUserIdentity()

	ok, err := bug.IsCcbMember(currentUserIdentity.Identity)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("you must be a CCB member to perform this operation")
	}

	// Search through all known users looking for an Id that matches or Name that
	// contains the supplied string

	var userToAddId entity.Id

	for _, id := range env.backend.AllIdentityIds() {
		i, err := env.backend.ResolveIdentityExcerpt(id)
		if err != nil {
			return err
		}

		if i.Id.HasPrefix(userToAddString) || strings.Contains(i.Name, userToAddString) {
			if userToAddId != "" {
				// TODO instead of doing this we could allow the user to select from a list
				return fmt.Errorf("multiple users matching %s", userToAddString)
			}
			userToAddId = i.Id
		}
	}

	if userToAddId == "" {
		return fmt.Errorf("no users matching %s", userToAddString)
	}

	userToAddIdentity, err := env.backend.ResolveIdentity(userToAddId)
	if err != nil {
		return err
	}

	ok, err = bug.IsCcbMember(userToAddIdentity.Identity)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New(userToAddIdentity.DisplayName() + " is not a CCB member")
	}

	if b.Snapshot().GetCcbState(userToAddId, status) != bug.RemovedCcbState {
		fmt.Printf("%s is already in the ticket %s CCB group\n", userToAddIdentity.DisplayName(), status)
		return nil
	}

	// Everything looks ok, add the user

	_, err = b.CcbAdd(userToAddIdentity, status)
	if err != nil {
		return err
	}

	fmt.Printf("Adding %s to %s CCB group of ticket %s\n", userToAddIdentity.DisplayName(), status, b.Id().Human())

	return b.Commit()
}
