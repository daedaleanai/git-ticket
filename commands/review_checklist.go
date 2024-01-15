package commands

import (
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/bug"
	_select "github.com/daedaleanai/git-ticket/commands/select"
	"github.com/daedaleanai/git-ticket/input"
)

type reviewChecklistOptions struct {
	blank bool
}

func newReviewChecklistCommand() *cobra.Command {
	env := newEnv()
	options := reviewChecklistOptions{}

	cmd := &cobra.Command{
		Use:      "checklist [ticket id]",
		Short:    "Complete a checklist associated with a ticket.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReviewChecklist(env, options, args)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.BoolVarP(&options.blank, "blank", "b", false,
		"Discard any previously edited checklist and start again with a blank one",
	)

	return cmd

}

func runReviewChecklist(env *Env, opts reviewChecklistOptions, args []string) error {
	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	id, err := env.backend.GetUserIdentity()
	if err != nil {
		return err
	}

	ticketChecklists, err := b.Snapshot().GetUserChecklists(id.Id(), opts.blank)
	if err != nil {
		return err
	}

	if len(ticketChecklists) == 0 {
		fmt.Println("No checklists associated with ticket")
		return nil
	}

	// Collect checklist labels
	ticketChecklistLabels := make([]string, 0, len(ticketChecklists))

	for k := range ticketChecklists {
		ticketChecklistLabels = append(ticketChecklistLabels, string(k))
	}

	// If there are multiple checklists associated with the ticket then give the
	// user the option to choose which to edit rather than editing every one

	var selectedChecklistLabel string

	if len(ticketChecklistLabels) > 1 {
		prompt := promptui.Select{
			Label: "Select Checklist",
			Items: ticketChecklistLabels,
		}

		_, selectedChecklistLabel, err = prompt.Run()

		if err != nil {
			return err
		}
	} else {
		selectedChecklistLabel = ticketChecklistLabels[0]
	}

	// Use the editor to edit the checklist, if it changed then create an update
	// operation and commit
	clChange, err := input.ChecklistEditorInput(env.repo, ticketChecklists[bug.Label(selectedChecklistLabel)])
	if err != nil {
		return err
	}

	if clChange {
		_, err = b.SetChecklist(ticketChecklists[bug.Label(selectedChecklistLabel)])
		if err != nil {
			return err
		}

		return b.Commit()
	}

	fmt.Println("Checklists unchanged")
	return nil
}
