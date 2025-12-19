package cleanup

import (
	"testing"
	"time"

	"github.com/bmf/yagwt/internal/core"
)

// Test helper to create test workspaces
func makeTestWorkspace(opts map[string]interface{}) core.Workspace {
	ws := core.Workspace{
		ID:   "test-id",
		Name: "test-workspace",
		Path: "/test/path",
		Target: core.Target{
			Type:  "branch",
			Short: "main",
		},
		Flags: core.WorkspaceFlags{},
		Status: core.StatusInfo{
			Dirty: false,
		},
		Activity: core.ActivityInfo{},
	}

	// Apply options
	if id, ok := opts["id"].(string); ok {
		ws.ID = id
	}
	if name, ok := opts["name"].(string); ok {
		ws.Name = name
	}
	if pinned, ok := opts["pinned"].(bool); ok {
		ws.Flags.Pinned = pinned
	}
	if ephemeral, ok := opts["ephemeral"].(bool); ok {
		ws.Flags.Ephemeral = ephemeral
	}
	if locked, ok := opts["locked"].(bool); ok {
		ws.Flags.Locked = locked
	}
	if dirty, ok := opts["dirty"].(bool); ok {
		ws.Status.Dirty = dirty
	}
	if conflicts, ok := opts["conflicts"].(bool); ok {
		ws.Status.Conflicts = conflicts
	}
	if ahead, ok := opts["ahead"].(int); ok {
		ws.Status.Ahead = ahead
	}
	if lastActivity, ok := opts["lastActivity"].(time.Time); ok {
		ws.Activity.LastGitActivityAt = &lastActivity
	}
	if expiresAt, ok := opts["expiresAt"].(time.Time); ok {
		ws.Ephemeral = &core.EphemeralInfo{
			TTLSeconds: 3600,
			ExpiresAt:  expiresAt,
		}
	}

	return ws
}

func TestGetPolicy(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"default", "default"},
		{"conservative", "conservative"},
		{"aggressive", "aggressive"},
		{"unknown", "default"}, // Falls back to default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := GetPolicy(tt.name)
			if policy.Name() != tt.expected {
				t.Errorf("GetPolicy(%q).Name() = %q, want %q", tt.name, policy.Name(), tt.expected)
			}
		})
	}
}

func TestDefaultPolicyExpiredEphemeral(t *testing.T) {
	policy := &DefaultPolicy{}
	now := time.Now()

	tests := []struct {
		name         string
		ws           core.Workspace
		shouldRemove bool
		reasonCode   string
	}{
		{
			name:         "expired ephemeral",
			ws:           makeTestWorkspace(map[string]interface{}{"ephemeral": true, "expiresAt": now.Add(-1 * time.Hour)}),
			shouldRemove: true,
			reasonCode:   "expired_ephemeral",
		},
		{
			name:         "not expired ephemeral",
			ws:           makeTestWorkspace(map[string]interface{}{"ephemeral": true, "expiresAt": now.Add(1 * time.Hour)}),
			shouldRemove: false,
		},
		{
			name:         "non-ephemeral",
			ws:           makeTestWorkspace(map[string]interface{}{}),
			shouldRemove: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, remove := policy.Evaluate(tt.ws)
			if remove != tt.shouldRemove {
				t.Errorf("Evaluate() remove = %v, want %v", remove, tt.shouldRemove)
			}
			if remove && reason.Code != tt.reasonCode {
				t.Errorf("Evaluate() reason.Code = %q, want %q", reason.Code, tt.reasonCode)
			}
		})
	}
}

func TestDefaultPolicyIdle30Days(t *testing.T) {
	policy := &DefaultPolicy{}
	now := time.Now()

	tests := []struct {
		name         string
		ws           core.Workspace
		shouldRemove bool
		reasonCode   string
	}{
		{
			name:         "idle >30 days clean",
			ws:           makeTestWorkspace(map[string]interface{}{"lastActivity": now.Add(-40 * 24 * time.Hour)}),
			shouldRemove: true,
			reasonCode:   "idle_30d",
		},
		{
			name:         "idle >30 days but dirty",
			ws:           makeTestWorkspace(map[string]interface{}{"lastActivity": now.Add(-40 * 24 * time.Hour), "dirty": true}),
			shouldRemove: false,
		},
		{
			name:         "idle <30 days",
			ws:           makeTestWorkspace(map[string]interface{}{"lastActivity": now.Add(-10 * 24 * time.Hour)}),
			shouldRemove: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, remove := policy.Evaluate(tt.ws)
			if remove != tt.shouldRemove {
				t.Errorf("Evaluate() remove = %v, want %v", remove, tt.shouldRemove)
			}
			if remove && reason.Code != tt.reasonCode {
				t.Errorf("Evaluate() reason.Code = %q, want %q", reason.Code, tt.reasonCode)
			}
		})
	}
}

