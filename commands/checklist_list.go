package commands

import (
	"sort"

	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/config"
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
	var checklistsArray []config.Checklist

	var maxLabelWidth int
	env.backend.DoWithLockedConfigCache(func(c *config.ConfigCache) error {
		for _, checklist := range c.ChecklistConfig {
			checklistsArray = append(checklistsArray, checklist)

			l := colors.Cyan(string(checklist.Label))
			if len(l) > maxLabelWidth {
				maxLabelWidth = len(l)
			}
		}
		return nil
	})

	sort.Slice(checklistsArray, func(i, j int) bool {
		return checklistsArray[i].Label < checklistsArray[j].Label
	})

	for _, checklist := range checklistsArray {
		env.out.Printf("%-*s %s\n", maxLabelWidth, colors.Cyan(string(checklist.Label)), checklist.Title)
	}
	return nil
}
