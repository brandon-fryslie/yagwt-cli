package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmf/yagwt/internal/core"
)

// setupTestRepo creates a temporary git repository for testing
func setupTestRepo(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "test-repo")

	// Initialize git repo
	if err := os.Mkdir(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create repo dir: %v", err)
	}

	runGit(t, repoDir, "init")
	runGit(t, repoDir, "config", "user.email", "test@example.com")
	runGit(t, repoDir, "config", "user.name", "Test User")

	// Create initial commit
	writeFile(t, filepath.Join(repoDir, "README.md"), "# Test Repo\n")
	runGit(t, repoDir, "add", "README.md")
	runGit(t, repoDir, "commit", "-m", "Initial commit")

	// Resolve symlinks to get canonical path (handles /var vs /private/var on macOS)
	resolved, err := filepath.EvalSymlinks(repoDir)
	if err != nil {
		t.Fatalf("Failed to resolve symlinks: %v", err)
	}

	return resolved
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Git command failed: %v\nOutput: %s", err, output)
	}
	return string(output)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
}

// pathsEqual compares two paths, resolving symlinks
func pathsEqual(t *testing.T, a, b string) bool {
	t.Helper()
	aResolved, err := filepath.EvalSymlinks(a)
	if err != nil {
		aResolved = a
	}
	bResolved, err := filepath.EvalSymlinks(b)
	if err != nil {
		bResolved = b
	}
	return aResolved == bResolved
}

func TestNewRepository(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Test from repo root
	repo, err := NewRepository(repoDir)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	if !pathsEqual(t, repo.Root(), repoDir) {
		t.Errorf("Expected root %q, got %q", repoDir, repo.Root())
	}

	expectedGitDir := filepath.Join(repoDir, ".git")
	if !pathsEqual(t, repo.GitDir(), expectedGitDir) {
		t.Errorf("Expected gitDir %q, got %q", expectedGitDir, repo.GitDir())
	}

	// Test from subdirectory
	subDir := filepath.Join(repoDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	repo2, err := NewRepository(subDir)
	if err != nil {
		t.Fatalf("Failed to create repository from subdir: %v", err)
	}

	if !pathsEqual(t, repo2.Root(), repoDir) {
		t.Errorf("Expected root %q from subdir, got %q", repoDir, repo2.Root())
	}
}

func TestNewRepository_NotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := NewRepository(tmpDir)
	if err == nil {
		t.Fatal("Expected error when creating repository in non-git directory")
	}

	coreErr, ok := err.(*core.Error)
	if !ok {
		t.Fatalf("Expected *core.Error, got %T", err)
	}

	if coreErr.Code != core.ErrGit {
		t.Errorf("Expected error code %s, got %s", core.ErrGit, coreErr.Code)
	}
}

func TestResolveRef(t *testing.T) {
	repoDir := setupTestRepo(t)
	repo, _ := NewRepository(repoDir)

	// Create a branch
	runGit(t, repoDir, "branch", "test-branch")

	// Resolve HEAD
	sha, err := repo.ResolveRef("HEAD")
	if err != nil {
		t.Fatalf("Failed to resolve HEAD: %v", err)
	}

	if len(sha) != 40 {
		t.Errorf("Expected 40-character SHA, got %q", sha)
	}

	// Resolve branch
	branchSha, err := repo.ResolveRef("test-branch")
	if err != nil {
		t.Fatalf("Failed to resolve branch: %v", err)
	}

	if branchSha != sha {
		t.Errorf("Expected branch SHA to match HEAD, got %q vs %q", branchSha, sha)
	}

	// Try to resolve non-existent ref
	_, err = repo.ResolveRef("non-existent-branch")
	if err == nil {
		t.Fatal("Expected error when resolving non-existent ref")
	}

	coreErr, ok := err.(*core.Error)
	if !ok {
		t.Fatalf("Expected *core.Error, got %T", err)
	}

	// Accept both E_NOT_FOUND and E_GIT for non-existent refs
	if coreErr.Code != core.ErrNotFound && coreErr.Code != core.ErrGit {
		t.Errorf("Expected error code %s or %s, got %s", core.ErrNotFound, core.ErrGit, coreErr.Code)
	}
}

