package commands

import (
	"github.com/spf13/cobra"

	_select "github.com/daedaleanai/git-ticket/commands/select"
)

func newStatusCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "status [<ticket id>]",
		Short:    "Display or change a ticket status.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(env, args)
		},
	}

	for setCmd := range newStatusSetCommands() {
		cmd.AddCommand(setCmd)
	}

	return cmd
}

func runStatus(env *Env, args []string) error {
	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	snap := b.Snapshot()

	env.out.Println(snap.Status)

	return nil
}
