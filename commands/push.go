package commands

import (
	"errors"

	"github.com/spf13/cobra"

	_select "github.com/daedaleanai/git-ticket/commands/select"
)

type pushOptions struct {
	selectedTicket bool
}

func newPushCommand() *cobra.Command {
	env := newEnv()
	options := pushOptions{}

	cmd := &cobra.Command{
		Use:      "push [<remote>]",
		Short:    "Push tickets update to a git remote.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(env, options, args)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.BoolVarP(&options.selectedTicket, "selected", "s", false, "Push only the currently selected ticket")

	return cmd
}

func runPush(env *Env, opts pushOptions, args []string) error {
	if len(args) > 1 {
		return errors.New("Only pushing to one remote at a time is supported")
	}

	remote := "origin"
	if len(args) == 1 {
		remote = args[0]
	}

	if opts.selectedTicket {
		bug, _, err := _select.ResolveBug(env.backend, nil)
		if err != nil {
			return err
		}

		err = env.backend.PushTicket(remote, bug.Id().String(), env.out)
		if err != nil {
			return err
		}
	} else {
		err := env.backend.Push(remote, env.out)
		if err != nil {
			return err
		}
	}

	return nil
}
