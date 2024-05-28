package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/bug"
	_select "github.com/daedaleanai/git-ticket/commands/select"
)

var allowDeprecatedLabels bool
var createLabels bool

func newLabelAddCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "add [ticket_id] label...",
		Short:    "Add a label to a ticket.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLabelAdd(env, args)
		},
	}

	cmd.Flags().BoolVar(&allowDeprecatedLabels, "allow-deprecated", false, "When given, deprecated labels can be added to a ticket")
	cmd.Flags().BoolVar(&createLabels, "create", false, "When given, the flags are first created (added to the git-ticket configuration), then added to the ticket.")
	return cmd
}

func runLabelAdd(env *Env, args []string) error {
	labels := args

	if createLabels {
		// add labels to the configuration first.
		for _, label := range labels {
			err := bug.AppendLabelToConfiguration(bug.Label(label))
			if err != nil {
				return err
			}
		}

		// save configuration persistently
		labelStoreKey, labelStoreValue, err := bug.LabelStoreData()
		if err != nil {
			return fmt.Errorf("Unable to obtain label store data: %s", err)
		}

		err = env.backend.SetConfig(labelStoreKey, labelStoreValue)
		if err != nil {
			return fmt.Errorf("Unable to store label configuration persistently: %s", err)
		}
	}

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

	added := labels

	changes, _, err := b.ChangeLabels(added, nil, allowDeprecatedLabels)

	for _, change := range changes {
		env.out.Println(change)
	}

	if err != nil {
		return err
	}

	return b.Commit()
}
