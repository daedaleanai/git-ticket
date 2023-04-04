package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/daedaleanai/git-ticket/input"
)

type configSetOptions struct {
	file string
}

func newConfigSetCommand() *cobra.Command {
	env := newEnv()
	options := configSetOptions{}

	cmd := &cobra.Command{
		Use:      "set <config name>",
		Short:    "Set the named configuration data.",
		Args:     cobra.ExactArgs(1),
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigSet(env, options, args)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.file, "file", "F", "",
		"Take the config from the given file. Use - to read the config from the standard input",
	)

	return cmd
}

func runConfigSet(env *Env, options configSetOptions, args []string) error {
	var configData string

	if options.file != "" {
		var err error
		configData, err = input.ConfigFileInput(options.file)
		if err != nil {
			return err
		}
	} else {
		currentConfig, err := env.backend.GetConfig(args[0])
		if err != nil {
			configData, err = input.ConfigEditorInput(env.backend, "")
		} else {
			configData, err = input.ConfigEditorInput(env.backend, string(currentConfig))
		}
		if err == input.ErrEmptyMessage {
			fmt.Println("Empty config, aborting.")
			return nil
		}

		if err != nil {
			return fmt.Errorf("failed to get config data from the editor: %s", err)
		}
	}

	// Validate json
	var tmp map[string]interface{}
	if err := yaml.Unmarshal([]byte(configData), &tmp); err != nil {
		return fmt.Errorf("the config data you specified is not properly formatted: %s", err)
	}

	return env.backend.SetConfig(args[0], []byte(configData))
}
