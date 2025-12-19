package core_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/bmf/yagwt/internal/core"
)

// setupTestRepo creates a temporary git repository for testing
func setupTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "yagwt-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Initialize git repo
	if err := runCommand(tmpDir, "git", "init"); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user (required for commits)
	if err := runCommand(tmpDir, "git", "config", "user.name", "Test User"); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to config git user: %v", err)
	}
	if err := runCommand(tmpDir, "git", "config", "user.email", "test@example.com"); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to config git email: %v", err)
	}

	// Create initial commit
	readmePath := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test Repo\n"), 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to write README: %v", err)
	}

	if err := runCommand(tmpDir, "git", "add", "README.md"); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to add README: %v", err)
	}

	if err := runCommand(tmpDir, "git", "commit", "-m", "Initial commit"); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to commit: %v", err)
	}

	// Create a test branch
	if err := runCommand(tmpDir, "git", "branch", "feature-test"); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create branch: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func runCommand(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.Run()
}

func TestNewEngine(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Should create engine successfully
	engine, err := core.NewEngine(repoDir)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	if engine == nil {
		t.Fatal("NewEngine() returned nil engine")
	}
}

func TestNewEngineNonGitDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "yagwt-test-nogit-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Should fail on non-git directory
	_, err = core.NewEngine(tmpDir)
	if err == nil {
		t.Fatal("NewEngine() should fail on non-git directory")
	}
}

func TestListEmpty(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	engine, err := core.NewEngine(repoDir)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	// List should return primary workspace
	workspaces, err := engine.List(core.ListOptions{})
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	// Should have at least the primary workspace
	if len(workspaces) < 1 {
		t.Fatal("List() should return at least the primary workspace")
	}

	// First workspace should be primary
	if !workspaces[0].IsPrimary {
		t.Error("First workspace should be marked as primary")
	}
}

func TestCreateWorkspace(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	engine, err := core.NewEngine(repoDir)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	// Create workspace
	opts := core.CreateOptions{
		Target: "feature-test",
		Name:   "my-feature",
		Dir:    filepath.Join(repoDir, ".workspaces", "my-feature"),
	}

	ws, err := engine.Create(opts)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Verify workspace properties
	if ws.ID == "" {
		t.Error("Workspace should have an ID")
	}

	if ws.Name != "my-feature" {
		t.Errorf("Workspace name = %q, want %q", ws.Name, "my-feature")
	}

	if ws.IsPrimary {
		t.Error("Created workspace should not be primary")
	}

	// Verify workspace exists in filesystem
	if _, err := os.Stat(ws.Path); os.IsNotExist(err) {
		t.Errorf("Workspace path %q does not exist", ws.Path)
	}

	// Verify workspace appears in list
	workspaces, err := engine.List(core.ListOptions{})
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	found := false
	for _, w := range workspaces {
		if w.ID == ws.ID {
			found = true
			break
		}
	}

	if !found {
		t.Error("Created workspace not found in list")
	}
}

func TestCreateEphemeralWorkspace(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	engine, err := core.NewEngine(repoDir)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	// Create ephemeral workspace
	opts := core.CreateOptions{
		Target:    "feature-test",
		Name:      "ephemeral-ws",
		Dir:    filepath.Join(repoDir, ".workspaces", "ephemeral-ws"),
		Ephemeral: true,
		TTL:       7 * 24 * time.Hour,
	}

	ws, err := engine.Create(opts)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Verify ephemeral flag
	if !ws.Flags.Ephemeral {
		t.Error("Workspace should be ephemeral")
	}

	if ws.Ephemeral == nil {
		t.Fatal("Ephemeral info should not be nil")
	}

	if ws.Ephemeral.TTLSeconds != int((7*24*time.Hour).Seconds()) {
		t.Errorf("TTL = %d, want %d", ws.Ephemeral.TTLSeconds, int((7*24*time.Hour).Seconds()))
	}

	// ExpiresAt should be approximately 7 days from now
	expectedExpiry := time.Now().Add(7 * 24 * time.Hour)
	if ws.Ephemeral.ExpiresAt.Before(expectedExpiry.Add(-time.Minute)) ||
		ws.Ephemeral.ExpiresAt.After(expectedExpiry.Add(time.Minute)) {
		t.Errorf("ExpiresAt = %v, expected around %v", ws.Ephemeral.ExpiresAt, expectedExpiry)
	}
}

