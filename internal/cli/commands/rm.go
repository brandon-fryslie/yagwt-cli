package commands

import (
	"github.com/bmf/yagwt/internal/core"
	"github.com/spf13/cobra"
)

var (
	rmDeleteBranch bool
	rmKeepBranch   bool
	rmOnDirty      string
	rmPatchDir     string
	rmWipMessage   string
	rmForce        bool
)

var rmCmd = &cobra.Command{
	Use:   "rm <selector>",
	Short: "Remove a worktree",
	Long: `Remove a worktree and optionally its branch.

By default, removal will fail if the worktree has uncommitted changes.
Use --on-dirty to specify how to handle dirty worktrees:
  - fail: Abort removal (default)
  - stash: Stash changes before removal
  - patch: Save changes as a patch file
  - wip-commit: Create a WIP commit
  - force: Discard changes (dangerous!)

Examples:
  yagwt rm auth
  yagwt rm name:temp --force
  yagwt rm auth --on-dirty=stash
  yagwt rm auth --on-dirty=patch --patch-dir=/tmp/patches`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		initFormatter()

		// Initialize engine
		if err := initEngine(); err != nil {
			handleError(err)
		}

		// Parse selector
		selector := core.ParseSelector(args[0])

		// Handle --force shortcut
		onDirty := rmOnDirty
		if rmForce && onDirty == "" {
			onDirty = "force"
		}

		// Build remove options
		opts := core.RemoveOptions{
			DeleteBranch: rmDeleteBranch,
			KeepBranch:   rmKeepBranch,
			OnDirty:      onDirty,
			PatchDir:     rmPatchDir,
			WipMessage:   rmWipMessage,
			NoPrompt:     noPrompt || autoYes,
		}

		// Remove workspace
		if err := engine.Remove(selector, opts); err != nil {
			handleError(err)
		}

		// Print success message
		if !quiet {
			printOutput(formatter.FormatSuccess("Worktree removed successfully"))
		}
	},
}

func init() {
	rmCmd.Flags().BoolVar(&rmDeleteBranch, "delete-branch", false, "also delete the branch")
	rmCmd.Flags().BoolVar(&rmKeepBranch, "keep-branch", false, "keep the branch (default)")
	rmCmd.Flags().StringVar(&rmOnDirty, "on-dirty", "", "strategy: fail, stash, patch, wip-commit, force")
	rmCmd.Flags().StringVar(&rmPatchDir, "patch-dir", "", "directory for patches (with --on-dirty=patch)")
	rmCmd.Flags().StringVar(&rmWipMessage, "wip-message", "", "WIP commit message (with --on-dirty=wip-commit)")
	rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "shortcut for --on-dirty=force")
}
