package commands

import (
	"github.com/spf13/cobra"
)

func newUserAdoptCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "adopt {user_name | user_id}",
		Short:    "Adopt an existing identity as your own.",
		Args:     cobra.ExactArgs(1),
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUserAdopt(env, args)
		},
	}

	return cmd
}

func runUserAdopt(env *Env, args []string) error {
	i, args, err := ResolveUser(env.backend, args)
	if err != nil {
		return err
	}

	err = env.backend.SetUserIdentity(i)
	if err != nil {
		return err
	}

	env.out.Printf("Your identity is now: %s\n", i.DisplayName())

	return nil
}
