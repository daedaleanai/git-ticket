package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/input"
)

type userCreateOptions struct {
	ArmoredKeyFile string
	skipPhabId     bool
	skipGiteaId    bool
	giteaUserName  string
}

func newUserCreateCommand() *cobra.Command {
	env := newEnv()
	options := userCreateOptions{}

	cmd := &cobra.Command{
		Use:      "create",
		Short:    "Create a new identity.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUserCreate(env, options)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVar(&options.ArmoredKeyFile, "key-file", "",
		"Take the armored PGP public key from the given file. Use - to read the message from the standard input",
	)
	flags.BoolVar(&options.skipPhabId, "skip-phab-id", false,
		"Do not attempt to retrieve the users Phabricator ID (note: fetching reviews where they commented will fail if it is not set)")
	flags.BoolVar(&options.skipGiteaId, "skip-gitea-id", false,
		"Do not attempt to retrieve the users Gitea ID (note: fetching reviews where they commented will fail if it is not set)")
	flags.StringVar(&options.giteaUserName, "gitea-username", "",
		"The username of this user in the Gitea server. Must match exactly one user",
	)

	return cmd
}

func runUserCreate(env *Env, opts userCreateOptions) error {
	preName, err := env.backend.GetUserName()
	if err != nil {
		return err
	}

	name, err := input.PromptDefault("Name", "name", preName, input.Required)
	if err != nil {
		return err
	}

	preEmail, err := env.backend.GetUserEmail()
	if err != nil {
		return err
	}

	email, err := input.PromptDefault("Email", "email", preEmail, input.Required)
	if err != nil {
		return err
	}

	avatarURL, err := input.Prompt("Avatar URL", "avatar")
	if err != nil {
		return err
	}

	var key *identity.Key
	if opts.ArmoredKeyFile != "" {
		armoredPubkey, err := input.TextFileInput(opts.ArmoredKeyFile)
		if err != nil {
			return err
		}

		key, err = identity.NewKeyFromArmored(armoredPubkey)
		if err != nil {
			return err
		}

		fmt.Printf("Using key from file `%s`:\n%s\n", opts.ArmoredKeyFile, armoredPubkey)
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

	id, err := env.backend.NewIdentityWithKeyRaw(name, email, "", avatarURL, nil, key, opts.skipPhabId, opts.skipGiteaId, opts.giteaUserName)
	if err != nil {
		return err
	}

	err = id.CommitAsNeeded()
	if err != nil {
		return err
	}

	set, err := env.backend.IsUserIdentitySet()
	if err != nil {
		return err
	}

	if !set {
		err = env.backend.SetUserIdentity(id)
		if err != nil {
			return err
		}
	}

	env.err.Println()
	env.out.Println(id.Id())

	return nil
}
