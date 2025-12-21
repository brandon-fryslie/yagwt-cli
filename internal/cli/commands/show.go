package commands

import (
	"github.com/bmf/yagwt/internal/core"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <selector>",
	Short: "Show worktree details",
	Long: `Display detailed information about a worktree.

The selector can be:
  - Worktree ID (id:wsp_...)
  - Worktree name (name:feature-x or just feature-x)
  - Path (path:/full/path/to/worktree)
  - Branch name (branch:feature/x)

Examples:
  yagwt show auth
  yagwt show name:feature-x
  yagwt show id:wsp_01HZX...
  yagwt show --json auth`,
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

		// Format and print output
		output := formatter.FormatWorkspace(worktree)
		printOutput(output)
	},
}
