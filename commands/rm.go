package commands

import (
	"errors"

	"github.com/spf13/cobra"
)

func newRmCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "rm ticket_id",
		Short:    "Remove an existing ticket.",
		Long:     "Remove an existing ticket in the local repository.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRm(env, args)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	return cmd
}

func runRm(env *Env, args []string) (err error) {
	if len(args) == 0 {
		return errors.New("you must provide a ticket prefix to remove")
	}

	err = env.backend.RemoveBug(args[0])

	if err != nil {
		return
	}

	env.out.Printf("ticket %s removed\n", args[0])

	return
}
