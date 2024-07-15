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

func queryImpact(configCache *config.ConfigCache, env *Env) ([]string, error) {
	availableImpactLabels, err := configCache.ListLabelsWithNamespace("impact")
	if err != nil {
		return nil, err
	}

	availableImpactLabels = append([]string{"<Exit>"}, availableImpactLabels...)

	selectedImpact := []string{}
	for {
		prompt := promptui.Select{
			Label:  "Select impact. Select `<Exit>` to finish inserting entries",
			Items:  availableImpactLabels,
			Stdout: env.out.WriteCloser,
		}

		selectedItem, _, err := prompt.Run()
		if err != nil {
			return nil, err
		}

		if selectedItem == 0 {
			return selectedImpact, nil
		}

		selectedImpact = append(selectedImpact, bug.ImpactPrefix+availableImpactLabels[selectedItem])
		availableImpactLabels = removeFromSlice(availableImpactLabels, selectedItem)
	}
}

func queryWorkflow(env *Env) (string, error) {
	workflows := bug.GetWorkflowLabels()
	prompt := promptui.Select{
		Label:  "Select workflow",
		Items:  workflows,
		Stdout: env.out.WriteCloser,
	}

	selectedItem, _, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return string(workflows[selectedItem]), err
}

func queryRepo(configCache *config.ConfigCache, env *Env) (string, error) {
	repoLabels, err := configCache.ListLabelsWithNamespace("repo")
	if err != nil {
		return "", err
	}

	prompt := promptui.Select{
		Label:  "Select repository",
		Items:  repoLabels,
		Stdout: env.err.WriteCloser,
	}

	selectedItem, _, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return bug.RepoPrefix + string(repoLabels[selectedItem]), nil
}

func runAdd(env *Env, opts addOptions) error {
	var selectedImpact []string
	var selectedChecklists []string
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
			opts.workflow, err = queryWorkflow(env)
			if err != nil {
				return err
			}
		}

		if opts.repo == "" {
			opts.repo, err = queryRepo(configCache, env)
			if err != nil {
				return err
			}
		}

		if opts.impact == "" {
			selectedImpact, err = queryImpact(configCache, env)
			if err != nil {
				return err
			}
		} else {
			selectedImpact = strings.Split(opts.impact, ",")
		}

		checklistSet := make(map[string]struct{})
		labelMapping := configCache.LabelMapping()
		for _, impact := range selectedImpact {
			if mapping, ok := labelMapping[config.Label(impact)]; ok {
				env.out.Printf("Impact label %q automatically selects the following checklists: %q\n",
					impact, strings.Join(mapping.RequiredChecklists, ","))
				for _, checklist := range mapping.RequiredChecklists {
					checklistSet[checklist] = struct{}{}
				}
			} else {
				env.out.Printf("Impact label %q does not require any checklists\n", impact)
			}
		}

		if mapping, ok := labelMapping[config.Label(opts.repo)]; ok {
			env.out.Printf("Repo label %q automatically selects the following checklists: %q\n",
				opts.repo, strings.Join(mapping.RequiredChecklists, ","))
			for _, checklist := range mapping.RequiredChecklists {
				checklistSet[checklist] = struct{}{}
			}
		}

		for checklist := range checklistSet {
			selectedChecklists = append(selectedChecklists, checklist)
		}

		return nil
	})
	if err != nil {
		return err
	}

	b, _, err := env.backend.NewBug(cache.NewBugOpts{
		Title:      opts.title,
		Message:    opts.message,
		Workflow:   opts.workflow,
		Repo:       opts.repo,
		Impact:     selectedImpact,
		Checklists: selectedChecklists,
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
