package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Common flags used across multiple commands
	repoPath    string
	jsonOutput  bool
	porcelain   bool
	quiet       bool
	noPrompt    bool
	autoYes     bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "yagwt",
	Short: "Yet Another Git Worktree Manager",
	Long: `YAGWT is a powerful CLI tool for managing git worktrees with lifecycle management,
metadata tracking, and intelligent cleanup policies.

Use 'yagwt <command> --help' for information on a specific command.`,
	Version: "1.0.0-dev",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags available to all commands
	rootCmd.PersistentFlags().StringVar(&repoPath, "repo", "", "repository root (default: auto-detect from current directory)")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format (machine-readable)")
	rootCmd.PersistentFlags().BoolVar(&porcelain, "porcelain", false, "output in stable porcelain format (tab-separated)")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "suppress non-essential output")
	rootCmd.PersistentFlags().BoolVar(&noPrompt, "no-prompt", false, "never ask questions (fail if input required)")
	rootCmd.PersistentFlags().BoolVarP(&autoYes, "yes", "y", false, "automatically answer yes to all prompts")

	// Add subcommands here (will be implemented in separate files)
	// rootCmd.AddCommand(lsCmd)
	// rootCmd.AddCommand(newCmd)
	// rootCmd.AddCommand(rmCmd)
	// etc.
}

// Helper function to check if we're in machine mode (no prompts, stable output)
func isMachineMode() bool {
	return jsonOutput || porcelain || noPrompt
}

// Helper function to check if prompts are allowed
func promptsAllowed() bool {
	return !noPrompt && !autoYes && !jsonOutput && !porcelain
}

// Placeholder for future implementation
func placeholder(cmd *cobra.Command, args []string) error {
	return fmt.Errorf("command '%s' not yet implemented", cmd.Name())
}
