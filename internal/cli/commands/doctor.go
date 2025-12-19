package commands

import (
	"github.com/bmf/yagwt/internal/core"
	"github.com/spf13/cobra"
)

var (
	doctorDryRun        bool
	doctorForgetMissing bool
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose and repair issues",
	Long: `Detect and optionally repair inconsistencies in workspace metadata.

This command checks for:
  - Orphaned metadata (metadata without git worktree)
  - Untracked worktrees (git worktree without metadata)
  - Stale index entries

By default, this shows issues without fixing them (dry-run mode).

Examples:
  yagwt doctor
  yagwt doctor --forget-missing
  yagwt doctor --dry-run`,
	Run: func(cmd *cobra.Command, args []string) {
		initFormatter()

		// Initialize engine
		if err := initEngine(); err != nil {
			handleError(err)
		}

		// Build doctor options
		opts := core.DoctorOptions{
			DryRun:        doctorDryRun,
			ForgetMissing: doctorForgetMissing,
		}

		// If --forget-missing is set without explicit --dry-run, apply repairs
		if doctorForgetMissing && !doctorDryRun {
			opts.DryRun = false
		}

		// Run doctor
		report, err := engine.Doctor(opts)
		if err != nil {
			handleError(err)
		}

		// Format and print output
		output := formatter.FormatDoctorReport(report)
		printOutput(output)

		// Print success message if repairs were applied
		if !opts.DryRun && !quiet {
			printOutput(formatter.FormatSuccess("Doctor completed"))
		}
	},
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorDryRun, "dry-run", false, "show issues without fixing (default)")
	doctorCmd.Flags().BoolVar(&doctorForgetMissing, "forget-missing", false, "remove metadata for missing worktrees")
}