func TestDefaultPolicySkipsPinned(t *testing.T) {
	policy := &DefaultPolicy{}
	now := time.Now()

	ws := makeTestWorkspace(map[string]interface{}{
		"pinned":       true,
		"lastActivity": now.Add(-40 * 24 * time.Hour),
	})

	_, remove := policy.Evaluate(ws)
	if remove {
		t.Error("DefaultPolicy should skip pinned workspaces")
	}
}

func TestDefaultPolicySkipsLocked(t *testing.T) {
	policy := &DefaultPolicy{}
	now := time.Now()

	ws := makeTestWorkspace(map[string]interface{}{
		"locked":       true,
		"lastActivity": now.Add(-40 * 24 * time.Hour),
	})

	_, remove := policy.Evaluate(ws)
	if remove {
		t.Error("DefaultPolicy should skip locked workspaces")
	}
}

func TestConservativePolicyOnlyExpired(t *testing.T) {
	policy := &ConservativePolicy{}
	now := time.Now()

	tests := []struct {
		name         string
		ws           core.Workspace
		shouldRemove bool
	}{
		{
			name:         "expired ephemeral",
			ws:           makeTestWorkspace(map[string]interface{}{"ephemeral": true, "expiresAt": now.Add(-1 * time.Hour)}),
			shouldRemove: true,
		},
		{
			name:         "idle 40 days",
			ws:           makeTestWorkspace(map[string]interface{}{"lastActivity": now.Add(-40 * 24 * time.Hour)}),
			shouldRemove: false,
		},
		{
			name:         "idle 10 days",
			ws:           makeTestWorkspace(map[string]interface{}{"lastActivity": now.Add(-10 * 24 * time.Hour)}),
			shouldRemove: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, remove := policy.Evaluate(tt.ws)
			if remove != tt.shouldRemove {
				t.Errorf("Evaluate() remove = %v, want %v", remove, tt.shouldRemove)
			}
		})
	}
}

func TestAggressivePolicyIdle7Days(t *testing.T) {
	policy := &AggressivePolicy{}
	now := time.Now()

	tests := []struct {
		name         string
		ws           core.Workspace
		shouldRemove bool
		reasonCode   string
	}{
		{
			name:         "idle 10 days",
			ws:           makeTestWorkspace(map[string]interface{}{"lastActivity": now.Add(-10 * 24 * time.Hour)}),
			shouldRemove: true,
			reasonCode:   "idle_7d",
		},
		{
			name:         "idle 5 days",
			ws:           makeTestWorkspace(map[string]interface{}{"lastActivity": now.Add(-5 * 24 * time.Hour)}),
			shouldRemove: false,
		},
		{
			name:         "expired ephemeral",
			ws:           makeTestWorkspace(map[string]interface{}{"ephemeral": true, "expiresAt": now.Add(-1 * time.Hour)}),
			shouldRemove: true,
			reasonCode:   "expired_ephemeral",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, remove := policy.Evaluate(tt.ws)
			if remove != tt.shouldRemove {
				t.Errorf("Evaluate() remove = %v, want %v", remove, tt.shouldRemove)
			}
			if remove && reason.Code != tt.reasonCode {
				t.Errorf("Evaluate() reason.Code = %q, want %q", reason.Code, tt.reasonCode)
			}
		})
	}
}

func TestGeneratePlan(t *testing.T) {
	now := time.Now()
	policy := &DefaultPolicy{}

	workspaces := []core.Workspace{
		makeTestWorkspace(map[string]interface{}{
			"id":           "ws1",
			"name":         "expired",
			"ephemeral":    true,
			"expiresAt":    now.Add(-1 * time.Hour),
		}),
		makeTestWorkspace(map[string]interface{}{
			"id":           "ws2",
			"name":         "idle",
			"lastActivity": now.Add(-40 * 24 * time.Hour),
		}),
		makeTestWorkspace(map[string]interface{}{
			"id":           "ws3",
			"name":         "pinned",
			"pinned":       true,
			"lastActivity": now.Add(-40 * 24 * time.Hour),
		}),
		makeTestWorkspace(map[string]interface{}{
			"id":           "ws4",
			"name":         "recent",
			"lastActivity": now.Add(-5 * 24 * time.Hour),
		}),
	}

	plan := GeneratePlan(workspaces, policy)

	// Should have 2 actions (expired and idle, not pinned or recent)
	if len(plan.Actions) != 2 {
		t.Errorf("GeneratePlan() actions count = %d, want 2", len(plan.Actions))
	}

	// Verify expired is first (higher priority)
	if len(plan.Actions) > 0 && plan.Actions[0].Reason.Code != "expired_ephemeral" {
		t.Errorf("First action should be expired_ephemeral, got %q", plan.Actions[0].Reason.Code)
	}

	// Verify idle is second
	if len(plan.Actions) > 1 && plan.Actions[1].Reason.Code != "idle_30d" {
		t.Errorf("Second action should be idle_30d, got %q", plan.Actions[1].Reason.Code)
	}
}

