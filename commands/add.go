package commands

import (
	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/input"
)

type addOptions struct {
	title       string
	message     string
	messageFile string
}

func newAddCommand() *cobra.Command {
	env := newEnv()
	options := addOptions{}

	cmd := &cobra.Command{
		Use:      "add",
		Short:    "Create a new ticket.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(env, options)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.title, "title", "t", "",
		"Provide a title to describe the issue")
	flags.StringVarP(&options.message, "message", "m", "",
		"Provide a message to describe the issue")
	flags.StringVarP(&options.messageFile, "file", "F", "",
		"Take the message from the given file. Use - to read the message from the standard input")

	return cmd
}

func runAdd(env *Env, opts addOptions) error {
	var err error
	if opts.messageFile != "" && opts.message == "" {
		opts.title, opts.message, err = input.BugCreateFileInput(opts.messageFile)
		if err != nil {
			return err
		}
	}

	if opts.messageFile == "" && (opts.message == "" || opts.title == "") {
		opts.title, opts.message, err = input.BugCreateEditorInput(env.backend, opts.title, opts.message)

		if err == input.ErrEmptyTitle {
			env.out.Println("Empty title, aborting.")
			return nil
		}
		if err != nil {
			return err
		}
	}

	b, _, err := env.backend.NewBug(opts.title, opts.message)
	if err != nil {
		return err
	}

	env.out.Printf("%s created\n", b.Id().Human())

	return nil
}
