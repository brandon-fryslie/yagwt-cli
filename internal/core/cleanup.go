package core

import (
	"time"

	"github.com/bmf/yagwt/internal/cleanup"
)

// workspaceAdapter adapts core.Workspace to cleanup.Workspace interface
type workspaceAdapter struct {
	ws Workspace
}

func (w *workspaceAdapter) GetID() string {
	return w.ws.ID
}

func (w *workspaceAdapter) GetName() string {
	return w.ws.Name
}

func (w *workspaceAdapter) GetFlags() cleanup.Flags {
	return &flagsAdapter{flags: w.ws.Flags}
}

func (w *workspaceAdapter) GetEphemeral() *cleanup.EphemeralInfo {
	if w.ws.Ephemeral == nil {
		return nil
	}
	return &cleanup.EphemeralInfo{
		TTLSeconds: w.ws.Ephemeral.TTLSeconds,
		ExpiresAt:  w.ws.Ephemeral.ExpiresAt,
	}
}

func (w *workspaceAdapter) GetActivity() cleanup.Activity {
	return &activityAdapter{activity: w.ws.Activity}
}

func (w *workspaceAdapter) GetStatus() cleanup.Status {
	return &statusAdapter{status: w.ws.Status}
}

// flagsAdapter adapts WorkspaceFlags to cleanup.Flags interface
type flagsAdapter struct {
	flags WorkspaceFlags
}

func (f *flagsAdapter) IsPinned() bool {
	return f.flags.Pinned
}

func (f *flagsAdapter) IsLocked() bool {
	return f.flags.Locked
}

func (f *flagsAdapter) IsEphemeral() bool {
	return f.flags.Ephemeral
}

// activityAdapter adapts ActivityInfo to cleanup.Activity interface
type activityAdapter struct {
	activity ActivityInfo
}

func (a *activityAdapter) GetLastGitActivityAt() *time.Time {
	return a.activity.LastGitActivityAt
}

// statusAdapter adapts StatusInfo to cleanup.Status interface
type statusAdapter struct {
	status StatusInfo
}

func (s *statusAdapter) IsDirty() bool {
	return s.status.Dirty
}

func (s *statusAdapter) HasConflicts() bool {
	return s.status.Conflicts
}

func (s *statusAdapter) GetAhead() int {
	return s.status.Ahead
}

// generateCleanupPlan generates a cleanup plan from workspaces and policy
func generateCleanupPlan(workspaces []Workspace, policy cleanup.Policy) ([]RemovalAction, []Warning) {
	var actions []RemovalAction
	var warnings []Warning

	// Evaluate each workspace
	for _, ws := range workspaces {
		adapter := &workspaceAdapter{ws: ws}
		reason, shouldRemove := policy.Evaluate(adapter)

		if shouldRemove {
			action := RemovalAction{
				Workspace: ws,
				Reason:    reason.Message,
			}
			actions = append(actions, action)

			// Add warnings for potentially risky removals
			if ws.Status.Dirty {
				warnings = append(warnings, Warning{
					Code:    "dirty_workspace",
					Message: "Workspace '" + ws.Name + "' has uncommitted changes",
				})
			}

			// Warn about workspaces with conflicts
			if ws.Status.Conflicts {
				warnings = append(warnings, Warning{
					Code:    "conflicts",
					Message: "Workspace '" + ws.Name + "' has merge conflicts",
				})
			}

			// Warn about workspaces that are ahead of upstream
			if ws.Status.Ahead > 0 {
				warnings = append(warnings, Warning{
					Code:    "unpushed_commits",
					Message: "Workspace '" + ws.Name + "' has unpushed commits",
				})
			}
		}
	}

	// Sort actions by safety
	actions = sortCleanupActions(actions)

	return actions, warnings
}

// sortCleanupActions sorts removal actions by safety (safest first)
func sortCleanupActions(actions []RemovalAction) []RemovalAction {
	// For now, just return as-is since we removed cleanup package dependency
	// TODO: Sort by reason code if needed
	return actions
}
