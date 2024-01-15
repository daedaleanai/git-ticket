package commands

import (
	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/bug"
	_select "github.com/daedaleanai/git-ticket/commands/select"
)

func newLabelAddCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "add [ticket id] label...",
		Short:    "Add a label to a ticket.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLabelAdd(env, args)
		},
	}

	return cmd
}

func runLabelAdd(env *Env, args []string) error {
	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		// If ResolveBug failed it may just be because no ticket id was
		// provided because labels are to be added to selected ticket
		if err == bug.ErrBugNotExist {
			b, _, err = _select.ResolveBug(env.backend, nil)
		}
		if err != nil {
			return err
		}
	}

	added := args

	changes, _, err := b.ChangeLabels(added, nil)

	for _, change := range changes {
		env.out.Println(change)
	}

	if err != nil {
		return err
	}

	return b.Commit()
}
