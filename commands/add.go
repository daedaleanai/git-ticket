package commands

import (
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/bug"
	"github.com/daedaleanai/git-ticket/input"

	_select "github.com/daedaleanai/git-ticket/commands/select"
)

type addOptions struct {
	title       string
	message     string
	messageFile string
	workflow    string
	noSelect    bool
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
	flags.StringVarP(&options.workflow, "workflow", "w", "",
		"Provide a workflow to apply to this ticket")
	flags.BoolVarP(&options.noSelect, "noselect", "n", false,
		"Do not automatically select the new ticket once it's created")

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

	if opts.workflow == "" {
		workflows := bug.GetWorkflowLabels()
		prompt := promptui.Select{
			Label: "Select workflow",
			Items: workflows,
		}

		selectedItem, _, err := prompt.Run()
		if err != nil {
			return err
		}
		opts.workflow = string(workflows[selectedItem])
	}

	b, _, err := env.backend.NewBug(opts.title, opts.message, opts.workflow)
	if err != nil {
		return err
	}

	env.out.Printf("%s created\n", b.Id().Human())

	if opts.noSelect == false {
		err = _select.Select(env.backend, b.Id())
		if err != nil {
			return err
		}

		env.out.Printf("selected ticket: %s\n", opts.title)
	}

	return nil
}
