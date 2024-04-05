package commands

import (
	"errors"
	"fmt"

	"github.com/daedaleanai/git-ticket/bug"
	_select "github.com/daedaleanai/git-ticket/commands/select"
	"github.com/spf13/cobra"
)

func newCcbApproveCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "approve status [ticket_id]",
		Short:    "Approve a ticket status.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCcbApprove(env, args)
		},
	}

	return cmd
}

func runCcbApprove(env *Env, args []string) error {
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

	// Perform some checks before approving the status of the ticket:
	//   is the current user an approver of the ticket status?
	//   has the current user already approved the ticket status?
	//   is the ticket in a status that can transition to the given ticket status?

	currentUserIdentity, err := env.backend.GetUserIdentity()
	if err != nil {
		return err
	}

	currentUserState := b.Snapshot().GetCcbState(currentUserIdentity.Id(), status)

	if currentUserState == bug.RemovedCcbState {
		return fmt.Errorf("you are not an approver of the ticket status %s", status)
	}
	if currentUserState == bug.ApprovedCcbState {
		fmt.Printf("you have already approved this ticket status %s\n", status)
		return nil
	}

	nextStatusMatchesRequestedApproval := func(currentStatus, nextStatus bug.Status, workflow *bug.Workflow) bool {
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

	if !nextStatusMatchesRequestedApproval(b.Snapshot().Status, status, workflow) {
		// Prevent accidental approval of states, when the ticket is not in a state that transitions to the
		// requested state
		return fmt.Errorf("Requested CCB approval for ticket status %s, but ticket is in status %s, which cannot directly transition to %s", status, b.Snapshot().Status, status)
	}

	// Everything looks ok, approve

	_, err = b.CcbApprove(status)
	if err != nil {
		return err
	}

	fmt.Printf("Approving ticket %s\n", b.Id().Human())

	// Attempt to transition to the approved state, but warn if there are more approvals required.
	if err := bug.ValidateCcb(b.Snapshot(), status); err == nil {
		_, err = b.SetStatus(status)
		if err != nil {
			return err
		}
		fmt.Printf("Set status to %s\n", status)
	} else {
		fmt.Printf("Warning: did not transition the ticket to %s: %v\n", status, err)
	}

	return b.Commit()
}
