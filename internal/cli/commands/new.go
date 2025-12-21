package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bmf/yagwt/internal/core"
	"github.com/spf13/cobra"
)

var (
	newName        string
	newDir         string
	newBase        string
	newNewBranch   bool
	newDetach      bool
	newEphemeral   bool
	newTTL         string
	newPin         bool
	newNoCheckout  bool
	newInteractive bool
)

var newCmd = &cobra.Command{
	Use:   "new [target]",
	Short: "Create a new worktree",
	Long: `Create a new git worktree.

Without arguments, runs in interactive mode to guide you through the process.

The target can be:
  - An existing branch name (e.g., feature/auth)
  - A commit SHA
  - A new branch name (with --new-branch)

Examples:
  yagwt new                              # Interactive mode
  yagwt new feature/auth                 # Checkout existing branch
  yagwt new feature/auth --name auth     # With custom name
  yagwt new my-feature --new-branch -b main  # Create new branch from main
  yagwt new abc123 --detach              # Detached HEAD at commit`,
	Args: cobra.MaximumNArgs(1),
	Run:  runNew,
}

func init() {
	newCmd.Flags().StringVarP(&newName, "name", "n", "", "worktree name (default: derived from target)")
	newCmd.Flags().StringVarP(&newDir, "dir", "d", "", "directory path (default: sibling to repo)")
	newCmd.Flags().StringVarP(&newBase, "base", "b", "", "base branch for new branch")
	newCmd.Flags().BoolVar(&newNewBranch, "new-branch", false, "create new branch")
	newCmd.Flags().BoolVar(&newDetach, "detach", false, "create detached HEAD")
	newCmd.Flags().BoolVar(&newEphemeral, "ephemeral", false, "mark as ephemeral (auto-cleanup eligible)")
	newCmd.Flags().StringVar(&newTTL, "ttl", "", "time-to-live for ephemeral (e.g., '7d', '24h')")
	newCmd.Flags().BoolVar(&newPin, "pin", false, "pin to prevent cleanup")
	newCmd.Flags().BoolVar(&newNoCheckout, "no-checkout", false, "don't checkout after creation")
	newCmd.Flags().BoolVarP(&newInteractive, "interactive", "i", false, "run in interactive mode")
}

func runNew(cmd *cobra.Command, args []string) {
	initFormatter()

	if err := initEngine(); err != nil {
		handleError(err)
	}

	// Interactive mode if no args or -i flag
	if len(args) == 0 || newInteractive {
		runNewInteractive()
		return
	}

	// Non-interactive mode
	var ttl time.Duration
	if newTTL != "" {
		var err error
		ttl, err = parseDuration(newTTL)
		if err != nil {
			handleError(err)
		}
	}

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

	worktree, err := engine.Create(opts)
	if err != nil {
		handleError(err)
	}

	output := formatter.FormatWorkspace(worktree)
	printOutput(output)

	if !quiet && !jsonOutput && !porcelain {
		printOutput(formatter.FormatSuccess("Worktree created successfully"))
	}
}

