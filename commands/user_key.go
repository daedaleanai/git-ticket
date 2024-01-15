package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newUserKeyCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "key [user name/id]",
		Short:    "Display, add or remove keys to/from a user.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUserKey(env, args)
		},
	}

	cmd.AddCommand(newUserKeyAddCommand())
	cmd.AddCommand(newUserKeyRmCommand())

	return cmd
}

func runUserKey(env *Env, args []string) error {
	id, args, err := ResolveUser(env.backend, args)
	if err != nil {
		return err
	}

	if len(args) > 0 {
		return fmt.Errorf("unexpected arguments: %s", args)
	}

	for _, key := range id.Keys() {
		fmt.Println(key.Fingerprint())
	}

	return nil
}
