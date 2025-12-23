package commands

import (
	"github.com/bmf/yagwt/internal/core"
	"github.com/spf13/cobra"
)

var (
	lsFilter string
	lsAll    bool
)

var lsCmd = &cobra.Command{
	Use:   "ls [filter]",
	Short: "List worktrees",
	Long: `List all worktrees with their status and metadata.

Supports filtering with expressions like:
  flag:pinned          - Show only pinned worktrees
  flag:ephemeral       - Show only ephemeral worktrees
  status:dirty         - Show only dirty worktrees

Examples:
  yagwt ls
  yagwt ls --json
  yagwt ls --filter "flag:pinned"
  yagwt ls flag:ephemeral`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		initFormatter()

		// Initialize engine
		if err := initEngine(); err != nil {
			handleError(err)
		}

		// Determine filter (from flag or positional arg)
		filter := lsFilter
		if len(args) > 0 {
			filter = args[0]
		}

		// List workspaces
		workspaces, err := engine.List(core.ListOptions{
			Filter: filter,
			All:    lsAll,
		})
		if err != nil {
			handleError(err)
		}

		// Format and print output
		output := formatter.FormatWorkspaces(workspaces)
		printOutput(output)
	},
}

func init() {
	lsCmd.Flags().StringVarP(&lsFilter, "filter", "f", "", "filter expression")
	lsCmd.Flags().BoolVarP(&lsAll, "all", "a", false, "show all worktrees including broken")
}
