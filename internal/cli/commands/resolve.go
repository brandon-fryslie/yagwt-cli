package commands

import (
	"github.com/spf13/cobra"
)

var resolveCmd = &cobra.Command{
	Use:   "resolve <selector>",
	Short: "Resolve selector to worktree(s)",
	Long: `Resolve a selector to all matching worktrees.

This is useful for debugging selector ambiguity or seeing what worktrees
match a particular pattern.

Examples:
  yagwt resolve auth
  yagwt resolve branch:feature/x
  yagwt resolve --json name:test`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		initFormatter()

		// Initialize engine
		if err := initEngine(); err != nil {
			handleError(err)
		}

		// Resolve selector
		worktrees, err := engine.Resolve(args[0])
		if err != nil {
			handleError(err)
		}

		// Format and print output
		output := formatter.FormatWorkspaces(worktrees)
		printOutput(output)
	},
}