func TestListWorktrees(t *testing.T) {
	repoDir := setupTestRepo(t)
	repo, _ := NewRepository(repoDir)

	// Initially, should have only the main worktree
	worktrees, err := repo.ListWorktrees()
	if err != nil {
		t.Fatalf("Failed to list worktrees: %v", err)
	}

	if len(worktrees) != 1 {
		t.Fatalf("Expected 1 worktree, got %d", len(worktrees))
	}

	if !pathsEqual(t, worktrees[0].Path, repoDir) {
		t.Errorf("Expected worktree path %q, got %q", repoDir, worktrees[0].Path)
	}

	if worktrees[0].Branch != "master" && worktrees[0].Branch != "main" {
		t.Errorf("Expected branch 'master' or 'main', got %q", worktrees[0].Branch)
	}
}

func TestAddRemoveWorktree(t *testing.T) {
	repoDir := setupTestRepo(t)
	repo, _ := NewRepository(repoDir)

	// Create a new branch
	runGit(t, repoDir, "branch", "feature-branch")

	// Add worktree
	wtDir := filepath.Join(t.TempDir(), "feature-wt")
	// Resolve the worktree directory path
	wtDirResolved, _ := filepath.EvalSymlinks(filepath.Dir(wtDir))
	wtDirResolved = filepath.Join(wtDirResolved, filepath.Base(wtDir))

	err := repo.AddWorktree(wtDirResolved, "feature-branch", AddOptions{})
	if err != nil {
		t.Fatalf("Failed to add worktree: %v", err)
	}

	// Verify worktree exists
	if _, err := os.Stat(wtDirResolved); os.IsNotExist(err) {
		t.Error("Worktree directory was not created")
	}

	// List worktrees
	worktrees, err := repo.ListWorktrees()
	if err != nil {
		t.Fatalf("Failed to list worktrees: %v", err)
	}

	if len(worktrees) != 2 {
		t.Fatalf("Expected 2 worktrees, got %d", len(worktrees))
	}

	// Find our worktree
	var found bool
	for _, wt := range worktrees {
		if pathsEqual(t, wt.Path, wtDirResolved) {
			found = true
			if wt.Branch != "feature-branch" {
				t.Errorf("Expected branch 'feature-branch', got %q", wt.Branch)
			}
		}
	}

	if !found {
		t.Errorf("Added worktree not found in list. Looking for %q", wtDirResolved)
		for i, wt := range worktrees {
			t.Logf("Worktree %d: %q (branch: %q)", i, wt.Path, wt.Branch)
		}
	}

	// Remove worktree
	err = repo.RemoveWorktree(wtDirResolved, false)
	if err != nil {
		t.Fatalf("Failed to remove worktree: %v", err)
	}

	// Verify removed
	worktrees, err = repo.ListWorktrees()
	if err != nil {
		t.Fatalf("Failed to list worktrees: %v", err)
	}

	if len(worktrees) != 1 {
		t.Fatalf("Expected 1 worktree after removal, got %d", len(worktrees))
	}
}

func TestAddWorktree_Detached(t *testing.T) {
	repoDir := setupTestRepo(t)
	repo, _ := NewRepository(repoDir)

	// Get current commit SHA
	sha, _ := repo.ResolveRef("HEAD")

	// Add detached worktree
	wtDir := filepath.Join(t.TempDir(), "detached-wt")
	// Resolve the worktree directory path
	wtDirResolved, _ := filepath.EvalSymlinks(filepath.Dir(wtDir))
	wtDirResolved = filepath.Join(wtDirResolved, filepath.Base(wtDir))

	err := repo.AddWorktree(wtDirResolved, sha, AddOptions{Detach: true})
	if err != nil {
		t.Fatalf("Failed to add detached worktree: %v", err)
	}

	// List worktrees
	worktrees, err := repo.ListWorktrees()
	if err != nil {
		t.Fatalf("Failed to list worktrees: %v", err)
	}

	// Find our worktree
	var found bool
	for _, wt := range worktrees {
		if pathsEqual(t, wt.Path, wtDirResolved) {
			found = true
			if wt.Branch != "" {
				t.Errorf("Expected detached worktree (empty branch), got %q", wt.Branch)
			}
			if wt.HEAD != sha {
				t.Errorf("Expected HEAD %q, got %q", sha, wt.HEAD)
			}
		}
	}

	if !found {
		t.Errorf("Detached worktree not found in list. Looking for %q", wtDirResolved)
		for i, wt := range worktrees {
			t.Logf("Worktree %d: %q (branch: %q, HEAD: %q)", i, wt.Path, wt.Branch, wt.HEAD)
		}
	}
}

