package commands

import (
	"errors"
	"fmt"

	_select "github.com/daedaleanai/git-ticket/commands/select"
	"github.com/spf13/cobra"
)

type pushOptions struct {
	selectedTicket bool
}

func newPushCommand() *cobra.Command {
	env := newEnv()
	options := pushOptions{}

	cmd := &cobra.Command{
		Use:      "push [remote]",
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

func runPush(env *Env, options pushOptions, args []string) error {
	if len(args) > 1 {
		return errors.New("Only pushing to one remote at a time is supported")
	}

	remote := "origin"
	if len(args) == 1 {
		remote = args[0]
	}
	fmt.Println("Pushing to remote", remote)

	if options.selectedTicket {
		bug, _, err := _select.ResolveBug(env.backend, nil)
		if err != nil {
			return err
		}
		fmt.Println("Pushing ticket ", bug.Id())

		out, err := env.backend.PushTicket(remote, bug.Id().String())
		if err != nil {
			return err
		}
		env.out.Print(out)

	} else {
		out, err := env.backend.Push(remote)
		if err != nil {
			return err
		}
		env.out.Print(out)
	}

	return nil
}
