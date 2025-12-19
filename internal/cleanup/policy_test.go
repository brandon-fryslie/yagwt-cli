package cleanup

import (
	"testing"
	"time"
)

// mockFlags implements Flags interface for testing
type mockFlags struct {
	pinned    bool
	locked    bool
	ephemeral bool
}

func (f *mockFlags) IsPinned() bool    { return f.pinned }
func (f *mockFlags) IsLocked() bool    { return f.locked }
func (f *mockFlags) IsEphemeral() bool { return f.ephemeral }

// mockActivity implements Activity interface for testing
type mockActivity struct {
	lastGitActivityAt *time.Time
}

func (a *mockActivity) GetLastGitActivityAt() *time.Time {
	return a.lastGitActivityAt
}

// mockStatus implements Status interface for testing
type mockStatus struct {
	dirty     bool
	conflicts bool
	ahead     int
}

func (s *mockStatus) IsDirty() bool     { return s.dirty }
func (s *mockStatus) HasConflicts() bool { return s.conflicts }
func (s *mockStatus) GetAhead() int     { return s.ahead }

// mockWorkspace implements Workspace interface for testing
type mockWorkspace struct {
	id        string
	name      string
	flags     *mockFlags
	ephemeral *EphemeralInfo
	activity  *mockActivity
	status    *mockStatus
}

func (w *mockWorkspace) GetID() string                { return w.id }
func (w *mockWorkspace) GetName() string              { return w.name }
func (w *mockWorkspace) GetFlags() Flags              { return w.flags }
func (w *mockWorkspace) GetEphemeral() *EphemeralInfo { return w.ephemeral }
func (w *mockWorkspace) GetActivity() Activity        { return w.activity }
func (w *mockWorkspace) GetStatus() Status            { return w.status }

func timePtr(t time.Time) *time.Time {
	return &t
}

func TestGetPolicy(t *testing.T) {
	tests := []struct {
		name       string
		policyName string
		wantName   string
	}{
		{
			name:       "default policy",
			policyName: "default",
			wantName:   "default",
		},
		{
			name:       "conservative policy",
			policyName: "conservative",
			wantName:   "conservative",
		},
		{
			name:       "aggressive policy",
			policyName: "aggressive",
			wantName:   "aggressive",
		},
		{
			name:       "unknown returns default",
			policyName: "unknown",
			wantName:   "default",
		},
		{
			name:       "empty returns default",
			policyName: "",
			wantName:   "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := GetPolicy(tt.policyName)
			if policy.Name() != tt.wantName {
				t.Errorf("GetPolicy(%q).Name() = %q, want %q", tt.policyName, policy.Name(), tt.wantName)
			}
		})
	}
}

func TestDefaultPolicy(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		workspace  *mockWorkspace
		wantRemove bool
		wantCode   string
	}{
		{
			name: "skip pinned workspace",
			workspace: &mockWorkspace{
				id:   "ws-1",
				name: "pinned-ws",
				flags: &mockFlags{
					pinned: true,
				},
				activity: &mockActivity{},
				status:   &mockStatus{},
			},
			wantRemove: false,
		},
		{
			name: "skip locked workspace",
			workspace: &mockWorkspace{
				id:   "ws-2",
				name: "locked-ws",
				flags: &mockFlags{
					locked: true,
				},
				activity: &mockActivity{},
				status:   &mockStatus{},
			},
			wantRemove: false,
		},
		{
			name: "remove expired ephemeral",
			workspace: &mockWorkspace{
				id:   "ws-3",
				name: "expired-ephemeral",
				flags: &mockFlags{
					ephemeral: true,
				},
				ephemeral: &EphemeralInfo{
					TTLSeconds: 3600,
					ExpiresAt:  now.Add(-1 * time.Hour), // expired 1 hour ago
				},
				activity: &mockActivity{},
				status:   &mockStatus{},
			},
			wantRemove: true,
			wantCode:   "expired_ephemeral",
		},
		{
			name: "keep non-expired ephemeral",
			workspace: &mockWorkspace{
				id:   "ws-4",
				name: "active-ephemeral",
				flags: &mockFlags{
					ephemeral: true,
				},
				ephemeral: &EphemeralInfo{
					TTLSeconds: 3600,
					ExpiresAt:  now.Add(1 * time.Hour), // expires in 1 hour
				},
				activity: &mockActivity{},
				status:   &mockStatus{},
			},
			wantRemove: false,
		},
		{
			name: "remove idle workspace (30+ days)",
			workspace: &mockWorkspace{
				id:   "ws-5",
				name: "idle-ws",
				flags: &mockFlags{},
				activity: &mockActivity{
					lastGitActivityAt: timePtr(now.Add(-35 * 24 * time.Hour)), // 35 days ago
				},
				status: &mockStatus{
					dirty: false,
				},
			},
			wantRemove: true,
			wantCode:   "idle_30d",
		},
		{
			name: "keep idle but dirty workspace",
			workspace: &mockWorkspace{
				id:   "ws-6",
				name: "idle-dirty-ws",
				flags: &mockFlags{},
				activity: &mockActivity{
					lastGitActivityAt: timePtr(now.Add(-35 * 24 * time.Hour)), // 35 days ago
				},
				status: &mockStatus{
					dirty: true,
				},
			},
			wantRemove: false,
		},
		{
			name: "keep recently active workspace",
			workspace: &mockWorkspace{
				id:   "ws-7",
				name: "active-ws",
				flags: &mockFlags{},
				activity: &mockActivity{
					lastGitActivityAt: timePtr(now.Add(-7 * 24 * time.Hour)), // 7 days ago
				},
				status: &mockStatus{},
			},
			wantRemove: false,
		},
	}

	policy := &DefaultPolicy{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, remove := policy.Evaluate(tt.workspace)
			if remove != tt.wantRemove {
				t.Errorf("Evaluate() remove = %v, want %v", remove, tt.wantRemove)
			}
			if tt.wantRemove && reason.Code != tt.wantCode {
				t.Errorf("Evaluate() reason.Code = %q, want %q", reason.Code, tt.wantCode)
			}
		})
	}
}

