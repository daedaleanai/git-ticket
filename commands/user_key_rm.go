package commands

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/identity"
)

func newUserKeyRmCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "rm key_fingerprint [{user_name | user_id}]",
		Short:    "Remove a PGP key from the adopted or the specified user.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUserKeyRm(env, args)
		},
	}

	return cmd
}

func runUserKeyRm(env *Env, args []string) error {
	if len(args) == 0 {
		return errors.New("missing key fingerprint")
	}

	fingerprint := args[0]
	args = args[1:]

	id, args, err := ResolveUser(env.backend, args)
	if err != nil {
		return err
	}

	if len(args) > 0 {
		return fmt.Errorf("unexpected arguments: %s", args)
	}

	var removedKey *identity.Key
	err = id.Mutate(identity.RemoveKeyMutator(fingerprint, &removedKey))
	if err != nil {
		return err
	}

	if removedKey == nil {
		return errors.New("key not found")
	}

	return id.Commit()
}
