package commands

import (
	"github.com/bmf/yagwt/internal/core"
	"github.com/spf13/cobra"
)

var pathCmd = &cobra.Command{
	Use:   "path <selector>",
	Short: "Print worktree path",
	Long: `Print the absolute path to a worktree.

This command is designed for scripting and always outputs just the path,
regardless of the --json or --porcelain flags.

Examples:
  cd $(yagwt path auth)
  yagwt path feature-x`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		initFormatter()

		// Initialize engine
		if err := initEngine(); err != nil {
			handleError(err)
		}

		// Parse selector
		selector := core.ParseSelector(args[0])

		// Get worktree
		worktree, err := engine.Get(selector)
		if err != nil {
			handleError(err)
		}

		// Print just the path (for scripting)
		output := formatter.FormatWorkspacePath(worktree)
		printOutput(output)
	},
}
