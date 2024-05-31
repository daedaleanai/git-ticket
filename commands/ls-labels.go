package commands

import (
	"fmt"
	"regexp"

	"github.com/spf13/cobra"
)

func newLsLabelCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "ls-label [pattern]",
		Short:    "List valid labels.",
		Long:     `List valid labels, an optional regexp pattern limits the output.`,
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLsLabel(env, args)
		},
	}

	return cmd
}

func runLsLabel(env *Env, args []string) error {
	labels, err := env.backend.ValidLabels()
	if err != nil {
		return fmt.Errorf("Error reading the list of valid labels: %s", err)
	}

	if len(args) == 0 {
		// No pattern provided, output full list
		for _, l := range labels {
			env.out.Println(l)
		}
	} else {
		pattern, err := regexp.Compile(args[0])
		if err != nil {
			return fmt.Errorf("Error compiling pattern: %s", err)
		}
		for _, l := range labels {
			if pattern.MatchString(l.String()) {
				env.out.Println(l)
			}
		}
	}

	return nil
}
