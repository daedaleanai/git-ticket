package commands

import (
	"fmt"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/bug"
	"github.com/daedaleanai/git-ticket/cache"
	"github.com/daedaleanai/git-ticket/config"
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/input"
	"github.com/daedaleanai/git-ticket/util/colors"

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

func queryCcbMemberFromTeam(configCache *config.ConfigCache, env *Env, teamName string, excludedUserId *entity.Id) (*cache.IdentityExcerpt, error) {
	team, err := configCache.GetCcbTeam(teamName)
	if err != nil {
		return nil, err
	}

	var items []*cache.IdentityExcerpt
	for _, member := range team.Members {
		if excludedUserId != nil && member.Id == *excludedUserId {
			continue
		}

		id, err := env.backend.ResolveIdentityExcerpt(member.Id)
		if err != nil {
			return nil, err
		}

		items = append(items, id)
	}
	items = append(items, nil)

	m := promptui.FuncMap
	m["DisplayName"] = func(ident *cache.IdentityExcerpt) string {
		if ident == nil {
			return "<None>"
		}
		return ident.DisplayName()
	}
	m["Details"] = func(ident *cache.IdentityExcerpt) string {
		if ident == nil {
			return colors.Red("Select this if you are sure that this CCB review is not required")
		}
		return ""
	}

	prompt := promptui.Select{
		Label: fmt.Sprintf("Select CCB member from %s", teamName),
		Items: items,
		Templates: &promptui.SelectTemplates{
			Active:   fmt.Sprintf("%s {{ . | DisplayName | underline }}", promptui.IconSelect),
			Inactive: "  {{ . | DisplayName }}",
			Selected: fmt.Sprintf(`{{ "%s" | green }} {{ . | DisplayName | faint }}`, promptui.IconGood),
			Details:  "{{ . | Details }}",
			FuncMap:  m,
		},
		Stdout: env.err.WriteCloser,
	}

	selectedItem, _, err := prompt.Run()
	if err != nil {
		return nil, err
	}

	return items[selectedItem], nil
}

func queryCCBMembers(configCache *config.ConfigCache, env *Env, selectedImpact []string, repo string) (map[bug.Status][]entity.Id, error) {
	// The keys to these maps is the CCB team. The values are the identities selected for each team.
	// In principle there is no reason to have more than one person from a given team CCB'ing the change,
	// other than the fact that they may be different between primary and secondary, which is already handled
	// by having separate data structures for each.
	primaryCCBPerTeam := map[string]entity.Id{}
	secondaryCCBPerTeam := map[string]entity.Id{}

	labelMapping := configCache.LabelMapping()

	handleLabel := func(label string) error {
		handleCcbGroup := func(requiredTeams []string, ccbGroup map[string]entity.Id, otherGroup map[string]entity.Id, groupName string) error {
			for _, team := range requiredTeams {
				if selectedUser, ok := ccbGroup[team]; ok {
					env.out.Printf("Label %q requires %s CCB from team %q. A CCB user has been added already from this team: %q\n",
						label, groupName, team, selectedUser)
				} else {
					env.out.Printf("Label %q requires %s CCB from team %q\n",
						label, groupName, team)

					var excludedMember *entity.Id = nil
					if secondaryCcbMember, ok := otherGroup[team]; ok {
						excludedMember = &secondaryCcbMember
					}

					ident, err := queryCcbMemberFromTeam(configCache, env, team, excludedMember)
					if err != nil {
						return err
					}
					if ident != nil {
						ccbGroup[team] = ident.Id
					}
				}
			}
			return nil
		}

		if mapping, ok := labelMapping[config.Label(label)]; ok {
			if err := handleCcbGroup(mapping.PrimaryCcbTeams, primaryCCBPerTeam, secondaryCCBPerTeam, "primary"); err != nil {
				return err
			}
			if err := handleCcbGroup(mapping.SecondaryCcbTeams, secondaryCCBPerTeam, primaryCCBPerTeam, "secondary"); err != nil {
				return err
			}
		}
		return nil
	}

	for _, impact := range selectedImpact {
		err := handleLabel(impact)
		if err != nil {
			return nil, err
		}
	}

	err := handleLabel(repo)
	if err != nil {
		return nil, err
	}

	ccbMembers := map[bug.Status][]entity.Id{
		bug.VettedStatus:   make([]entity.Id, 0),
		bug.AcceptedStatus: make([]entity.Id, 0),
	}
	for _, member := range primaryCCBPerTeam {
		ccbMembers[bug.VettedStatus] = append(ccbMembers[bug.VettedStatus], member)
		ccbMembers[bug.AcceptedStatus] = append(ccbMembers[bug.AcceptedStatus], member)
	}

	for _, member := range secondaryCCBPerTeam {
		ccbMembers[bug.VettedStatus] = append(ccbMembers[bug.VettedStatus], member)
	}

	return ccbMembers, nil
}

func runAdd(env *Env, opts addOptions) error {
	var selectedImpact []string
	var selectedCcbMembers map[bug.Status][]entity.Id

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

		selectedCcbMembers, err = queryCCBMembers(configCache, env, selectedImpact, opts.repo)
		return err
	})
	if err != nil {
		return err
	}

	labels := append([]string{opts.repo}, selectedImpact...)
	selectedChecklists, checklistMessages := env.backend.FindChecklists(labels)
	for _, message := range checklistMessages {
		env.out.Print(message)
	}

	b, _, err := env.backend.NewBug(cache.NewBugOpts{
		Title:      opts.title,
		Message:    opts.message,
		Workflow:   opts.workflow,
		Repo:       opts.repo,
		Impact:     selectedImpact,
		Checklists: selectedChecklists,
		CcbMembers: selectedCcbMembers,
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
