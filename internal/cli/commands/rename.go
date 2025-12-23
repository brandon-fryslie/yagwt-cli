package commands

import (
	"github.com/bmf/yagwt/internal/core"
	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:   "rename <selector> <new-name>",
	Short: "Rename a worktree",
	Long: `Change the name (alias) of a worktree.

This only changes the worktree name in metadata, not the directory path.

Examples:
  yagwt rename auth new-auth
  yagwt rename name:old-name new-name`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		initFormatter()

		// Initialize engine
		if err := initEngine(); err != nil {
			handleError(err)
		}

		// Parse selector
		selector := core.ParseSelector(args[0])
		newName := args[1]

		// Rename worktree
		if err := engine.Rename(selector, newName); err != nil {
			handleError(err)
		}

		// Print success message
		if !quiet {
			printOutput(formatter.FormatSuccess("Worktree renamed successfully"))
		}
	},
}