func TestGetStatus(t *testing.T) {
	repoDir := setupTestRepo(t)
	repo, _ := NewRepository(repoDir)

	// Test clean status
	status, err := repo.GetStatus(repoDir)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if status.Dirty {
		t.Error("Expected clean repository, got dirty")
	}

	if status.Conflicts {
		t.Error("Expected no conflicts")
	}

	// Make repository dirty
	writeFile(t, filepath.Join(repoDir, "new-file.txt"), "test\n")

	status, err = repo.GetStatus(repoDir)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if !status.Dirty {
		t.Error("Expected dirty repository, got clean")
	}

	// Test modified file
	writeFile(t, filepath.Join(repoDir, "README.md"), "# Modified\n")

	status, err = repo.GetStatus(repoDir)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if !status.Dirty {
		t.Error("Expected dirty repository after modification")
	}
}

func TestRemoveWorktree_Dirty(t *testing.T) {
	repoDir := setupTestRepo(t)
	repo, _ := NewRepository(repoDir)

	// Create a branch and worktree
	runGit(t, repoDir, "branch", "test-branch")
	wtDir := filepath.Join(t.TempDir(), "test-wt")
	wtDirResolved, _ := filepath.EvalSymlinks(filepath.Dir(wtDir))
	wtDirResolved = filepath.Join(wtDirResolved, filepath.Base(wtDir))

	repo.AddWorktree(wtDirResolved, "test-branch", AddOptions{})

	// Make the worktree dirty
	writeFile(t, filepath.Join(wtDirResolved, "dirty.txt"), "dirty\n")

	// Try to remove (should fail)
	err := repo.RemoveWorktree(wtDirResolved, false)
	if err == nil {
		t.Fatal("Expected error when removing dirty worktree")
	}

	coreErr, ok := err.(*core.Error)
	if !ok {
		t.Fatalf("Expected *core.Error, got %T", err)
	}

	if coreErr.Code != core.ErrDirty {
		t.Errorf("Expected error code %s, got %s", core.ErrDirty, coreErr.Code)
	}

	// Force remove should work
	err = repo.RemoveWorktree(wtDirResolved, true)
	if err != nil {
		t.Fatalf("Failed to force remove dirty worktree: %v", err)
	}
}

func TestGetBranch(t *testing.T) {
	repoDir := setupTestRepo(t)
	repo, _ := NewRepository(repoDir)

	// Get current branch (master or main)
	defaultBranch := strings.TrimSpace(runGit(t, repoDir, "branch", "--show-current"))

	branch, err := repo.GetBranch(defaultBranch)
	if err != nil {
		t.Fatalf("Failed to get branch: %v", err)
	}

	if branch.Name != defaultBranch {
		t.Errorf("Expected branch name %q, got %q", defaultBranch, branch.Name)
	}

	if len(branch.HEAD) != 40 {
		t.Errorf("Expected 40-character SHA for HEAD, got %q", branch.HEAD)
	}

	// Create a branch with upstream
	runGit(t, repoDir, "checkout", "-b", "feature")
	runGit(t, repoDir, "branch", "--set-upstream-to="+defaultBranch)

	branch, err = repo.GetBranch("feature")
	if err != nil {
		t.Fatalf("Failed to get feature branch: %v", err)
	}

	if branch.Name != "feature" {
		t.Errorf("Expected branch name 'feature', got %q", branch.Name)
	}

	// Upstream might be set
	// Note: In a fresh test repo without remotes, upstream tracking might not work as expected
	// so we just verify the branch info was retrieved
}
