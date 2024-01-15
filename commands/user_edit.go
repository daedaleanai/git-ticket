package commands

import (
	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/input"
)

type userEditOptions struct {
	skipPhabId bool
}

func newUserEditCommand() *cobra.Command {
	env := newEnv()
	options := userEditOptions{}

	cmd := &cobra.Command{
		Use:      "edit [user name/id]",
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

	flags.BoolVarP(&options.skipPhabId, "skipPhabId", "s", false,
		"Do not attempt to retrieve the users Phabricator ID (note: fetching reviews where they commented will fail if it is not set)")

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

	return env.backend.UpdateIdentity(id, name, email, "", avatarURL, opts.skipPhabId)
}
