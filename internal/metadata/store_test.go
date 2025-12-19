package metadata

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bmf/yagwt/internal/errors"
	"github.com/google/uuid"
)

func TestNewStore(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")

	store, err := NewStore(gitDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Verify directory was created
	storeDir := filepath.Join(gitDir, "yagwt")
	if _, err := os.Stat(storeDir); os.IsNotExist(err) {
		t.Error("Store directory was not created")
	}

	// Load should return empty metadata
	metadata, err := store.Load()
	if err != nil {
		t.Fatalf("Failed to load empty metadata: %v", err)
	}

	if metadata.SchemaVersion != 1 {
		t.Errorf("Expected schema version 1, got %d", metadata.SchemaVersion)
	}

	if len(metadata.Workspaces) != 0 {
		t.Errorf("Expected empty workspaces, got %d", len(metadata.Workspaces))
	}
}

func TestSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	store, _ := NewStore(gitDir)

	// Create metadata
	now := time.Now()
	metadata := Metadata{
		SchemaVersion: 1,
		Workspaces: map[string]WorkspaceMetadata{
			"ws1": {
				ID:        "ws1",
				Name:      "feature-a",
				Path:      "/path/to/feature-a",
				Flags:     map[string]bool{"pinned": true},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		Index: Index{
			ByPath: map[string]string{
				"/path/to/feature-a": "ws1",
			},
			ByName: map[string]string{
				"feature-a": "ws1",
			},
			ByBranch: make(map[string][]string),
		},
	}

	// Save
	err := store.Save(metadata)
	if err != nil {
		t.Fatalf("Failed to save metadata: %v", err)
	}

	// Load
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	// Verify
	if loaded.SchemaVersion != 1 {
		t.Errorf("Expected schema version 1, got %d", loaded.SchemaVersion)
	}

	if len(loaded.Workspaces) != 1 {
		t.Fatalf("Expected 1 workspace, got %d", len(loaded.Workspaces))
	}

	ws := loaded.Workspaces["ws1"]
	if ws.Name != "feature-a" {
		t.Errorf("Expected name 'feature-a', got %q", ws.Name)
	}

	if ws.Path != "/path/to/feature-a" {
		t.Errorf("Expected path '/path/to/feature-a', got %q", ws.Path)
	}

	if !ws.Flags["pinned"] {
		t.Error("Expected pinned flag to be true")
	}
}

func TestSetGet(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	store, _ := NewStore(gitDir)

	id := uuid.New().String()
	now := time.Now()

	workspace := WorkspaceMetadata{
		ID:        id,
		Name:      "test-workspace",
		Path:      "/tmp/test-workspace",
		Flags:     make(map[string]bool),
		CreatedAt: now,
	}

	// Set
	err := store.Set(id, workspace)
	if err != nil {
		t.Fatalf("Failed to set workspace: %v", err)
	}

	// Get
	retrieved, err := store.Get(id)
	if err != nil {
		t.Fatalf("Failed to get workspace: %v", err)
	}

	if retrieved.ID != id {
		t.Errorf("Expected ID %q, got %q", id, retrieved.ID)
	}

	if retrieved.Name != "test-workspace" {
		t.Errorf("Expected name 'test-workspace', got %q", retrieved.Name)
	}

	// UpdatedAt should be set
	if retrieved.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}
}

func TestFindByName(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	store, _ := NewStore(gitDir)

	id := uuid.New().String()

	workspace := WorkspaceMetadata{
		ID:        id,
		Name:      "my-feature",
		Path:      "/tmp/my-feature",
		Flags:     make(map[string]bool),
		CreatedAt: time.Now(),
	}

	store.Set(id, workspace)

	// Find by name
	found, err := store.FindByName("my-feature")
	if err != nil {
		t.Fatalf("Failed to find workspace by name: %v", err)
	}

	if found.ID != id {
		t.Errorf("Expected ID %q, got %q", id, found.ID)
	}

	// Try non-existent name
	_, err = store.FindByName("non-existent")
	if err == nil {
		t.Fatal("Expected error for non-existent workspace")
	}

	coreErr, ok := err.(*errors.Error)
	if !ok {
		t.Fatalf("Expected *errors.Error, got %T", err)
	}

	if coreErr.Code != errors.ErrNotFound {
		t.Errorf("Expected error code %s, got %s", errors.ErrNotFound, coreErr.Code)
	}
}

func TestFindByPath(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	store, _ := NewStore(gitDir)

	id := uuid.New().String()
	path := "/tmp/test-path"

	workspace := WorkspaceMetadata{
		ID:        id,
		Name:      "test",
		Path:      path,
		Flags:     make(map[string]bool),
		CreatedAt: time.Now(),
	}

	store.Set(id, workspace)

	// Find by path
	found, err := store.FindByPath(path)
	if err != nil {
		t.Fatalf("Failed to find workspace by path: %v", err)
	}

	if found.ID != id {
		t.Errorf("Expected ID %q, got %q", id, found.ID)
	}

	// Try non-existent path
	_, err = store.FindByPath("/non/existent/path")
	if err == nil {
		t.Fatal("Expected error for non-existent workspace")
	}

	coreErr, ok := err.(*errors.Error)
	if !ok {
		t.Fatalf("Expected *errors.Error, got %T", err)
	}

	if coreErr.Code != errors.ErrNotFound {
		t.Errorf("Expected error code %s, got %s", errors.ErrNotFound, coreErr.Code)
	}
}

func TestDelete(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	store, _ := NewStore(gitDir)

	id := uuid.New().String()

	workspace := WorkspaceMetadata{
		ID:        id,
		Name:      "to-delete",
		Path:      "/tmp/to-delete",
		Flags:     make(map[string]bool),
		CreatedAt: time.Now(),
	}

	store.Set(id, workspace)

	// Verify it exists
	_, err := store.Get(id)
	if err != nil {
		t.Fatalf("Workspace should exist before delete: %v", err)
	}

	// Delete
	err = store.Delete(id)
	if err != nil {
		t.Fatalf("Failed to delete workspace: %v", err)
	}

	// Verify it's gone
	_, err = store.Get(id)
	if err == nil {
		t.Fatal("Workspace should not exist after delete")
	}

	// Verify indexes are cleaned up
	_, err = store.FindByName("to-delete")
	if err == nil {
		t.Error("Name index should be cleaned up after delete")
	}

	_, err = store.FindByPath("/tmp/to-delete")
	if err == nil {
		t.Error("Path index should be cleaned up after delete")
	}
}

func TestAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	store, _ := NewStore(gitDir)

	metadata := Metadata{
		SchemaVersion: 1,
		Workspaces:    make(map[string]WorkspaceMetadata),
		Index: Index{
			ByPath:   make(map[string]string),
			ByName:   make(map[string]string),
			ByBranch: make(map[string][]string),
		},
	}

	// Save
	err := store.Save(metadata)
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Verify temp file was cleaned up
	tmpPath := filepath.Join(gitDir, "yagwt", "meta.json.tmp")
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("Temp file should be cleaned up after save")
	}

	// Verify metadata can be reloaded
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Failed to reload saved metadata: %v", err)
	}

	if loaded.SchemaVersion != 1 {
		t.Errorf("Expected schema version 1 after reload, got %d", loaded.SchemaVersion)
	}
}

