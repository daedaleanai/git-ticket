package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	_select "github.com/daedaleanai/git-ticket/commands/select"
)

func newAssignCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "assign [username/id] [ticket id]",
		Short:    "Assign a user to a ticket.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAssign(env, args)
		},
	}
	return cmd
}

func runAssign(env *Env, args []string) error {
	// TODO allow the user to clear the assignee field
	userToAssignIdentity, args, err := ResolveUser(env.backend, args)
	if err != nil {
		return err
	}

	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	// Check the ticket is not already assigned to the new assignee
	if b.Snapshot().Assignee != nil {
		currentAssignee, err := env.backend.ResolveIdentityExcerpt(b.Snapshot().Assignee.Id())
		if err != nil {
			return err
		}
		if userToAssignIdentity.Id() == currentAssignee.Id {
			return fmt.Errorf("ticket already assigned to %s", currentAssignee.Name)
		}
	}

	_, err = b.SetAssignee(userToAssignIdentity)
	if err != nil {
		return err
	}

	fmt.Printf("Assigning ticket %s to %s\n", b.Id().Human(), userToAssignIdentity.DisplayName())

	return b.Commit()
}
