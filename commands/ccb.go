package commands

import (
	"github.com/spf13/cobra"
)

func newCcbCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ccb",
		Short: "Change Control Board (CCB) actions of a ticket.",
		Long: `Change Control Board (CCB) allows the transition of tickets through the workflow to be monitored.

CCB members, as defined in the "ccb" config, can be added as approvers to the status of a ticket, meaning they must approve the ticket before it can be moved to that status.
`,
	}

	cmd.AddCommand(newCcbAddCommand())
	cmd.AddCommand(newCcbApproveCommand())
	cmd.AddCommand(newCcbBlockCommand())
	cmd.AddCommand(newCcbRmCommand())
	cmd.AddCommand(newCcbListCommand())

	return cmd
}