func TestGetWorkspace(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	engine, err := core.NewEngine(repoDir)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	// Create workspace
	created, err := engine.Create(core.CreateOptions{
		Target: "feature-test",
		Name:   "test-workspace",
		Dir:    filepath.Join(repoDir, ".workspaces", "test-workspace"),
	})
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Get by ID
	ws, err := engine.Get(core.Selector{Type: core.SelectorID, Value: created.ID})
	if err != nil {
		t.Fatalf("Get() by ID failed: %v", err)
	}

	if ws.ID != created.ID {
		t.Errorf("Get() returned wrong workspace: got %q, want %q", ws.ID, created.ID)
	}

	// Get by name
	ws, err = engine.Get(core.Selector{Type: core.SelectorName, Value: "test-workspace"})
	if err != nil {
		t.Fatalf("Get() by name failed: %v", err)
	}

	if ws.ID != created.ID {
		t.Errorf("Get() returned wrong workspace: got %q, want %q", ws.ID, created.ID)
	}
}

func TestGetNonExistentWorkspace(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	engine, err := core.NewEngine(repoDir)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	// Should return not found error
	_, err = engine.Get(core.Selector{Type: core.SelectorName, Value: "does-not-exist"})
	if err == nil {
		t.Fatal("Get() should fail for non-existent workspace")
	}

	coreErr, ok := err.(*core.Error)
	if !ok {
		t.Fatalf("Expected *core.Error, got %T", err)
	}

	if coreErr.Code != core.ErrNotFound {
		t.Errorf("Expected error code %s, got %s", core.ErrNotFound, coreErr.Code)
	}
}

func TestResolveWorkspaces(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	engine, err := core.NewEngine(repoDir)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	// Create two workspaces on the same branch
	ws1, err := engine.Create(core.CreateOptions{
		Target: "feature-test",
		Name:   "workspace-1",
		Dir:    filepath.Join(repoDir, ".workspaces", "workspace-1"),
	})
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Resolve by branch should return at least one workspace
	matches, err := engine.Resolve("branch:feature-test")
	if err != nil {
		t.Fatalf("Resolve() failed: %v", err)
	}

	if len(matches) < 1 {
		t.Fatal("Resolve() should find at least one workspace on branch")
	}

	// Verify the created workspace is in the matches
	found := false
	for _, ws := range matches {
		if ws.ID == ws1.ID {
			found = true
			break
		}
	}

	if !found {
		t.Error("Resolve() should include created workspace")
	}
}

func TestRemoveWorkspace(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	engine, err := core.NewEngine(repoDir)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	// Create workspace
	ws, err := engine.Create(core.CreateOptions{
		Target: "feature-test",
		Name:   "to-remove",
		Dir:    filepath.Join(repoDir, ".workspaces", "to-remove"),
	})
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	wsPath := ws.Path

	// Remove workspace
	err = engine.Remove(
		core.Selector{Type: core.SelectorID, Value: ws.ID},
		core.RemoveOptions{},
	)
	if err != nil {
		t.Fatalf("Remove() failed: %v", err)
	}

	// Verify workspace path is gone
	if _, err := os.Stat(wsPath); !os.IsNotExist(err) {
		t.Errorf("Workspace path %q should be removed", wsPath)
	}

	// Verify workspace is not in list
	workspaces, err := engine.List(core.ListOptions{})
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	for _, w := range workspaces {
		if w.ID == ws.ID {
			t.Error("Removed workspace should not appear in list")
		}
	}
}

