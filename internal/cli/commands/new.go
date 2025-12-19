package commands

import (
	"time"

	"github.com/bmf/yagwt/internal/core"
	"github.com/spf13/cobra"
)

var (
	newName       string
	newDir        string
	newBase       string
	newNewBranch  bool
	newDetach     bool
	newEphemeral  bool
	newTTL        string
	newPin        bool
	newNoCheckout bool
)

var newCmd = &cobra.Command{
	Use:   "new <target>",
	Short: "Create a new workspace",
	Long: `Create a new workspace for a branch or commit.

The target can be:
  - An existing branch name (e.g., feature/auth)
  - A commit SHA
  - A new branch name (with --new-branch)

Examples:
  yagwt new feature/auth
  yagwt new feature/auth --name auth
  yagwt new main --name temp --ephemeral --ttl 7d
  yagwt new feature/new --new-branch --base main
  yagwt new abc123 --detach --name commit-test`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		initFormatter()

		// Initialize engine
		if err := initEngine(); err != nil {
			handleError(err)
		}

		// Parse TTL duration if provided
		var ttl time.Duration
		if newTTL != "" {
			var err error
			ttl, err = time.ParseDuration(newTTL)
			if err != nil {
				handleError(err)
			}
		}

		// Build create options
		opts := core.CreateOptions{
			Target:    args[0],
			Name:      newName,
			Dir:       newDir,
			Base:      newBase,
			NewBranch: newNewBranch,
			Detached:  newDetach,
			Ephemeral: newEphemeral,
			TTL:       ttl,
			Pin:       newPin,
			Checkout:  !newNoCheckout,
		}

		// Create workspace
		workspace, err := engine.Create(opts)
		if err != nil {
			handleError(err)
		}

		// Format and print output
		output := formatter.FormatWorkspace(workspace)
		printOutput(output)

		// Print success message if not in quiet mode
		if !quiet && !jsonOutput && !porcelain {
			printOutput(formatter.FormatSuccess("Workspace created successfully"))
		}
	},
}

func init() {
	newCmd.Flags().StringVarP(&newName, "name", "n", "", "workspace name (default: derived from target)")
	newCmd.Flags().StringVarP(&newDir, "dir", "d", "", "directory path (default: derived from config)")
	newCmd.Flags().StringVarP(&newBase, "base", "b", "", "base branch for new branch")
	newCmd.Flags().BoolVar(&newNewBranch, "new-branch", false, "create new branch from target")
	newCmd.Flags().BoolVar(&newDetach, "detach", false, "create detached HEAD")
	newCmd.Flags().BoolVar(&newEphemeral, "ephemeral", false, "mark as ephemeral")
	newCmd.Flags().StringVar(&newTTL, "ttl", "", "TTL for ephemeral (e.g., '7d', '24h')")
	newCmd.Flags().BoolVar(&newPin, "pin", false, "pin immediately")
	newCmd.Flags().BoolVar(&newNoCheckout, "no-checkout", false, "don't checkout after creation")
}
