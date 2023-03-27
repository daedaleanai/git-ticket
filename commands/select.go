package commands

import (
	"errors"

	"github.com/spf13/cobra"

	_select "github.com/daedaleanai/git-ticket/commands/select"
)

func newSelectCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:   "select <ticket id>",
		Short: "Select a ticket for implicit use in future commands.",
		Example: `git ticket select 2f15
git ticket comment
git ticket status
`,
		Long: `Select a ticket for implicit use in future commands.

This command allows you to omit any ticket ID argument, for example:
  git ticket show
instead of
  git ticket show 2f153ca

The complementary command is "git ticket deselect" performing the opposite operation.
`,
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSelect(env, args)
		},
	}

	return cmd
}

func runSelect(env *Env, args []string) error {
	if len(args) == 0 {
		return errors.New("You must provide a ticket id")
	}

	prefix := args[0]

	b, err := env.backend.ResolveBugPrefix(prefix)
	if err != nil {
		return err
	}

	err = _select.Select(env.backend, b.Id())
	if err != nil {
		return err
	}

	env.out.Printf("selected ticket %s: %s\n", b.Id().Human(), b.Snapshot().Title)

	return nil
}
