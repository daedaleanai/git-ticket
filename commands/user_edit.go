package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/input"
)

type userEditOptions struct {
	skipPhabId    bool
	skipGiteaId   bool
	giteaUserName string
}

func newUserEditCommand() *cobra.Command {
	env := newEnv()
	options := userEditOptions{}

	cmd := &cobra.Command{
		Use:      "edit [{user_name | user_id}]",
		Short:    "Edit a user identity.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		Args:     cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUserEdit(env, options, args)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.BoolVar(&options.skipPhabId, "skipPhabId", false,
		"Do not attempt to retrieve the users Phabricator ID (note: fetching reviews where they commented will fail if it is not set)")
	flags.BoolVar(&options.skipGiteaId, "skipGiteaId", false,
		"Do not attempt to retrieve the users Gitea ID (note: fetching reviews where they commented will fail if it is not set)")
	flags.StringVar(&options.giteaUserName, "gitea-username", "",
		"The username of this user in the Gitea server. Must match exactly one user",
	)

	return cmd
}

func runUserEdit(env *Env, opts userEditOptions, args []string) error {
	id, args, err := ResolveUser(env.backend, args)

	if err != nil {
		return err
	}

	name, err := input.PromptDefault("Name", "name", id.DisplayName(), input.Required)
	if err != nil {
		return err
	}

	email, err := input.PromptDefault("Email", "email", id.Email(), input.Required)
	if err != nil {
		return err
	}

	avatarURL, err := input.PromptDefault("Avatar URL", "avatar", id.AvatarUrl())
	if err != nil {
		return err
	}

	if opts.skipGiteaId && len(opts.giteaUserName) != 0 {
		return fmt.Errorf("Attempted to skip obtaining the gitea user ID, but provided a gitea username")
	}

	if !opts.skipGiteaId && len(opts.giteaUserName) == 0 {
		userName, err := input.Prompt("Gitea Username", "gitea username")
		if err != nil {
			return err
		}
		opts.giteaUserName = userName
	}

	return env.backend.UpdateIdentity(id, name, email, "", avatarURL, opts.skipPhabId, opts.skipGiteaId, opts.giteaUserName)
}
