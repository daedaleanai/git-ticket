package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/cache"
	"github.com/daedaleanai/git-ticket/entity"
)

type userOptions struct {
	fields string
}

func ResolveUser(repo *cache.RepoCache, args []string) (*cache.IdentityCache, []string, error) {
	var err error
	var id *cache.IdentityCache
	if len(args) > 0 {
		var userToSelectId entity.Id

		for _, userId := range repo.AllIdentityIds() {
			i, err := repo.ResolveIdentityExcerpt(userId)
			if err != nil {
				return id, nil, err
			}

			if i.Id.HasPrefix(args[0]) || strings.Contains(i.Name, args[0]) {
				if userToSelectId != "" {
					return id, nil, fmt.Errorf("multiple users matching %s", args[0])
				}
				userToSelectId = i.Id
			}
		}

		if userToSelectId == "" {
			return id, nil, fmt.Errorf("no users matching %s", args[0])
		}

		id, err = repo.ResolveIdentity(userToSelectId)
		if err != nil {
			return id, nil, err
		}

		args = args[1:]
	} else {
		id, err = repo.GetUserIdentity()
	}
	return id, args, err
}

func newUserCommand() *cobra.Command {
	env := newEnv()
	options := userOptions{}

	cmd := &cobra.Command{
		Use:      "user [<username/id>]",
		Short:    "Display or change the user identity.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUser(env, options, args)
		},
	}

	cmd.AddCommand(newUserAdoptCommand())
	cmd.AddCommand(newUserCreateCommand())
	cmd.AddCommand(newUserEditCommand())
	cmd.AddCommand(newUserKeyCommand())
	cmd.AddCommand(newUserLsCommand())

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.fields, "field", "f", "",
		"Select field to display. Valid values are [email,humanId,id,keys,lastModification,lastModificationLamport,login,metadata,name,phabId]")

	return cmd
}

func runUser(env *Env, opts userOptions, args []string) error {
	if len(args) > 1 {
		return errors.New("only one identity can be displayed at a time")
	}

	id, args, err := ResolveUser(env.backend, args)

	if err != nil {
		return err
	}

	if opts.fields != "" {
		switch opts.fields {
		case "email":
			env.out.Printf("%s\n", id.Email())
		case "login":
			env.out.Printf("%s\n", id.Login())
		case "humanId":
			env.out.Printf("%s\n", id.Id().Human())
		case "id":
			env.out.Printf("%s\n", id.Id())
		case "keys":
			for _, key := range id.Keys() {
				env.out.Printf("%s\n", key.Fingerprint())
			}
		case "lastModification":
			env.out.Printf("%s\n", id.LastModification().
				Time().Format("Mon Jan 2 15:04:05 2006 +0200"))
		case "lastModificationLamport":
			env.out.Printf("%d\n", id.LastModificationLamport())
		case "metadata":
			for key, value := range id.ImmutableMetadata() {
				env.out.Printf("%s\n%s\n", key, value)
			}
		case "name":
			env.out.Printf("%s\n", id.Name())
		case "phabId":
			env.out.Printf("%s\n", id.PhabID())

		default:
			return fmt.Errorf("\nUnsupported field: %s\n", opts.fields)
		}

		return nil
	}

	env.out.Printf("Id: %s\n", id.Id())
	env.out.Printf("Name: %s\n", id.Name())
	env.out.Printf("Email: %s\n", id.Email())
	env.out.Printf("Login: %s\n", id.Login())
	env.out.Printf("PhabID: %s\n", id.PhabID())
	env.out.Printf("Last modification: %s (lamport %d)\n",
		id.LastModification().Time().Format("2006-01-02 15:04:05"),
		id.LastModificationLamport())
	env.out.Println("Metadata:")
	for key, value := range id.ImmutableMetadata() {
		env.out.Printf("    %s --> %s\n", key, value)
	}
	env.out.Println("Keys:")
	for _, key := range id.Keys() {
		env.out.Printf("    %s (created: %s)\n", key.Fingerprint(), key.CreationTime().Format("2006-01-02 15:04:05"))
	}
	// env.out.Printf("Protected: %v\n", id.IsProtected())

	return nil
}
