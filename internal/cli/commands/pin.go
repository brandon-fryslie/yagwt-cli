package commands

import (
	"github.com/bmf/yagwt/internal/core"
	"github.com/spf13/cobra"
)

var pinCmd = &cobra.Command{
	Use:   "pin <selector>",
	Short: "Pin a workspace",
	Long: `Pin a workspace to prevent it from being automatically cleaned up.

Pinned workspaces are protected from cleanup operations and must be
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

		// Pin workspace
		if err := engine.Pin(selector); err != nil {
			handleError(err)
		}

		// Print success message
		if !quiet {
			printOutput(formatter.FormatSuccess("Workspace pinned successfully"))
		}
	},
}

var unpinCmd = &cobra.Command{
	Use:   "unpin <selector>",
	Short: "Unpin a workspace",
	Long: `Remove the pin flag from a workspace.

This allows the workspace to be cleaned up by cleanup policies.

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

		// Unpin workspace
		if err := engine.Unpin(selector); err != nil {
			handleError(err)
		}

		// Print success message
		if !quiet {
			printOutput(formatter.FormatSuccess("Workspace unpinned successfully"))
		}
	},
}
