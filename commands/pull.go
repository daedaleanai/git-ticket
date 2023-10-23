package commands

import (
	"errors"

	"github.com/spf13/cobra"
)

func newPullCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "pull [<remote>]",
		Short:    "Pull tickets update from a git remote.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPull(env, args)
		},
	}

	return cmd
}

func runPull(env *Env, args []string) error {
	if len(args) > 1 {
		return errors.New("Only pulling from one remote at a time is supported")
	}

	remote := "origin"
	if len(args) == 1 {
		remote = args[0]
	}

	err := env.backend.Pull(remote, env.out)
	if err != nil {
		return err
	}

	return nil
}