func runNewInteractive() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("Create a new git worktree")
	fmt.Println("─────────────────────────")
	fmt.Println()

	// Step 1: What do you want to work on?
	fmt.Println("What do you want to checkout?")
	fmt.Println("  1. An existing branch")
	fmt.Println("  2. Create a new branch")
	fmt.Println("  3. A specific commit (detached)")
	fmt.Println()

	choice := promptInt(reader, "Choose", 2)

	var target, base, name string
	var isNewBranch, isDetached bool

	switch choice {
	case 1:
		// Existing branch
		branches := listBranches()
		if len(branches) > 0 {
			fmt.Println()
			fmt.Println("Available branches:")
			for i, b := range branches {
				if i < 10 {
					fmt.Printf("  %s\n", b)
				}
			}
			if len(branches) > 10 {
				fmt.Printf("  ... and %d more\n", len(branches)-10)
			}
			fmt.Println()
		}
		target = promptString(reader, "Branch name", "")
		if target == "" {
			fmt.Println("Cancelled.")
			return
		}

	case 2:
		// New branch
		isNewBranch = true
		target = promptString(reader, "New branch name", "")
		if target == "" {
			fmt.Println("Cancelled.")
			return
		}
		base = promptString(reader, "Base branch", getCurrentBranch())

	case 3:
		// Detached commit
		isDetached = true
		target = promptString(reader, "Commit SHA or ref", "HEAD")

	default:
		fmt.Println("Invalid choice.")
		return
	}

	// Step 2: Name for the worktree
	fmt.Println()
	defaultName := sanitizeName(target)
	name = promptString(reader, "Worktree name", defaultName)

	// Step 3: Directory (show default)
	fmt.Println()
	repoRoot := getRepoRoot()
	parentDir := filepath.Dir(repoRoot)
	defaultDir := filepath.Join(parentDir, name)
	fmt.Printf("Directory: %s\n", defaultDir)
	customDir := promptString(reader, "Custom directory (Enter to use default)", "")

	dir := defaultDir
	if customDir != "" {
		dir = customDir
	}

	// Step 4: Options
	fmt.Println()
	ephemeral := promptConfirm(reader, "Mark as ephemeral (auto-cleanup eligible)?", false)

	var ttl time.Duration
	if ephemeral {
		ttlStr := promptString(reader, "Time-to-live (e.g., 7d, 24h)", "7d")
		ttl, _ = parseDuration(ttlStr)
	}

	// Confirm
	fmt.Println()
	fmt.Println("─────────────────────────")
	fmt.Println("Summary:")
	if isNewBranch {
		fmt.Printf("  Create branch: %s (from %s)\n", target, base)
	} else if isDetached {
		fmt.Printf("  Checkout: %s (detached)\n", target)
	} else {
		fmt.Printf("  Checkout: %s\n", target)
	}
	fmt.Printf("  Name: %s\n", name)
	fmt.Printf("  Directory: %s\n", dir)
	if ephemeral {
		fmt.Printf("  Ephemeral: yes (TTL: %s)\n", ttl)
	}
	fmt.Println()

	if !promptConfirm(reader, "Create worktree?", true) {
		fmt.Println("Cancelled.")
		return
	}

	// Create it
	fmt.Println()
	opts := core.CreateOptions{
		Target:    target,
		Name:      name,
		Dir:       dir,
		Base:      base,
		NewBranch: isNewBranch,
		Detached:  isDetached,
		Ephemeral: ephemeral,
		TTL:       ttl,
		Checkout:  true,
	}

	worktree, err := engine.Create(opts)
	if err != nil {
		handleError(err)
	}

	fmt.Printf("Created worktree: %s\n", worktree.Path)
	fmt.Println()
	fmt.Println("To start working:")
	fmt.Printf("  cd %s\n", worktree.Path)
}

// Helper functions

func promptString(reader *bufio.Reader, prompt string, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultVal)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultVal
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func promptInt(reader *bufio.Reader, prompt string, defaultVal int) int {
	fmt.Printf("%s [%d]: ", prompt, defaultVal)

	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultVal
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}

	val, err := strconv.Atoi(input)
	if err != nil {
		return defaultVal
	}
	return val
}

func promptConfirm(reader *bufio.Reader, prompt string, defaultVal bool) bool {
	defaultStr := "y/N"
	if defaultVal {
		defaultStr = "Y/n"
	}
	fmt.Printf("%s [%s]: ", prompt, defaultStr)

	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultVal
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultVal
	}

	return input == "y" || input == "yes"
}

func sanitizeName(name string) string {
	// Remove refs/heads/ prefix if present
	name = strings.TrimPrefix(name, "refs/heads/")
	// Replace slashes with hyphens
	name = strings.ReplaceAll(name, "/", "-")
	return name
}

func getCurrentBranch() string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "main"
	}
	return strings.TrimSpace(string(output))
}

func getRepoRoot() string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "."
	}
	return strings.TrimSpace(string(output))
}

func listBranches() []string {
	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var branches []string
	for _, line := range lines {
		if line != "" {
			branches = append(branches, line)
		}
	}
	return branches
}

func parseDuration(s string) (time.Duration, error) {
	// Handle day suffix
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}
