package commands

import (
	"sort"

	"github.com/daedaleanai/git-ticket/cache"
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
	var users []*cache.IdentityExcerpt

	err := env.backend.DoWithLockedConfigCache(func(c *config.ConfigCache) error {
		for _, id := range c.CcbConfig {
			user, err := env.backend.ResolveIdentityExcerpt(id)
			if err != nil {
				return err
			}
			users = append(users, user)
		}
		return nil
	})
	if err != nil {
		return err
	}

	sort.Sort(byDisplayName(users))

	for _, member := range users {
		env.out.Printf("%s %s\n",
			colors.Cyan(member.Id.Human()),
			member.DisplayName(),
		)
	}

	return nil
}
