package commands

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/validate"
)

type validateOptions struct {
	external string
}

func newValidateCommand() *cobra.Command {
	env := newEnv()
	options := validateOptions{}

	cmd := &cobra.Command{
		Use:      "validate COMMIT...",
		Short:    "Validate identities and commits signatures.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(env, options, args)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.external, "external", "e", "", "Validate commit in an external repository at this path")

	return cmd
}

func runValidate(env *Env, options validateOptions, args []string) error {
	validator, err := validate.NewValidator(env.repo, env.backend)
	if err != nil {
		return err
	}

	// If there is no FirstKey it means the repository has no Identities.
	if validator.FirstKey != nil {
		fmt.Printf("first commit signed with key: %s\n", validator.FirstKey.Fingerprint())
	}

	var refErr error
	for _, ref := range args {
		var err error

		if options.external == "" {
			_, err = validator.ValidateCommit(ref)
		} else {
			_, err = validator.ValidateExternalCommit(options.external, ref)
		}

		if err != nil {
			if refErr == nil {
				refErr = errors.Errorf("ref %s check fail", ref)
			} else {
				refErr = errors.Wrapf(refErr, "ref %s check fail", ref)
			}
			fmt.Printf("ref %s\tFAIL: %s\n", ref, err.Error())
		} else {
			fmt.Printf("ref %s\tOK\n", ref)
		}
	}
	if refErr != nil {
		return refErr
	}

	return nil
}
