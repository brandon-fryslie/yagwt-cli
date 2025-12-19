package git

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bmf/yagwt/internal/errors"
)

// Repository provides git operations
type Repository interface {
	// Worktree operations
	ListWorktrees() ([]Worktree, error)
	AddWorktree(path, ref string, opts AddOptions) error
	RemoveWorktree(path string, force bool) error

	// Status operations
	GetStatus(path string) (Status, error)

	// Reference operations
	ResolveRef(ref string) (string, error) // Returns full SHA
	GetBranch(ref string) (Branch, error)

	// Dirty workspace operations
	Stash(path, message string) error
	CreatePatch(path, patchFile string) error
	CreateWIPCommit(path, message string) error

	// Repository info
	Root() string
	GitDir() string
}

// Worktree represents a git worktree
type Worktree struct {
	Path     string
	HEAD     string // SHA
	Branch   string // Empty if detached
	Locked   bool
	Prunable bool
}

// Status represents git status output
type Status struct {
	Dirty     bool
	Conflicts bool
	Branch    string
	Detached  bool
	Ahead     int
	Behind    int
}

// Branch represents a git branch
type Branch struct {
	Name     string
	Upstream string
	HEAD     string // SHA
}

// AddOptions specifies options for adding a worktree
type AddOptions struct {
	NewBranch bool
	Detach    bool
	Force     bool
	Checkout  bool
	Track     string // Upstream branch for --track
}

// repo implements Repository interface
type repo struct {
	root   string
	gitDir string
}

// NewRepository creates a new Repository for the given path
func NewRepository(path string) (Repository, error) {
	// Find the repository root
	cmd := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "not a git repository") {
				return nil, errors.NewError(errors.ErrGit, "not a git repository").
					WithDetail("path", path).
					WithHint("Initialize a git repository first", "git init")
			}
		}
		return nil, errors.WrapError(errors.ErrGit, "failed to find git repository", err).
			WithDetail("path", path)
	}

	root := strings.TrimSpace(string(output))

	// Find the git directory
	cmd = exec.Command("git", "-C", root, "rev-parse", "--git-dir")
	output, err = cmd.Output()
	if err != nil {
		return nil, errors.WrapError(errors.ErrGit, "failed to find git directory", err).
			WithDetail("root", root)
	}

	gitDir := strings.TrimSpace(string(output))
	// Make gitDir absolute if it's relative
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(root, gitDir)
	}

	return &repo{
		root:   root,
		gitDir: gitDir,
	}, nil
}

// ListWorktrees lists all worktrees
func (r *repo) ListWorktrees() ([]Worktree, error) {
	cmd := exec.Command("git", "-C", r.root, "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.WrapError(errors.ErrGit, "failed to list worktrees", err)
	}

	return parseWorktreeList(output)
}

// parseWorktreeList parses the porcelain output of git worktree list
func parseWorktreeList(output []byte) ([]Worktree, error) {
	var worktrees []Worktree
	var current *Worktree

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// Empty line separates worktrees
			if current != nil {
				worktrees = append(worktrees, *current)
				current = nil
			}
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 1 {
			continue
		}

		key := parts[0]
		value := ""
		if len(parts) > 1 {
			value = parts[1]
		}

		switch key {
		case "worktree":
			current = &Worktree{
				Path: value,
			}
		case "HEAD":
			if current != nil {
				current.HEAD = value
			}
		case "branch":
			if current != nil {
				// Branch ref is refs/heads/branch-name
				current.Branch = strings.TrimPrefix(value, "refs/heads/")
			}
		case "detached":
			// Detached HEAD (no branch)
			if current != nil {
				current.Branch = ""
			}
		case "locked":
			if current != nil {
				current.Locked = true
			}
		case "prunable":
			if current != nil {
				current.Prunable = true
			}
		}
	}

	// Add the last worktree if present
	if current != nil {
		worktrees = append(worktrees, *current)
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.WrapError(errors.ErrGit, "failed to parse worktree list", err)
	}

	return worktrees, nil
}

// AddWorktree creates a new worktree
func (r *repo) AddWorktree(path, ref string, opts AddOptions) error {
	args := []string{"-C", r.root, "worktree", "add"}

	if opts.Force {
		args = append(args, "--force")
	}

	if opts.Detach {
		args = append(args, "--detach")
	}

	if opts.NewBranch && opts.Track != "" {
		args = append(args, "--track", opts.Track)
	}

	args = append(args, path)

	if ref != "" {
		args = append(args, ref)
	}

	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		return errors.WrapError(errors.ErrGit, "failed to add worktree", err).
			WithDetail("path", path).
			WithDetail("ref", ref).
			WithDetail("stderr", errMsg)
	}

	return nil
}

// RemoveWorktree removes a worktree
func (r *repo) RemoveWorktree(path string, force bool) error {
	args := []string{"-C", r.root, "worktree", "remove"}

	if force {
		args = append(args, "--force")
	}

	args = append(args, path)

	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()

		// Check for specific error conditions
		if strings.Contains(errMsg, "is locked") {
			return errors.NewError(errors.ErrLocked, "worktree is locked").
				WithDetail("path", path).
				WithHint("Use --force to remove locked worktree", "")
		}

		if strings.Contains(errMsg, "contains modified or untracked files") {
			return errors.NewError(errors.ErrDirty, "worktree contains uncommitted changes").
				WithDetail("path", path).
				WithHint("Commit or stash changes, or use --force", "git -C "+path+" status")
		}

		return errors.WrapError(errors.ErrGit, "failed to remove worktree", err).
			WithDetail("path", path).
			WithDetail("stderr", errMsg)
	}

	return nil
}

