package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/webui"
)

func newWebUICommand() *cobra.Command {
	port := 0
	host := ""
	var features []string

	cmd := &cobra.Command{
		Use:     "webui",
		Aliases: []string{"web"},
		Short:   "Launch the web UI.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWebUI(host, port, features)
		},
	}

	cmd.Flags().StringVarP(&host, "host", "", "localhost", "Host to serve the http server on. Defaults to localhost. Use 0.0.0.0 for access on all interfaces.")
	cmd.Flags().IntVarP(&port, "port", "p", 3333, "Port to serve web UI on")
	cmd.Flags().StringSliceVarP(&features, "features", "f", features, "Features enabled by feature flagging")

	return cmd
}

func runWebUI(host string, port int, features []string) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("unable to get the current working directory: %q", err)
	}

	return webui.Run(currentDir, host, port, features)
}
