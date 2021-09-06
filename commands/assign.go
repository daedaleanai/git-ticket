package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	_select "github.com/daedaleanai/git-ticket/commands/select"
	"github.com/daedaleanai/git-ticket/entity"
)

func newAssignCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "assign USER [ID]",
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
	if len(args) < 1 {
		return fmt.Errorf("no user supplied")
	}

	// TODO allow the user to clear the assignee field
	user := args[0]
	args = args[1:]

	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	// Search through all known users looking for an Id that matches or Name that
	// contains the supplied string

	var assigneeId entity.Id

	for _, id := range env.backend.AllIdentityIds() {
		i, err := env.backend.ResolveIdentityExcerpt(id)
		if err != nil {
			return err
		}

		if i.Id.HasPrefix(user) || strings.Contains(i.Name, user) {
			if assigneeId != "" {
				// TODO instead of doing this we could allow the user to select from a list
				return fmt.Errorf("multiple users matching %s", user)
			}
			assigneeId = i.Id
		}
	}

	if assigneeId == "" {
		return fmt.Errorf("no users matching %s", user)
	}

	// Check the ticket is not already assigned to the new assignee
	if b.Snapshot().Assignee != nil {
		currentAssignee, err := env.backend.ResolveIdentityExcerpt(b.Snapshot().Assignee.Id())
		if err != nil {
			return err
		}
		if assigneeId == currentAssignee.Id {
			return fmt.Errorf("ticket already assigned to %s", currentAssignee.Name)
		}
	}

	// Looks good, get the full identitiy and update the ticket
	i, err := env.backend.ResolveIdentity(assigneeId)
	if err != nil {
		return err
	}

	_, err = b.SetAssignee(i)
	if err != nil {
		return err
	}

	fmt.Printf("Assigning ticket %s to %s\n", b.Id().Human(), i.DisplayName())

	return b.Commit()
}