func TestRemovePinnedWorkspace(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	engine, err := core.NewEngine(repoDir)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	// Create pinned workspace
	ws, err := engine.Create(core.CreateOptions{
		Target: "feature-test",
		Name:   "pinned-ws",
		Dir:    filepath.Join(repoDir, ".workspaces", "pinned-ws"),
		Pin:    true,
	})
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Should fail to remove pinned workspace
	err = engine.Remove(
		core.Selector{Type: core.SelectorID, Value: ws.ID},
		core.RemoveOptions{},
	)
	if err == nil {
		t.Fatal("Remove() should fail for pinned workspace")
	}

	coreErr, ok := err.(*core.Error)
	if !ok {
		t.Fatalf("Expected *core.Error, got %T", err)
	}

	if coreErr.Code != core.ErrLocked {
		t.Errorf("Expected error code %s, got %s", core.ErrLocked, coreErr.Code)
	}
}

func TestPinUnpinWorkspace(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	engine, err := core.NewEngine(repoDir)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	// Create workspace
	ws, err := engine.Create(core.CreateOptions{
		Target: "feature-test",
		Name:   "test-pin",
		Dir:    filepath.Join(repoDir, ".workspaces", "test-pin"),
	})
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Pin workspace
	err = engine.Pin(core.Selector{Type: core.SelectorID, Value: ws.ID})
	if err != nil {
		t.Fatalf("Pin() failed: %v", err)
	}

	// Verify pinned
	ws, err = engine.Get(core.Selector{Type: core.SelectorID, Value: ws.ID})
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if !ws.Flags.Pinned {
		t.Error("Workspace should be pinned")
	}

	// Unpin workspace
	err = engine.Unpin(core.Selector{Type: core.SelectorID, Value: ws.ID})
	if err != nil {
		t.Fatalf("Unpin() failed: %v", err)
	}

	// Verify unpinned
	ws, err = engine.Get(core.Selector{Type: core.SelectorID, Value: ws.ID})
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if ws.Flags.Pinned {
		t.Error("Workspace should be unpinned")
	}
}

func TestRenameWorkspace(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	engine, err := core.NewEngine(repoDir)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	// Create workspace
	ws, err := engine.Create(core.CreateOptions{
		Target: "feature-test",
		Name:   "old-name",
		Dir:    filepath.Join(repoDir, ".workspaces", "old-name"),
	})
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Rename workspace
	err = engine.Rename(
		core.Selector{Type: core.SelectorID, Value: ws.ID},
		"new-name",
	)
	if err != nil {
		t.Fatalf("Rename() failed: %v", err)
	}

	// Verify new name
	ws, err = engine.Get(core.Selector{Type: core.SelectorID, Value: ws.ID})
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if ws.Name != "new-name" {
		t.Errorf("Workspace name = %q, want %q", ws.Name, "new-name")
	}

	// Should be findable by new name
	ws, err = engine.Get(core.Selector{Type: core.SelectorName, Value: "new-name"})
	if err != nil {
		t.Fatalf("Get() by new name failed: %v", err)
	}

	// Should not be findable by old name
	_, err = engine.Get(core.Selector{Type: core.SelectorName, Value: "old-name"})
	if err == nil {
		t.Fatal("Get() by old name should fail")
	}
}

func TestLockUnlockWorkspace(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	engine, err := core.NewEngine(repoDir)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	// Create workspace
	ws, err := engine.Create(core.CreateOptions{
		Target: "feature-test",
		Name:   "test-lock",
		Dir:    filepath.Join(repoDir, ".workspaces", "test-lock"),
	})
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Lock workspace
	err = engine.Lock(core.Selector{Type: core.SelectorID, Value: ws.ID})
	if err != nil {
		t.Fatalf("Lock() failed: %v", err)
	}

	// Verify locked
	ws, err = engine.Get(core.Selector{Type: core.SelectorID, Value: ws.ID})
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if !ws.Flags.Locked {
		t.Error("Workspace should be locked")
	}

	// Unlock workspace
	err = engine.Unlock(core.Selector{Type: core.SelectorID, Value: ws.ID})
	if err != nil {
		t.Fatalf("Unlock() failed: %v", err)
	}

	// Verify unlocked
	ws, err = engine.Get(core.Selector{Type: core.SelectorID, Value: ws.ID})
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if ws.Flags.Locked {
		t.Error("Workspace should be unlocked")
	}
}
