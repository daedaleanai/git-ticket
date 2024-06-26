// Package commands contains the CLI commands
package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const rootCommandName = "git-ticket"

// These variables are initialized externally during the build. See the Makefile.
var GitCommit string
var GitLastTag string
var GitExactTag string

// Global flags
var RebuildCache bool
var ChangeDir string

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   rootCommandName,
		Short: "A ticket tracker embedded in Git.",
		Long: `git-ticket is a ticket tracker embedded in git.

git-ticket uses git objects to store the tickets history separate from the main repo
history. As tickets are regular git objects, they can be pushed and pulled from/to
the same git remote you are already using to collaborate with other people.

`,

		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()

			if GitExactTag == "undefined" {
				GitExactTag = ""
			}
			root.Version = GitLastTag
			if GitExactTag == "" {
				root.Version = fmt.Sprintf("%s-dev-%.10s", root.Version, GitCommit)
			}

			if ChangeDir != "" {
				return os.Chdir(ChangeDir)
			}
			return nil
		},

		// For the root command, force the execution of the PreRun
		// even if we just display the help. This is to make sure that we check
		// the repository and give the user early feedback.
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				os.Exit(1)
			}
		},

		SilenceUsage:      true,
		DisableAutoGenTag: true,

		// Custom bash code to connect the git completion for "git bug" to the
		// git-bug completion for "git-bug"
		BashCompletionFunction: `
_git_bug() {
    __start_git-bug "$@"
}
`,
	}

	cmd.PersistentFlags().BoolVarP(&RebuildCache, "rebuild-cache", "", false, "force the cache to be rebuilt")
	cmd.PersistentFlags().StringVarP(&ChangeDir, "change-dir", "", "", "Changes directory before running git-ticket")

	cmd.AddCommand(newAddCommand())
	cmd.AddCommand(newAssignCommand())
	cmd.AddCommand(newCcbCommand())
	cmd.AddCommand(newChecklistCommand())
	cmd.AddCommand(newCommandsCommand())
	cmd.AddCommand(newCommentCommand())
	cmd.AddCommand(newConfigCommand())
	cmd.AddCommand(newDeselectCommand())
	cmd.AddCommand(newLabelCommand())
	cmd.AddCommand(newLsCommand())
	cmd.AddCommand(newLsIdCommand())
	cmd.AddCommand(newLsLabelCommand())
	cmd.AddCommand(newPullCommand())
	cmd.AddCommand(newPushCommand())
	cmd.AddCommand(newResetCommand())
	cmd.AddCommand(newReviewCommand())
	cmd.AddCommand(newRmCommand())
	cmd.AddCommand(newSelectCommand())
	cmd.AddCommand(newShowCommand())
	cmd.AddCommand(newStatusCommand())
	cmd.AddCommand(newTermUICommand())
	cmd.AddCommand(newTitleCommand())
	cmd.AddCommand(newRefreshCommand())
	cmd.AddCommand(newUserCommand())
	cmd.AddCommand(newValidateCommand())
	cmd.AddCommand(newVersionCommand())
	cmd.AddCommand(newWebUICommand())
	cmd.AddCommand(newMigrateCommand())

	return cmd
}

func Execute() {
	if err := NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
