package commands

import (
	"fmt"

	"github.com/daedaleanai/git-ticket/cache"
	"github.com/daedaleanai/git-ticket/web"
	"github.com/spf13/cobra"
)

type webOptions struct {
	host string
	port uint16
}

func newWebCommand() *cobra.Command {
	options := webOptions{
		host: "localhost",
		port: 8080,
	}

	cmd := &cobra.Command{
		Use:   "web",
		Short: "Run web frontend.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO(ie): Use Gitea OAuth2 provider to verify that the user has access
			//           to the repo.
			// For now we simply guard access to localhost.
			// If someone took over your localhost, they would also likely
			// have access to the git repo on your disk.
			//
			// NOTE: There is deliberately no magic flag to make it `0.0.0.0:$port`.
			// If you choose this unwise path, change the code.
			web.MakeRouter(&web.WebEnvFactory{
				MakeEnv: func() (*cache.RepoCache, error) {
					env := newEnv()
					return env.backend, loadBackend(env)(nil, nil)
				},
				ReleaseEnv: func(backend *cache.RepoCache) {
					backend.Close()
				},
			}).Run(fmt.Sprintf("%s:%d", options.host, options.port))
			return nil
		},
	}

	flags := cmd.Flags()
	flags.Uint16VarP(&options.port, "port", "p", 8080, "Web server port")

	return cmd
}
