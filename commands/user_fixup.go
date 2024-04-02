package commands

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/cache"
	"github.com/daedaleanai/git-ticket/input"
)

func newUserFixupCommand() *cobra.Command {
	env := newEnv()
	cmd := &cobra.Command{
		Use:      "fixup",
		Short:    "Walks through all users and attempts to fetch their gitea identities",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		Args:     cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUserFixup(env, args)
		},
	}

	return cmd
}

func isGiteaIdError(err error) bool {
	_, ok := err.(cache.GiteaIdError)
	return ok
}

func fixupIdentity(backend *cache.RepoCache, id *cache.IdentityCache) error {
	if id.GiteaID() != -1 {
		fmt.Println("Skipping identity ", id.Name())
		return nil
	}

	// Attempt to update the identity using the name user name, without prompting the user.
	// This will often be sufficient.
	fmt.Println("Updating identity ", id.Name())
	err := backend.UpdateIdentity(id, id.Name(), id.Email(), "", id.AvatarUrl(), false, false, id.Name())
	if err == nil || !isGiteaIdError(err) {
		return err
	}

	// If we didn't manage to find the gitea id for the user, let's try this time prompting for the user name
	fmt.Println("Could not get Gitea ID for ", id.Name(), ", please insert the Gitea user name manually.")
	userName, err := input.PromptDefault("Gitea Username", "gitea username", id.Name())
	if err != nil {
		return err
	}

	err = backend.UpdateIdentity(id, id.Name(), id.Email(), "", id.AvatarUrl(), false, false, userName)
	if err == nil || !isGiteaIdError(err) {
		return err
	}

	fmt.Println("I still did not find the user. If you know the Gitea user ID can provide it directly")
	giteaIdStr, err := input.PromptDefault("Gitea ID", "gitea id", "-1")
	if err != nil {
		return err
	}

	giteaId, err := strconv.ParseInt(giteaIdStr, 0, 64)
	if err != nil {
		return fmt.Errorf("The Gitea user ID must be a signed integral number")
	}

	err = backend.UpdateIdentityWithGiteaId(id, id.Name(), id.Email(), "", id.AvatarUrl(), false, giteaId)
	if err != nil {
		return fmt.Errorf("Error updating identity %s: %v", id.Name(), err)
	}

	return nil
}

func runUserFixup(env *Env, args []string) error {
	ids := env.backend.AllIdentityIds()
	var users []*cache.IdentityCache
	for _, id := range ids {
		user, err := env.backend.ResolveIdentity(id)
		if err != nil {
			return err
		}
		users = append(users, user)
	}

	for _, id := range users {
		if err := fixupIdentity(env.backend, id); err != nil {
			return err
		}
	}
	return nil
}
