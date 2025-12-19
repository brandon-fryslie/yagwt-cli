package cleanup

import (
	"github.com/bmf/yagwt/internal/core"
)

// CleanupPlan describes what cleanup would do
type CleanupPlan struct {
	Actions  []RemovalAction
	Warnings []Warning
}

// RemovalAction describes a workspace to be removed
type RemovalAction struct {
	Workspace core.Workspace
	Reason    RemovalReason
}

// Warning represents a potentially risky removal
type Warning struct {
	Code    string
	Message string
}

// GeneratePlan evaluates workspaces against a policy and generates a cleanup plan
func GeneratePlan(workspaces []core.Workspace, policy Policy) CleanupPlan {
	plan := CleanupPlan{
		Actions:  []RemovalAction{},
		Warnings: []Warning{},
	}

	// Evaluate each workspace
	for _, ws := range workspaces {
		reason, shouldRemove := policy.Evaluate(ws)
		if shouldRemove {
			action := RemovalAction{
				Workspace: ws,
				Reason:    reason,
			}
			plan.Actions = append(plan.Actions, action)

			// Add warnings for potentially risky removals
			if ws.Status.Dirty {
				plan.Warnings = append(plan.Warnings, Warning{
					Code:    "dirty_workspace",
					Message: "Workspace '" + ws.Name + "' has uncommitted changes",
				})
			}

			// Warn about workspaces with conflicts
			if ws.Status.Conflicts {
				plan.Warnings = append(plan.Warnings, Warning{
					Code:    "conflicts",
					Message: "Workspace '" + ws.Name + "' has merge conflicts",
				})
			}

			// Warn about workspaces that are ahead of upstream
			if ws.Status.Ahead > 0 {
				plan.Warnings = append(plan.Warnings, Warning{
					Code:    "unpushed_commits",
					Message: "Workspace '" + ws.Name + "' has unpushed commits",
				})
			}
		}
	}

	// Sort actions by safety (ephemeral first, then by idle time)
	plan.Actions = sortActions(plan.Actions)

	return plan
}

// sortActions sorts removal actions by safety (safest first)
func sortActions(actions []RemovalAction) []RemovalAction {
	// Categorize actions
	var expired []RemovalAction
	var idle30d []RemovalAction
	var idle7d []RemovalAction
	var other []RemovalAction

	for _, action := range actions {
		switch action.Reason.Code {
		case "expired_ephemeral":
			expired = append(expired, action)
		case "idle_30d":
			idle30d = append(idle30d, action)
		case "idle_7d":
			idle7d = append(idle7d, action)
		default:
			other = append(other, action)
		}
	}

	// Combine in order of safety
	result := make([]RemovalAction, 0, len(actions))
	result = append(result, expired...)
	result = append(result, idle30d...)
	result = append(result, idle7d...)
	result = append(result, other...)

	return result
}
