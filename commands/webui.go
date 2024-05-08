package commands

import (
	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/webui"
)

func newWebUICommand() *cobra.Command {
	env := newEnv()
	port := 0

	cmd := &cobra.Command{
		Use:     "webui",
		Aliases: []string{"web"},
		Short:   "Launch the web UI.",
		PreRunE: loadRepo(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWebUI(env, port)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 3333, "Port to serve web UI on")

	return cmd
}

func runWebUI(env *Env, port int) error {
	return webui.Run(env.repo, port)
}
