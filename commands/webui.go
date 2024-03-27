package commands

import (
	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/webui"
)

func newWebUICommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "webui",
		Aliases:  []string{"web"},
		Short:    "Launch the web UI.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWebUI(env)
		},
	}

	return cmd
}

func runWebUI(env *Env) error {
	return webui.Run(env.backend)
}
