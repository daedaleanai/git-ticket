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
		Use:      "block status [ticket_id]",
		Short:    "Block a ticket status.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCcbBlock(env, args)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&forceCcbChange, "force", "f", false, "Forces the CCB operation, even if the ticket is not in a state that can directly transition to the blocked status. With great power comes great responsibility")

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
	//   has the current user already blocked the ticket status?
	//   is the ticket in a status that can transition to the given ticket status?

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

	nextStatusMatchesRequestedBlock := func(currentStatus, nextStatus bug.Status, workflow *bug.Workflow) bool {
		for _, status := range workflow.NextStatuses(currentStatus) {
			if status == nextStatus {
				return true
			}
		}
		return false
	}

	workflow := bug.FindWorkflow(b.Snapshot().Labels)
	if workflow == nil {
		return fmt.Errorf("Could not find associated workflow for ticket %v", b.Id())
	}

	if !nextStatusMatchesRequestedBlock(b.Snapshot().Status, status, workflow) && !forceCcbChange {
		// Prevent accidental block of states, when the ticket is not in a state that transitions to the
		// requested state
		return fmt.Errorf("Requested CCB block for ticket status %s, but ticket is in status %s, which cannot directly transition to %s", status, b.Snapshot().Status, status)
	}

	// Everything looks ok, block

	_, err = b.CcbBlock(status)
	if err != nil {
		return err
	}

	fmt.Printf("Blocking ticket %s\n", b.Id().Human())

	return b.Commit()
}
