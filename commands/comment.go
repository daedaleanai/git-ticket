package commands

import (
	termtext "github.com/MichaelMure/go-term-text"
	"github.com/spf13/cobra"

	_select "github.com/daedaleanai/git-ticket/commands/select"
	"github.com/daedaleanai/git-ticket/util/colors"
)

func newCommentCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "comment [<ticket id>]",
		Short:    "Display or add comments to a ticket.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runComment(env, args)
		},
	}

	cmd.AddCommand(newCommentAddCommand())
	cmd.AddCommand(newCommentEditCommand())

	return cmd
}

func runComment(env *Env, args []string) error {
	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	snap := b.Snapshot()

	for i, comment := range snap.Comments {
		if i != 0 {
			env.out.Println()
		}

		env.out.Printf("Author: %s\n", colors.Magenta(comment.Author.DisplayName()))
		env.out.Printf("Id: %s\n", colors.Cyan(comment.Id().Human()))
		env.out.Printf("Date: %s\n\n", comment.FormatTime())
		env.out.Println(termtext.LeftPadLines(comment.Message, 4))
	}

	return nil
}
