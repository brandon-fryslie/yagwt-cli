package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/bmf/yagwt/internal/cli/output"
	"github.com/bmf/yagwt/internal/core"
	"github.com/bmf/yagwt/internal/errors"
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

	// Shared formatter and engine (initialized per-command)
	formatter output.Formatter
	engine    core.WorkspaceManager
)

// Version information (set at build time)
var (
	Version   = "1.0.0-dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// Exit codes
const (
	ExitSuccess         = 0
	ExitFailure         = 1
	ExitInvalidUsage    = 2
	ExitSafetyRefusal   = 3
	ExitPartialSuccess  = 4
	ExitNotFound        = 5
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "yagwt",
	Short: "Yet Another Git Worktree Manager",
	Long: `YAGWT is a powerful CLI tool for managing git worktrees with lifecycle management,
metadata tracking, and intelligent cleanup policies.

Use 'yagwt <command> --help' for information on a specific command.`,
	Version:       Version,
	SilenceErrors: true, // We handle errors ourselves
	SilenceUsage:  true, // Don't show usage on errors
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

// defaultUsageFunc stores the default usage function
var defaultUsageFunc func(*cobra.Command) error

func init() {
	// Global flags available to all commands
	rootCmd.PersistentFlags().StringVarP(&repoPath, "repo", "r", "", "repository root (default: auto-detect from current directory)")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format (machine-readable)")
	rootCmd.PersistentFlags().BoolVar(&porcelain, "porcelain", false, "output in stable porcelain format (tab-separated)")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress non-essential output")
	rootCmd.PersistentFlags().BoolVar(&noPrompt, "no-prompt", false, "never ask questions (fail if input required)")
	rootCmd.PersistentFlags().BoolVarP(&autoYes, "yes", "y", false, "automatically answer yes to all prompts")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(pathCmd)
	rootCmd.AddCommand(resolveCmd)
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(rmCmd)
	rootCmd.AddCommand(renameCmd)
	rootCmd.AddCommand(pinCmd)
	rootCmd.AddCommand(unpinCmd)
	rootCmd.AddCommand(lockCmd)
	rootCmd.AddCommand(unlockCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(doctorCmd)

	// Store the default usage function before we override it
	defaultUsageFunc = rootCmd.UsageFunc()

	// Set custom usage function to show command arguments with proper alignment
	rootCmd.SetUsageFunc(customUsageFunc)
}

// customUsageFunc returns a custom usage output with proper alignment for command arguments
func customUsageFunc(cmd *cobra.Command) error {
	// Only customize for the root command itself
	if cmd != rootCmd {
		// For all other commands, use default behavior
		return defaultUsageFunc(cmd)
	}

	// Calculate maximum Use length for proper padding
	maxUseLen := 0
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() && c.Name() != "help" {
			continue
		}
		if len(c.Use) > maxUseLen {
			maxUseLen = len(c.Use)
		}
	}

	// Build the usage output (Long description is printed by help template, not here)
	fmt.Fprintf(cmd.OutOrStdout(), "Usage:\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  %s [command]\n\n", cmd.CommandPath())

	// Commands section
	fmt.Fprintf(cmd.OutOrStdout(), "Available Commands:\n")
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() && c.Name() != "help" {
			continue
		}
		padding := strings.Repeat(" ", maxUseLen-len(c.Use))
		fmt.Fprintf(cmd.OutOrStdout(), "  %s%s  %s\n", c.Use, padding, c.Short)
	}

	// Flags section
	if cmd.HasAvailableLocalFlags() {
		fmt.Fprintf(cmd.OutOrStdout(), "\nFlags:\n")
		fmt.Fprintf(cmd.OutOrStdout(), "%s", cmd.LocalFlags().FlagUsages())
	}

	if cmd.HasAvailableInheritedFlags() {
		fmt.Fprintf(cmd.OutOrStdout(), "\nGlobal Flags:\n")
		fmt.Fprintf(cmd.OutOrStdout(), "%s", cmd.InheritedFlags().FlagUsages())
	}

	// Footer
	fmt.Fprintf(cmd.OutOrStdout(), "\nUse \"%s [command] --help\" for more information about a command.\n", cmd.CommandPath())

	return nil
}

// initFormatter initializes the formatter based on flags
func initFormatter() {
	mode := output.ModeHuman
	if jsonOutput {
		mode = output.ModeJSON
	} else if porcelain {
		mode = output.ModePorcelain
	}
	formatter = output.NewFormatter(mode, quiet)
}

// initEngine initializes the core engine from the repo path
func initEngine() error {
	path := repoPath
	if path == "" {
		// Use current directory
		var err error
		path, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	var err error
	engine, err = core.NewEngine(path)
	if err != nil {
		return err
	}

	return nil
}

// handleError formats and prints an error, then exits with appropriate code
func handleError(err error) {
	if err == nil {
		return
	}

	initFormatter()
	fmt.Fprint(os.Stderr, formatter.FormatError(err))

	// Determine exit code from error type
	exitCode := ExitFailure
	if yerr, ok := err.(*errors.Error); ok {
		switch yerr.Code {
		case errors.ErrNotFound:
			exitCode = ExitNotFound
		case errors.ErrAmbiguous:
			exitCode = ExitInvalidUsage
		case errors.ErrDirty, errors.ErrLocked:
			exitCode = ExitSafetyRefusal
		default:
			exitCode = ExitFailure
		}
	}

	os.Exit(exitCode)
}

// printOutput prints formatted output to stdout
func printOutput(output string) {
	fmt.Print(output)
}

// Helper function to check if we're in machine mode (no prompts, stable output)
func isMachineMode() bool {
	return jsonOutput || porcelain || noPrompt
}

// Helper function to check if prompts are allowed
func promptsAllowed() bool {
	return !noPrompt && !autoYes && !jsonOutput && !porcelain
}