func TestRebuildIndex(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	store, _ := NewStore(gitDir)

	// Create workspaces
	ws1 := WorkspaceMetadata{
		ID:        "ws1",
		Name:      "workspace-1",
		Path:      "/tmp/ws1",
		Flags:     make(map[string]bool),
		CreatedAt: time.Now(),
	}

	ws2 := WorkspaceMetadata{
		ID:        "ws2",
		Name:      "workspace-2",
		Path:      "/tmp/ws2",
		Flags:     make(map[string]bool),
		CreatedAt: time.Now(),
	}

	store.Set("ws1", ws1)
	store.Set("ws2", ws2)

	// Corrupt the index manually
	metadata, _ := store.Load()
	metadata.Index = Index{
		ByPath:   map[string]string{},
		ByName:   map[string]string{},
		ByBranch: map[string][]string{},
	}
	store.Save(metadata)

	// Rebuild index
	err := store.RebuildIndex()
	if err != nil {
		t.Fatalf("Failed to rebuild index: %v", err)
	}

	// Verify index is correct
	found, err := store.FindByName("workspace-1")
	if err != nil {
		t.Errorf("Should find workspace-1 after rebuild: %v", err)
	}
	if found.ID != "ws1" {
		t.Errorf("Expected ID ws1, got %q", found.ID)
	}

	found, err = store.FindByPath("/tmp/ws2")
	if err != nil {
		t.Errorf("Should find workspace at /tmp/ws2 after rebuild: %v", err)
	}
	if found.ID != "ws2" {
		t.Errorf("Expected ID ws2, got %q", found.ID)
	}
}

