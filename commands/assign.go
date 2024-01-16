package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	_select "github.com/daedaleanai/git-ticket/commands/select"
)

func newAssignCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "assign [{user_name | user_id}] [ticket_id]",
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
	userToAssign, args, err := ResolveUser(env.backend, args)
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
		if userToAssign.Id() == currentAssignee.Id {
			return fmt.Errorf("ticket already assigned to %s", currentAssignee.Name)
		}
	}

	_, err = b.SetAssignee(userToAssign)
	if err != nil {
		return err
	}

	fmt.Printf("Assigning ticket %s to %s\n", b.Id().Human(), userToAssign.DisplayName())

	return b.Commit()
}
