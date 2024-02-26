package commands

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/bug"
)

func newChecklistShowCommand() *cobra.Command {
	env := newEnv()
	cmd := &cobra.Command{
		Use:      "show <label>",
		Short:    "Shows the contents of the given checklist.",
		Args:     cobra.ExactArgs(1),
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChecklistShow(env, args)
		},
	}

	return cmd
}

func runChecklistShow(env *Env, args []string) error {
	checklist, err := bug.GetChecklist(bug.Label(args[0]))
	if err != nil {
		return err
	}

	env.out.Printf("%s\n", color.CyanString(checklist.Title))

	for i, section := range checklist.Sections {
		env.out.Printf(color.GreenString(fmt.Sprintf("#### %d. %s ####\n", i, section.Title)))
		for j, question := range section.Questions {
			env.out.Printf("(%d.%d) %s\n", i, j, question.Question)
		}
	}

	return nil
}
