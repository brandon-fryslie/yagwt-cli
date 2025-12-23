package core

import (
	"testing"
	"time"
)

func TestWorkspaceCreation(t *testing.T) {
	// Basic test to verify Workspace structure
	ws := Workspace{
		ID:        "wsp_test123",
		Name:      "test-workspace",
		Path:      "/path/to/workspace",
		IsPrimary: false,
		Target: Target{
			Type:    "branch",
			Ref:     "refs/heads/main",
			Short:   "main",
			HeadSHA: "abc123",
		},
		Flags: WorkspaceFlags{
			Pinned:    false,
			Ephemeral: true,
			Locked:    false,
			Broken:    false,
		},
		Ephemeral: &EphemeralInfo{
			TTLSeconds: 604800,
			ExpiresAt:  time.Now().Add(7 * 24 * time.Hour),
		},
	}

	if ws.ID != "wsp_test123" {
		t.Errorf("Expected ID 'wsp_test123', got '%s'", ws.ID)
	}

	if ws.Name != "test-workspace" {
		t.Errorf("Expected name 'test-workspace', got '%s'", ws.Name)
	}

	if !ws.Flags.Ephemeral {
		t.Error("Expected workspace to be ephemeral")
	}

	if ws.Ephemeral == nil {
		t.Error("Expected ephemeral info to be set")
	}
}

func TestSelectorParsing(t *testing.T) {
	// Test basic selector parsing
	selector := ParseSelector("test-workspace")

	if selector.Type != SelectorBare {
		t.Errorf("Expected SelectorBare, got %v", selector.Type)
	}

	if selector.Value != "test-workspace" {
		t.Errorf("Expected value 'test-workspace', got '%s'", selector.Value)
	}
}

func TestErrorCreation(t *testing.T) {
	// Test error creation with hints
	err := NewError(ErrDirty, "Workspace has uncommitted changes").
		WithDetail("files", 3).
		WithHint("To stash changes", "yagwt rm --on-dirty=stash")

	if err.Code != ErrDirty {
		t.Errorf("Expected code E_DIRTY, got %s", err.Code)
	}

	if len(err.Hints) != 1 {
		t.Errorf("Expected 1 hint, got %d", len(err.Hints))
	}

	if err.ExitCode() != 3 {
		t.Errorf("Expected exit code 3, got %d", err.ExitCode())
	}
}
