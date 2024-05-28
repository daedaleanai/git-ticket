package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newLsLabelCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:   "ls-label",
		Short: "List valid labels.",
		Long: `List valid labels.

Note: in the future, a proper label policy could be implemented where valid labels are defined in a configuration file. Until that, the default behavior is to return the list of labels already used.`,
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLsLabel(env)
		},
	}

	return cmd
}

func runLsLabel(env *Env) error {
	labels, err := env.backend.ValidLabels()
	if err != nil {
		return fmt.Errorf("Error reading the list of valid labels: %s", err)
	}

	for _, l := range labels {
		env.out.Println(l)
	}

	return nil
}
