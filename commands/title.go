package commands

import (
	"github.com/spf13/cobra"

	_select "github.com/daedaleanai/git-ticket/commands/select"
)

func newTitleCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "title [ticket id]",
		Short:    "Display or change a title of a ticket.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTitle(env, args)
		},
	}

	cmd.AddCommand(newTitleEditCommand())

	return cmd
}

func runTitle(env *Env, args []string) error {
	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	snap := b.Snapshot()

	env.out.Println(snap.Title)

	return nil
}