func TestGeneratePlanWarnings(t *testing.T) {
	now := time.Now()
	policy := &DefaultPolicy{}

	workspaces := []core.Workspace{
		makeTestWorkspace(map[string]interface{}{
			"id":           "ws1",
			"name":         "dirty-idle",
			"lastActivity": now.Add(-40 * 24 * time.Hour),
			"dirty":        true,
		}),
		makeTestWorkspace(map[string]interface{}{
			"id":           "ws2",
			"name":         "conflicts-expired",
			"ephemeral":    true,
			"expiresAt":    now.Add(-1 * time.Hour),
			"conflicts":    true,
		}),
		makeTestWorkspace(map[string]interface{}{
			"id":           "ws3",
			"name":         "unpushed-expired",
			"ephemeral":    true,
			"expiresAt":    now.Add(-1 * time.Hour),
			"ahead":        3,
		}),
	}

	plan := GeneratePlan(workspaces, policy)

	// Should have warnings for dirty, conflicts, and unpushed
	// Note: ws1 is dirty so won't be removed by default policy
	// ws2 and ws3 will be removed (expired ephemeral)

	if len(plan.Warnings) < 2 {
		t.Errorf("GeneratePlan() should generate warnings, got %d", len(plan.Warnings))
	}

	// Check for specific warning types
	hasConflictsWarning := false
	hasUnpushedWarning := false

	for _, w := range plan.Warnings {
		if w.Code == "conflicts" {
			hasConflictsWarning = true
		}
		if w.Code == "unpushed_commits" {
			hasUnpushedWarning = true
		}
	}

	if !hasConflictsWarning {
		t.Error("GeneratePlan() should warn about conflicts")
	}
	if !hasUnpushedWarning {
		t.Error("GeneratePlan() should warn about unpushed commits")
	}
}

func TestSortActions(t *testing.T) {
	actions := []RemovalAction{
		{
			Workspace: makeTestWorkspace(map[string]interface{}{"name": "idle7"}),
			Reason:    RemovalReason{Code: "idle_7d"},
		},
		{
			Workspace: makeTestWorkspace(map[string]interface{}{"name": "expired"}),
			Reason:    RemovalReason{Code: "expired_ephemeral"},
		},
		{
			Workspace: makeTestWorkspace(map[string]interface{}{"name": "idle30"}),
			Reason:    RemovalReason{Code: "idle_30d"},
		},
	}

	sorted := sortActions(actions)

	// Verify order: expired, idle30d, idle7d
	if len(sorted) != 3 {
		t.Fatalf("Expected 3 actions, got %d", len(sorted))
	}

	if sorted[0].Reason.Code != "expired_ephemeral" {
		t.Errorf("First action should be expired_ephemeral, got %q", sorted[0].Reason.Code)
	}
	if sorted[1].Reason.Code != "idle_30d" {
		t.Errorf("Second action should be idle_30d, got %q", sorted[1].Reason.Code)
	}
	if sorted[2].Reason.Code != "idle_7d" {
		t.Errorf("Third action should be idle_7d, got %q", sorted[2].Reason.Code)
	}
}

func TestGeneratePlanEmptyWorkspaces(t *testing.T) {
	policy := &DefaultPolicy{}
	plan := GeneratePlan([]core.Workspace{}, policy)

	if len(plan.Actions) != 0 {
		t.Errorf("GeneratePlan() with empty workspaces should have 0 actions, got %d", len(plan.Actions))
	}
	if len(plan.Warnings) != 0 {
		t.Errorf("GeneratePlan() with empty workspaces should have 0 warnings, got %d", len(plan.Warnings))
	}
}

func TestGeneratePlanAllPinned(t *testing.T) {
	now := time.Now()
	policy := &DefaultPolicy{}

	workspaces := []core.Workspace{
		makeTestWorkspace(map[string]interface{}{
			"pinned":       true,
			"lastActivity": now.Add(-40 * 24 * time.Hour),
		}),
		makeTestWorkspace(map[string]interface{}{
			"pinned":    true,
			"ephemeral": true,
			"expiresAt": now.Add(-1 * time.Hour),
		}),
	}

	plan := GeneratePlan(workspaces, policy)

	if len(plan.Actions) != 0 {
		t.Errorf("GeneratePlan() with all pinned should have 0 actions, got %d", len(plan.Actions))
	}
}
