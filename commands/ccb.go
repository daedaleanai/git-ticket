package commands

import (
	"github.com/spf13/cobra"
)

func newCcbCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ccb",
		Short: "Change Control Board (CCB) actions of a ticket.",
	}

	cmd.AddCommand(newCcbAddCommand())
	cmd.AddCommand(newCcbApproveCommand())
	cmd.AddCommand(newCcbBlockCommand())
	cmd.AddCommand(newCcbRmCommand())

	return cmd
}
