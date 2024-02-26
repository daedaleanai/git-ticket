package commands

import "github.com/spf13/cobra"

func newChecklistCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checklist",
		Short: "List available checklists and their contents.",
	}

	cmd.AddCommand(newChecklistListCommand())
	cmd.AddCommand(newChecklistShowCommand())

	return cmd
}
