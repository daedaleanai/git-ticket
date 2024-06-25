package commands

import (
	"fmt"

	"github.com/daedaleanai/git-ticket/config"
	"github.com/daedaleanai/git-ticket/util/colors"
	"github.com/spf13/cobra"
)

func newCcbListCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "ls",
		Short:    "Lists all current CCB members",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		Args:     cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCcbList(env, args)
		},
	}

	return cmd
}

func runCcbList(env *Env, args []string) error {
	return env.backend.DoWithLockedConfigCache(func(c *config.ConfigCache) error {
		for _, team := range c.CcbConfig {
			env.out.Printf("Team: %s \n",
				colors.WhiteBold(team.Name),
			)
			for _, member := range team.Members {
				user, err := env.backend.ResolveIdentityExcerpt(member.Id)
				if err != nil {
					return err
				}
				if member.Name != user.DisplayName() {
					return fmt.Errorf("Configured user name does not match its id. Expected %q, got %q", user.DisplayName(), member.Name)
				}

				env.out.Printf("\t%s %s\n",
					colors.Cyan(member.Id.Human()),
					user.DisplayName(),
				)
			}
		}

		return nil
	})
}
