package commands

import (
	"github.com/bmf/yagwt/internal/core"
	"github.com/spf13/cobra"
)

var pinCmd = &cobra.Command{
	Use:   "pin <selector>",
	Short: "Pin a worktree",
	Long: `Pin a worktree to prevent it from being automatically cleaned up.

Pinned worktrees are protected from cleanup operations and must be
explicitly removed with 'yagwt rm'.

Examples:
  yagwt pin auth
  yagwt pin name:important-feature`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		initFormatter()

		// Initialize engine
		if err := initEngine(); err != nil {
			handleError(err)
		}

		// Parse selector
		selector := core.ParseSelector(args[0])

		// Pin worktree
		if err := engine.Pin(selector); err != nil {
			handleError(err)
		}

		// Print success message
		if !quiet {
			printOutput(formatter.FormatSuccess("Worktree pinned successfully"))
		}
	},
}

var unpinCmd = &cobra.Command{
	Use:   "unpin <selector>",
	Short: "Unpin a worktree",
	Long: `Remove the pin flag from a worktree.

This allows the worktree to be cleaned up by cleanup policies.

Examples:
  yagwt unpin auth
  yagwt unpin name:old-feature`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		initFormatter()

		// Initialize engine
		if err := initEngine(); err != nil {
			handleError(err)
		}

		// Parse selector
		selector := core.ParseSelector(args[0])

		// Unpin worktree
		if err := engine.Unpin(selector); err != nil {
			handleError(err)
		}

		// Print success message
		if !quiet {
			printOutput(formatter.FormatSuccess("Worktree unpinned successfully"))
		}
	},
}
