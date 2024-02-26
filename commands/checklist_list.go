package commands

import (
	"sort"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/bug"
	"github.com/daedaleanai/git-ticket/util/colors"
)

func newChecklistListCommand() *cobra.Command {
	env := newEnv()
	cmd := &cobra.Command{
		Use:      "list",
		Short:    "Lists the available checklists.",
		Args:     cobra.NoArgs,
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChecklistList(env, args)
		},
	}

	return cmd
}

func runChecklistList(env *Env, args []string) error {
	checklists, err := bug.ListChecklists()
	if err != nil {
		return err
	}

	var checklistsArray []bug.Checklist

	var maxLabelWidth int
	for _, checklist := range checklists {
		checklistsArray = append(checklistsArray, checklist)

		l := colors.Cyan(string(checklist.Label))
		if len(l) > maxLabelWidth {
			maxLabelWidth = len(l)
		}
	}
	sort.Slice(checklistsArray, func(i, j int) bool {
		return checklistsArray[i].Label < checklistsArray[j].Label
	})

	for _, checklist := range checklistsArray {
		env.out.Printf("%-*s %s\n", maxLabelWidth, color.CyanString(string(checklist.Label)), checklist.Title)
	}
	return nil
}
