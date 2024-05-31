package commands

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/daedaleanai/git-ticket/bug"
	"github.com/daedaleanai/git-ticket/cache"
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

func newMigrateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "migrate",
		Short:  "Perform database migrations for git-ticket updates",
		Hidden: true,
	}
	cmd.AddCommand(newMigrateRepoLabelCommand())
	return cmd
}

func newMigrateRepoLabelCommand() *cobra.Command {
	env := newEnv()

	return &cobra.Command{
		Use:      "repo-label",
		Short:    "Migrates repository labels in ticket titles to `repo:` labels in an automated way",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrateRepoLabel(env)
		},
		Hidden: true,
	}
}

var titleMatch *regexp.Regexp = regexp.MustCompile(`^\[([a-zA-Z0-9-]+)\] (.*)$`)

func ticketHasLabel(ticket *cache.BugExcerpt, label bug.Label) bool {
	for _, l := range ticket.Labels {
		if l == label {
			return true
		}
	}
	return false

}

type skipTicketError int

const kSkipTicketError skipTicketError = 0

func (e skipTicketError) Error() string {
	return "Skip this ticket"
}

func promptUnknownRepositoryLabel(repo, id, title string) (string, error) {
	fmt.Printf("Unknown repository for ticket %s: %s\n", id, repo)
	fmt.Printf("Original ticket title: %s\n", title)

	const kSkipIt string = "Skip it"
	const kSelectRepo string = "Select repo label manually"

	selectActionPrompt := promptui.Select{
		Label: "How shall I handle this ticket?",
		Items: []string{
			kSkipIt,
			kSelectRepo,
		},
	}
	_, selectedAction, err := selectActionPrompt.Run()
	if err != nil {
		return "", err
	}

	switch selectedAction {
	case kSkipIt:
		return "", kSkipTicketError

	case kSelectRepo:
		labelsMap, err := bug.ListLabels()
		if err != nil {
			return "", fmt.Errorf("Unable to list supported labels: %s", err)
		}

		labels := []bug.Label{}
		for label := range labelsMap {
			if strings.HasPrefix(string(label), "repo:") {
				labels = append(labels, label)
			}
		}

		_, selectedLabel, err := (&promptui.Select{
			Label: "Select repo label",
			Items: labels,
			Searcher: func(input string, index int) bool {
				return strings.HasPrefix(string(labels[index]), input)
			},
			StartInSearchMode: true,
		}).Run()
		return selectedLabel, err

	}

	return "", fmt.Errorf("Unknown selected action: %s", selectedAction)
}

func runMigrateRepoLabel(env *Env) error {
	labels, err := bug.ListLabels()
	if err != nil {
		return fmt.Errorf("Unable to list supported labels: %s", err)
	}

	skipped := []entity.Id{}
	for _, ticketId := range env.backend.AllBugsIds() {
		ticketExcerpt, err := env.backend.ResolveBugExcerpt(ticketId)
		if err != nil {
			return fmt.Errorf("Unable to get ticket %s: %s", ticketId, err)
		}

		title := ticketExcerpt.Title
		if match := titleMatch.FindStringSubmatch(title); match != nil {
			repo := match[1]
			title := match[2]

			// Assume repo is a valid label and match it agains known labels
			if !strings.HasPrefix(repo, "prod-") {
				repo = fmt.Sprintf("prod-%s", repo)
			}
			repo = fmt.Sprintf("repo:%s", repo)

			if _, ok := labels[bug.Label(repo)]; !ok {
				selected, err := promptUnknownRepositoryLabel(repo, ticketId.Human(), title)
				if err == kSkipTicketError {
					skipped = append(skipped, ticketId)
					continue
				} else if err != nil {
					return fmt.Errorf("Error during repo query: %s", err)
				} else {
					repo = selected
				}
			}

			ticket, err := env.backend.ResolveBug(ticketId)
			if err != nil {
				return fmt.Errorf("Unable to get ticket %s: %s", ticketId, err)
			}

			if !ticketHasLabel(ticketExcerpt, bug.Label(repo)) {
				fmt.Printf("Adding label %s to ticket %s\n", repo, ticketId.Human())
				changes, _, err := ticket.ChangeLabels([]string{repo}, nil, false)
				for _, change := range changes {
					env.out.Println(change)
				}
				if err != nil {
					return fmt.Errorf("Unable to change labels for ticket %s: %s", ticketId.Human(), err)
				}
			}

			// Remove repo from title
			_, err = ticket.SetTitle(title)
			if err != nil {
				return fmt.Errorf("Unable to set title for ticket %s: %s", ticketId.Human(), err)
			}

			err = ticket.Commit()
			if err != nil {
				return fmt.Errorf("Unable to commit changes for ticket %s: %s", ticketId.Human(), err)
			}
		}
	}

	fmt.Println("The following tickets were skipped:")
	for _, skipped := range skipped {
		ticketExcerpt, err := env.backend.ResolveBugExcerpt(skipped)
		if err != nil {
			return err
		}

		fmt.Println(skipped.Human(), ": ", ticketExcerpt.Title)
	}

	return nil
}
