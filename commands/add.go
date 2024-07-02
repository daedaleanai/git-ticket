package commands

import (
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/bug"
	"github.com/daedaleanai/git-ticket/cache"
	"github.com/daedaleanai/git-ticket/config"
	"github.com/daedaleanai/git-ticket/input"

	_select "github.com/daedaleanai/git-ticket/commands/select"
)

type addOptions struct {
	title       string
	message     string
	messageFile string
	workflow    string
	repo        string
	impact      string
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
	flags.StringVarP(&options.repo, "repo", "r", "",
		"Provide the repository affected by this ticket")
	flags.StringVarP(&options.impact, "impact", "i", "",
		"Provide the impact labels, using commas as separators")
	flags.BoolVarP(&options.noSelect, "noselect", "n", false,
		"Do not automatically select the new ticket once it's created")

	return cmd
}

/// Keeps relative order so that the user does not notice any changes other than the removal.
func removeFromSlice(s []string, index int) []string {
	return append(s[:index], s[index+1:]...)
}

func queryImpact(configCache *config.ConfigCache) ([]string, error) {
	availableImpactLabels, err := configCache.ListLabelsWithNamespace("impact")
	if err != nil {
		return nil, err
	}

	selectedImpact := []string{}
	for {
		prompt := promptui.Select{
			Label: "Select impact. Use <CTRL-D> to stop submitting impact labels.",
			Items: availableImpactLabels,
		}

		selectedItem, _, err := prompt.Run()
		if err == promptui.ErrEOF {
			break
		}
		if err != nil {
			return nil, err
		}

		selectedImpact = append(selectedImpact, "impact:"+availableImpactLabels[selectedItem])
		availableImpactLabels = removeFromSlice(availableImpactLabels, selectedItem)
	}
	return selectedImpact, nil
}

func runAdd(env *Env, opts addOptions) error {
	var selectedImpact []string
	err := env.backend.DoWithLockedConfigCache(func(configCache *config.ConfigCache) error {
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

		if opts.repo == "" {
			repoLabels, err := configCache.ListLabelsWithNamespace("repo")
			if err != nil {
				return err
			}

			prompt := promptui.Select{
				Label: "Select repository",
				Items: repoLabels,
			}

			selectedItem, _, err := prompt.Run()
			if err != nil {
				return err
			}
			opts.repo = "repo:" + string(repoLabels[selectedItem])
		}

		if opts.impact == "" {
			selectedImpact, err = queryImpact(configCache)
			if err != nil {
				return err
			}
		} else {
			// TODO: check validity of the given impact label
			selectedImpact = strings.Split(opts.impact, ",")
		}
		return nil
	})
	if err != nil {
		return err
	}

	b, _, err := env.backend.NewBug(cache.NewBugOpts{
		Title:    opts.title,
		Message:  opts.message,
		Workflow: opts.workflow,
		Repo:     opts.repo,
		Impact:   selectedImpact,
	})
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
