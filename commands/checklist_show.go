package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/config"
	"github.com/daedaleanai/git-ticket/util/colors"
)

func newChecklistShowCommand() *cobra.Command {
	env := newEnv()
	cmd := &cobra.Command{
		Use:      "show <label>",
		Short:    "Shows the contents of the given checklist.",
		Args:     cobra.ExactArgs(1),
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChecklistShow(env, args)
		},
	}

	return cmd
}

func runChecklistShow(env *Env, args []string) error {
	var checklist config.Checklist
	err := env.backend.DoWithLockedConfigCache(func(c *config.ConfigCache) error {
		inner, err := c.GetChecklist(config.Label(args[0]))
		checklist = inner
		return err
	})
	if err != nil {
		return err
	}

	env.out.Printf("%s\n", colors.Cyan(checklist.Title))

	for i, section := range checklist.Sections {
		env.out.Printf(colors.Green(fmt.Sprintf("#### %d. %s ####\n", i, section.Title)))
		for j, question := range section.Questions {
			env.out.Printf("(%d.%d) %s\n", i, j, question.Question)
		}
	}

	return nil
}