func TestMultipleWorkspaces(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	store, _ := NewStore(gitDir)

	// Create multiple workspaces
	for i := 0; i < 10; i++ {
		id := uuid.New().String()
		ws := WorkspaceMetadata{
			ID:        id,
			Name:      "workspace-" + id,
			Path:      "/tmp/" + id,
			Flags:     make(map[string]bool),
			CreatedAt: time.Now(),
		}
		if err := store.Set(id, ws); err != nil {
			t.Fatalf("Failed to set workspace %d: %v", i, err)
		}
	}

	// Load and verify
	metadata, err := store.Load()
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	if len(metadata.Workspaces) != 10 {
		t.Errorf("Expected 10 workspaces, got %d", len(metadata.Workspaces))
	}

	if len(metadata.Index.ByPath) != 10 {
		t.Errorf("Expected 10 entries in path index, got %d", len(metadata.Index.ByPath))
	}

	if len(metadata.Index.ByName) != 10 {
		t.Errorf("Expected 10 entries in name index, got %d", len(metadata.Index.ByName))
	}
}

func TestEphemeralMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	store, _ := NewStore(gitDir)

	id := uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour)

	workspace := WorkspaceMetadata{
		ID:   id,
		Name: "ephemeral-ws",
		Path: "/tmp/ephemeral",
		Ephemeral: &EphemeralMetadata{
			TTLSeconds: 86400,
			ExpiresAt:  expiresAt,
		},
		Flags:     make(map[string]bool),
		CreatedAt: time.Now(),
	}

	store.Set(id, workspace)

	// Retrieve and verify
	retrieved, err := store.Get(id)
	if err != nil {
		t.Fatalf("Failed to get workspace: %v", err)
	}

	if retrieved.Ephemeral == nil {
		t.Fatal("Expected ephemeral metadata to be set")
	}

	if retrieved.Ephemeral.TTLSeconds != 86400 {
		t.Errorf("Expected TTL 86400, got %d", retrieved.Ephemeral.TTLSeconds)
	}

	// Verify time is roughly correct (allow for serialization rounding)
	if retrieved.Ephemeral.ExpiresAt.Unix() != expiresAt.Unix() {
		t.Errorf("Expected ExpiresAt %v, got %v", expiresAt, retrieved.Ephemeral.ExpiresAt)
	}
}

func TestCorruptedMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	metaPath := filepath.Join(gitDir, "yagwt", "meta.json")

	// Create store directory
	os.MkdirAll(filepath.Dir(metaPath), 0755)

	// Write corrupted JSON
	os.WriteFile(metaPath, []byte("{invalid json"), 0644)

	store, _ := NewStore(gitDir)

	// Load should fail
	_, err := store.Load()
	if err == nil {
		t.Fatal("Expected error when loading corrupted metadata")
	}

	coreErr, ok := err.(*errors.Error)
	if !ok {
		t.Fatalf("Expected *errors.Error, got %T", err)
	}

	if coreErr.Code != errors.ErrConfig {
		t.Errorf("Expected error code %s, got %s", errors.ErrConfig, coreErr.Code)
	}
}
