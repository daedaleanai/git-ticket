package commands

import (
	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/bug"
	_select "github.com/daedaleanai/git-ticket/commands/select"
)

func newLabelRmCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "rm [<ticket id>] <label>...",
		Short:    "Remove a label from a ticket.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLabelRm(env, args)
		},
	}

	return cmd
}

func runLabelRm(env *Env, args []string) error {
	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		// If ResolveBug failed it may just be because no ticket id was
		// provided because labels are to be removed from selected ticket
		if err == bug.ErrBugNotExist {
			b, _, err = _select.ResolveBug(env.backend, nil)
		}
		if err != nil {
			return err
		}
	}

	removed := args

	changes, _, err := b.ChangeLabels(nil, removed)

	for _, change := range changes {
		env.out.Println(change)
	}

	if err != nil {
		return err
	}

	return b.Commit()
}
