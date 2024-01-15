package commands

import (
	"github.com/spf13/cobra"

	_select "github.com/daedaleanai/git-ticket/commands/select"
	"github.com/daedaleanai/git-ticket/input"
)

type commentEditOptions struct {
	id int
}

func newCommentEditCommand() *cobra.Command {
	env := newEnv()
	options := commentEditOptions{}

	cmd := &cobra.Command{
		Use:      "edit [ticket id]",
		Short:    "Edit a comment on a ticket.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommentEdit(env, options, args)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.IntVarP(&options.id, "id", "i", 0,
		"Which comment to edit, default 0")

	return cmd
}

func runCommentEdit(env *Env, opts commentEditOptions, args []string) error {
	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	op, err := b.Snapshot().GetComment(opts.id)
	if err != nil {
		return err
	}

	newMessage, err := input.BugCommentEditorInput(env.backend, op.Message)
	if err == input.ErrEmptyMessage {
		env.err.Println("Empty message, aborting.")
		return nil
	}
	if err != nil {
		return err
	}
	if newMessage == op.Message {
		env.err.Println("Unchanged comment, aborting.")
		return nil
	}

	_, err = b.EditComment(op.Id(), newMessage)
	if err != nil {
		return err
	}

	return b.Commit()
}