func TestConservativePolicy(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		workspace  *mockWorkspace
		wantRemove bool
		wantCode   string
	}{
		{
			name: "skip pinned workspace",
			workspace: &mockWorkspace{
				id:   "ws-1",
				name: "pinned-ws",
				flags: &mockFlags{
					pinned: true,
				},
				activity: &mockActivity{},
				status:   &mockStatus{},
			},
			wantRemove: false,
		},
		{
			name: "remove expired ephemeral",
			workspace: &mockWorkspace{
				id:   "ws-2",
				name: "expired-ephemeral",
				flags: &mockFlags{
					ephemeral: true,
				},
				ephemeral: &EphemeralInfo{
					ExpiresAt: now.Add(-1 * time.Hour),
				},
				activity: &mockActivity{},
				status:   &mockStatus{},
			},
			wantRemove: true,
			wantCode:   "expired_ephemeral",
		},
		{
			name: "keep idle workspace (conservative never removes non-ephemeral)",
			workspace: &mockWorkspace{
				id:   "ws-3",
				name: "idle-ws",
				flags: &mockFlags{},
				activity: &mockActivity{
					lastGitActivityAt: timePtr(now.Add(-90 * 24 * time.Hour)), // 90 days ago
				},
				status: &mockStatus{},
			},
			wantRemove: false,
		},
	}

	policy := &ConservativePolicy{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, remove := policy.Evaluate(tt.workspace)
			if remove != tt.wantRemove {
				t.Errorf("Evaluate() remove = %v, want %v", remove, tt.wantRemove)
			}
			if tt.wantRemove && reason.Code != tt.wantCode {
				t.Errorf("Evaluate() reason.Code = %q, want %q", reason.Code, tt.wantCode)
			}
		})
	}
}

func TestAggressivePolicy(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		workspace  *mockWorkspace
		wantRemove bool
		wantCode   string
	}{
		{
			name: "skip pinned workspace",
			workspace: &mockWorkspace{
				id:   "ws-1",
				name: "pinned-ws",
				flags: &mockFlags{
					pinned: true,
				},
				activity: &mockActivity{},
				status:   &mockStatus{},
			},
			wantRemove: false,
		},
		{
			name: "skip locked workspace (even in aggressive)",
			workspace: &mockWorkspace{
				id:   "ws-2",
				name: "locked-ws",
				flags: &mockFlags{
					locked: true,
				},
				activity: &mockActivity{},
				status:   &mockStatus{},
			},
			wantRemove: false,
		},
		{
			name: "remove expired ephemeral",
			workspace: &mockWorkspace{
				id:   "ws-3",
				name: "expired-ephemeral",
				flags: &mockFlags{
					ephemeral: true,
				},
				ephemeral: &EphemeralInfo{
					ExpiresAt: now.Add(-1 * time.Hour),
				},
				activity: &mockActivity{},
				status:   &mockStatus{},
			},
			wantRemove: true,
			wantCode:   "expired_ephemeral",
		},
		{
			name: "remove idle workspace (7+ days in aggressive)",
			workspace: &mockWorkspace{
				id:   "ws-4",
				name: "idle-ws",
				flags: &mockFlags{},
				activity: &mockActivity{
					lastGitActivityAt: timePtr(now.Add(-10 * 24 * time.Hour)), // 10 days ago
				},
				status: &mockStatus{},
			},
			wantRemove: true,
			wantCode:   "idle_7d",
		},
		{
			name: "remove idle dirty workspace (aggressive removes anyway)",
			workspace: &mockWorkspace{
				id:   "ws-5",
				name: "idle-dirty-ws",
				flags: &mockFlags{},
				activity: &mockActivity{
					lastGitActivityAt: timePtr(now.Add(-10 * 24 * time.Hour)),
				},
				status: &mockStatus{
					dirty: true,
				},
			},
			wantRemove: true,
			wantCode:   "idle_7d",
		},
		{
			name: "keep recently active workspace",
			workspace: &mockWorkspace{
				id:   "ws-6",
				name: "active-ws",
				flags: &mockFlags{},
				activity: &mockActivity{
					lastGitActivityAt: timePtr(now.Add(-3 * 24 * time.Hour)), // 3 days ago
				},
				status: &mockStatus{},
			},
			wantRemove: false,
		},
	}

	policy := &AggressivePolicy{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, remove := policy.Evaluate(tt.workspace)
			if remove != tt.wantRemove {
				t.Errorf("Evaluate() remove = %v, want %v", remove, tt.wantRemove)
			}
			if tt.wantRemove && reason.Code != tt.wantCode {
				t.Errorf("Evaluate() reason.Code = %q, want %q", reason.Code, tt.wantCode)
			}
		})
	}
}
