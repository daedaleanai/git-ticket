package commands

import (
	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/webui"
)

func newWebUICommand() *cobra.Command {
	env := newEnv()
	port := 0
	host := ""

	cmd := &cobra.Command{
		Use:     "webui",
		Aliases: []string{"web"},
		Short:   "Launch the web UI.",
		PreRunE: loadRepo(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWebUI(env, host, port)
		},
	}

	cmd.Flags().StringVarP(&host, "host", "", "localhost", "Host to serve the http server on. Defaults to localhost. Use 0.0.0.0 for access on all interfaces.")
	cmd.Flags().IntVarP(&port, "port", "p", 3333, "Port to serve web UI on")

	return cmd
}

func runWebUI(env *Env, host string, port int) error {
	return webui.Run(env.repo, host, port)
}
