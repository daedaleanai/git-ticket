package commands

import (
	"github.com/spf13/cobra"
)

func newRefreshCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "refresh",
		Short:    "Refresh cache.",
		Hidden:   true, // not needed client side, so hide from command list.
		Long:     `Refreshes the local cache by comparing the state of each ticket with the cached version and updating as necessary. Useful to run on a remote after it has been pushed to.`,
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRefresh(env, args)
		},
	}

	return cmd
}

func runRefresh(env *Env, args []string) error {
	results, err := env.backend.RefreshCache()

	if err != nil {
		return err
	}
	if results == nil {
		return nil
	}

	for _, r := range results {
		if r.From.IsZero() {
			env.out.Printf("%s,NEW\n", r.Id)
		} else {
			env.out.Printf("%s,UPDATE,%s,%s\n", r.Id, r.From.Format("2006-01-02T15:04:05"), r.To.Format("2006-01-02T15:04:05"))
		}
	}

	return nil
}
