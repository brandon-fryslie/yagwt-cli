package commands

import (
	"github.com/bmf/yagwt/internal/core"
	"github.com/spf13/cobra"
)

var (
	cleanPolicy  string
	cleanDryRun  bool
	cleanApply   bool
	cleanOnDirty string
	cleanMax     int
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up stale worktrees",
	Long: `Identify and optionally remove stale worktrees based on cleanup policies.

By default, this shows a plan without executing (dry-run mode).
Use --apply to actually remove worktrees.

Cleanup policies:
  - default: Remove expired ephemeral worktrees and idle worktrees (>30 days)
  - conservative: Only remove expired ephemeral worktrees
  - aggressive: Remove expired ephemeral and idle worktrees (>7 days)

Examples:
  yagwt clean
  yagwt clean --policy aggressive
  yagwt clean --apply
  yagwt clean --apply --max 5
  yagwt clean --apply --on-dirty=stash`,
	Run: func(cmd *cobra.Command, args []string) {
		initFormatter()

		// Initialize engine
		if err := initEngine(); err != nil {
			handleError(err)
		}

		// Determine dry-run mode (default to dry-run unless --apply is set)
		dryRun := !cleanApply
		if cleanDryRun {
			dryRun = true // Explicit --dry-run always wins
		}

		// Build cleanup options
		opts := core.CleanupOptions{
			Policy:  cleanPolicy,
			DryRun:  dryRun,
			OnDirty: cleanOnDirty,
			Max:     cleanMax,
		}

		// Run cleanup
		plan, err := engine.Cleanup(opts)
		if err != nil {
			handleError(err)
		}

		// Format and print output
		output := formatter.FormatCleanupPlan(plan)
		printOutput(output)

		// Print success message if applied
		if !dryRun && !quiet {
			printOutput(formatter.FormatSuccess("Cleanup completed"))
		}
	},
}

func init() {
	cleanCmd.Flags().StringVar(&cleanPolicy, "policy", "default", "cleanup policy: default, conservative, aggressive")
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "show plan without executing (default)")
	cleanCmd.Flags().BoolVar(&cleanApply, "apply", false, "execute the cleanup plan")
	cleanCmd.Flags().StringVar(&cleanOnDirty, "on-dirty", "", "strategy for dirty worktrees: fail, stash, patch, wip-commit, force")
	cleanCmd.Flags().IntVar(&cleanMax, "max", 0, "maximum worktrees to remove (0 = unlimited)")
}
