package commands

import (
	"errors"

	"github.com/spf13/cobra"
)

func newResetCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "reset ticket_id",
		Short:    "Reset a ticket state to discard local changes.",
		Long:     "Discards all local changes to a ticket.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReset(env, args)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	return cmd
}

func runReset(env *Env, args []string) (err error) {
	if len(args) == 0 {
		return errors.New("you must provide a ticket prefix to reset")
	}

	err = env.backend.ResetBug(args[0])

	if err != nil {
		return
	}

	env.out.Printf("ticket %s reset\n", args[0])

	return
}
