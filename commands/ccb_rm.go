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

func newCcbRmCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "rm <user> [<id>]",
		Short:    "Remove the CCB member from a ticket.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCcbRm(env, args)
		},
	}

	return cmd
}

func runCcbRm(env *Env, args []string) error {
	if len(args) < 1 {
		return errors.New("no user supplied")
	}

	userToRemoveString := args[0]
	args = args[1:]

	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	// Perform some checks before removing the user from the CCB of the ticket:
	//   is the current user a CCB member?
	//   is the user to remove in the CCB group of the ticket?

	currentUserIdentity, err := env.backend.GetUserIdentity()

	ok, err := bug.IsCcbMember(currentUserIdentity.Identity)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("you must be a CCB member to perform this operation")
	}

	// Search through all known users looking for and Id that matches or Name that
	// contains the supplied string

	var userToRemoveId entity.Id

	for _, id := range env.backend.AllIdentityIds() {
		i, err := env.backend.ResolveIdentityExcerpt(id)
		if err != nil {
			return err
		}

		if i.Id.HasPrefix(userToRemoveString) || strings.Contains(i.Name, userToRemoveString) {
			if userToRemoveId != "" {
				// TODO instead of doing this we could allow the user to select from a list
				return fmt.Errorf("multiple users matching %s", userToRemoveString)
			}
			userToRemoveId = i.Id
		}
	}

	if userToRemoveId == "" {
		return fmt.Errorf("no users matching %s", userToRemoveString)
	}

	userToRemoveIdentity, err := env.backend.ResolveIdentity(userToRemoveId)
	if err != nil {
		return err
	}

	if b.Snapshot().GetCcbState(userToRemoveId) == bug.RemovedCcbState {
		return errors.New(userToRemoveIdentity.DisplayName() + " is not in the ticket CCB group")
	}

	// Everything looks ok, add the user

	_, err = b.CcbRm(userToRemoveIdentity)
	if err != nil {
		return err
	}

	fmt.Printf("Removing %s from CCB group of ticket %s\n", userToRemoveIdentity.DisplayName(), b.Id().Human())

	return b.Commit()
}
