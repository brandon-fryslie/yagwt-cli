package commands

import (
	"github.com/bmf/yagwt/internal/core"
	"github.com/spf13/cobra"
)

var lockCmd = &cobra.Command{
	Use:   "lock <selector>",
	Short: "Lock a workspace",
	Long: `Lock a workspace to prevent modifications.

Locked workspaces cannot be removed or modified until unlocked.

Examples:
  yagwt lock auth
  yagwt lock name:production`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		initFormatter()

		// Initialize engine
		if err := initEngine(); err != nil {
			handleError(err)
		}

		// Parse selector
		selector := core.ParseSelector(args[0])

		// Lock workspace
		if err := engine.Lock(selector); err != nil {
			handleError(err)
		}

		// Print success message
		if !quiet {
			printOutput(formatter.FormatSuccess("Workspace locked successfully"))
		}
	},
}

var unlockCmd = &cobra.Command{
	Use:   "unlock <selector>",
	Short: "Unlock a workspace",
	Long: `Remove the lock flag from a workspace.

This allows the workspace to be modified or removed.

Examples:
  yagwt unlock auth
  yagwt unlock name:production`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		initFormatter()

		// Initialize engine
		if err := initEngine(); err != nil {
			handleError(err)
		}

		// Parse selector
		selector := core.ParseSelector(args[0])

		// Unlock workspace
		if err := engine.Unlock(selector); err != nil {
			handleError(err)
		}

		// Print success message
		if !quiet {
			printOutput(formatter.FormatSuccess("Workspace unlocked successfully"))
		}
	},
}
