package commands

import (
	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/bug"
	_select "github.com/daedaleanai/git-ticket/commands/select"
)

func newStatusSetCommands() <-chan *cobra.Command {
	env := newEnv()

	cmds := make(chan *cobra.Command)
	go func() {
		for s := bug.FirstStatus; s <= bug.LastStatus; s++ {
			cmd := &cobra.Command{
				Use:     s.String() + " ID",
				Short:   "Ticket is " + s.Action() + ".",
				Args:    cobra.ExactArgs(1),
				PreRunE: loadRepoEnsureUser(env),
				RunE: func(cmd *cobra.Command, args []string) error {
					return runStatusSet(env, args, s)
				},
			}

			// Use the Annotations map to store new status
			cmd.Annotations = make(map[string]string)
			cmd.Annotations["status"] = s.String()

			cmds <- cmd
		}
		close(cmds)
	}()

	return cmds
}

func runStatusSet(env *Env, args []string, s bug.Status) error {
	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	_, err = b.SetStatus(s)
	if err != nil {
		return err
	}

	return b.Commit()
}
