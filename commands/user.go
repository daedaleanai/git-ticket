package commands

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/daedaleanai/git-ticket/cache"
)

type userOptions struct {
	fields string
}

func ResolveUser(repo *cache.RepoCache, args []string) (*cache.IdentityCache, []string, error) {
	var err error
	var id *cache.IdentityCache
	if len(args) > 0 {
		for _, userId := range repo.AllIdentityIds() {
			i, err := repo.ResolveIdentityExcerpt(userId)
			if err != nil {
				return id, nil, err
			}
			userMatch, err := compareUsername(i.Name, args[0])
			if err != nil {
				return id, nil, err
			}

			if i.Id.HasPrefix(args[0]) || userMatch {
				if id != nil {
					return id, nil, fmt.Errorf("multiple users matching %s:\n%s\n%s", args[0], id.Name(), i.Name)
				}

				id, err = repo.ResolveIdentity(i.Id)
				if err != nil {
					return id, nil, err
				}
			}
		}

		if id == nil {
			return id, nil, fmt.Errorf("no users matching %s", args[0])
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
		Use:      "user [{user_name | user_id}]",
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
	cmd.AddCommand(newUserFixupCommand())

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
	env.out.Printf("GiteaID: %v\n", id.GiteaID())
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

// compareUsername tries to match a search string with a username, ignoring case and removing diacritics
func compareUsername(username, search string) (bool, error) {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	usernameOut, _, err := transform.String(t, username)
	if err != nil {
		return false, err
	}
	usernameOut = strings.ToUpper(usernameOut)
	searchOut, _, err := transform.String(t, search)
	if err != nil {
		return false, err
	}
	searchOut = strings.ToUpper(searchOut)

	if strings.HasPrefix(usernameOut, searchOut) {
		return true, nil
	}

	for _, userPart := range strings.Fields(usernameOut) {
		if strings.HasPrefix(userPart, searchOut) {
			return true, nil
		}
	}
	return false, nil
}
