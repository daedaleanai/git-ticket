package commands

import (
	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/input"
)

func newUserEditCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "edit [<username/id>]",
		Short:    "Edit a user identity.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		Args:     cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUserEdit(env, args)
		},
	}

	return cmd
}

func runUserEdit(env *Env, args []string) error {
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

	return env.backend.UpdateIdentity(id, name, email, "", avatarURL)
}
