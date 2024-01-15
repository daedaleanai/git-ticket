package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newConfigCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "config [config name]",
		Short:    "List configs or show the specified config",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfig(env, args)
		},
	}

	cmd.AddCommand(newConfigSetCommand())

	return cmd
}

func runConfig(env *Env, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("only one config can be displayed at a time")
	}

	if len(args) == 1 {
		data, err := env.backend.GetConfig(args[0])

		if err != nil {
			return fmt.Errorf("failed to get config %s: %s", args[0], err)
		}

		fmt.Println(string(data))
	} else {
		configs, err := env.backend.ListConfigs()
		if err != nil {
			return fmt.Errorf("failed to list configs: %s", err)
		}

		for _, config := range configs {
			fmt.Printf("%s\n", config)
		}
	}

	return nil
}