// GetStatus returns the git status for a path
func (r *repo) GetStatus(path string) (Status, error) {
	cmd := exec.Command("git", "-C", path, "status", "--porcelain=v2", "--branch")
	output, err := cmd.Output()
	if err != nil {
		return Status{}, errors.WrapError(errors.ErrGit, "failed to get status", err).
			WithDetail("path", path)
	}

	return parseStatusV2(output)
}

// parseStatusV2 parses git status --porcelain=v2 output
func parseStatusV2(output []byte) (Status, error) {
	var status Status

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "# branch.head ") {
			head := strings.TrimPrefix(line, "# branch.head ")
			if head == "(detached)" {
				status.Detached = true
			} else {
				status.Branch = head
			}
		} else if strings.HasPrefix(line, "# branch.ab ") {
			// Parse ahead/behind: "# branch.ab +2 -3"
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				ahead := strings.TrimPrefix(parts[2], "+")
				behind := strings.TrimPrefix(parts[3], "-")

				if a, err := strconv.Atoi(ahead); err == nil {
					status.Ahead = a
				}
				if b, err := strconv.Atoi(behind); err == nil {
					status.Behind = b
				}
			}
		} else if strings.HasPrefix(line, "1 ") || strings.HasPrefix(line, "2 ") {
			// Modified file: "1 .M N... ..."
			status.Dirty = true
		} else if strings.HasPrefix(line, "? ") {
			// Untracked file
			status.Dirty = true
		} else if strings.HasPrefix(line, "u ") {
			// Unmerged (conflict)
			status.Conflicts = true
			status.Dirty = true
		}
	}

	if err := scanner.Err(); err != nil {
		return Status{}, errors.WrapError(errors.ErrGit, "failed to parse status output", err)
	}

	return status, nil
}

// ResolveRef resolves a ref to a full SHA
func (r *repo) ResolveRef(ref string) (string, error) {
	cmd := exec.Command("git", "-C", r.root, "rev-parse", "--verify", ref)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "unknown revision") ||
				strings.Contains(stderr, "bad revision") ||
				strings.Contains(stderr, "not a valid") {
				return "", errors.NewError(errors.ErrNotFound, "ref not found").
					WithDetail("ref", ref)
			}
		}
		return "", errors.WrapError(errors.ErrGit, "failed to resolve ref", err).
			WithDetail("ref", ref)
	}

	sha := strings.TrimSpace(string(output))
	return sha, nil
}

// GetBranch returns information about a branch
func (r *repo) GetBranch(ref string) (Branch, error) {
	// Resolve the ref to get HEAD
	head, err := r.ResolveRef(ref)
	if err != nil {
		return Branch{}, err
	}

	// Get branch info using for-each-ref
	cmd := exec.Command("git", "-C", r.root, "for-each-ref",
		"--format=%(refname:short)\t%(upstream:short)\t%(objectname)",
		"refs/heads/"+ref)
	output, err := cmd.Output()
	if err != nil {
		return Branch{}, errors.WrapError(errors.ErrGit, "failed to get branch info", err).
			WithDetail("ref", ref)
	}

	line := strings.TrimSpace(string(output))
	if line == "" {
		// Not a branch, just return HEAD
		return Branch{
			Name: ref,
			HEAD: head,
		}, nil
	}

	parts := strings.Split(line, "\t")
	branch := Branch{
		Name: parts[0],
		HEAD: head,
	}

	if len(parts) > 1 && parts[1] != "" {
		branch.Upstream = parts[1]
	}

	return branch, nil
}

// Stash creates a stash with a message
func (r *repo) Stash(path, message string) error {
	cmd := exec.Command("git", "-C", path, "stash", "push", "-m", message)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		return errors.WrapError(errors.ErrGit, "failed to stash changes", err).
			WithDetail("path", path).
			WithDetail("message", message).
			WithDetail("stderr", errMsg)
	}

	return nil
}

// CreatePatch creates a patch file with all uncommitted changes
func (r *repo) CreatePatch(path, patchFile string) error {
	// Create patch directory if it doesn't exist
	patchDir := filepath.Dir(patchFile)
	if err := os.MkdirAll(patchDir, 0755); err != nil {
		return errors.WrapError(errors.ErrGit, "failed to create patch directory", err).
			WithDetail("dir", patchDir)
	}

	// Create the patch file
	file, err := os.Create(patchFile)
	if err != nil {
		return errors.WrapError(errors.ErrGit, "failed to create patch file", err).
			WithDetail("file", patchFile)
	}
	defer file.Close()

	// Get diff for unstaged changes
	cmd := exec.Command("git", "-C", path, "diff", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return errors.WrapError(errors.ErrGit, "failed to generate diff", err).
			WithDetail("path", path)
	}

	if _, err := file.Write(output); err != nil {
		return errors.WrapError(errors.ErrGit, "failed to write patch", err).
			WithDetail("file", patchFile)
	}

	return nil
}

// CreateWIPCommit creates a WIP commit with all changes
func (r *repo) CreateWIPCommit(path, message string) error {
	// Add all changes
	cmd := exec.Command("git", "-C", path, "add", "-A")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		return errors.WrapError(errors.ErrGit, "failed to add changes", err).
			WithDetail("path", path).
			WithDetail("stderr", errMsg)
	}

	// Create commit
	cmd = exec.Command("git", "-C", path, "commit", "-m", message)
	stderr.Reset()
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		return errors.WrapError(errors.ErrGit, "failed to create WIP commit", err).
			WithDetail("path", path).
			WithDetail("message", message).
			WithDetail("stderr", errMsg)
	}

	return nil
}

// Root returns the repository root
func (r *repo) Root() string {
	return r.root
}

// GitDir returns the git directory
func (r *repo) GitDir() string {
	return r.gitDir
}
