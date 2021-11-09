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
		Use:      "rm <user> <status> [<id>]",
		Short:    "Remove a CCB member from a ticket status.",
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

	// Perform some checks before removing the user from the CCB of the ticket:
	//   is the current user a CCB member?
	//   is the user to remove in the CCB group of the ticket status?

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

	if b.Snapshot().GetCcbState(userToRemoveId, status) == bug.RemovedCcbState {
		fmt.Printf("%s is not in the ticket CCB group\n", userToRemoveIdentity.DisplayName())
		return nil
	}

	// Everything looks ok, remove the user

	_, err = b.CcbRm(userToRemoveIdentity, status)
	if err != nil {
		return err
	}

	fmt.Printf("Removing %s from %s CCB group of ticket %s\n", userToRemoveIdentity.DisplayName(), status, b.Id().Human())

	return b.Commit()
}
