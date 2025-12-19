package cleanup

import (
	"time"

	"github.com/bmf/yagwt/internal/core"
)

// Policy defines a cleanup policy
type Policy interface {
	Name() string
	Evaluate(ws core.Workspace) (RemovalReason, bool)
}

// RemovalReason describes why a workspace should be removed
type RemovalReason struct {
	Code    string
	Message string
}

// GetPolicy returns a named policy
func GetPolicy(name string) Policy {
	switch name {
	case "conservative":
		return &ConservativePolicy{}
	case "aggressive":
		return &AggressivePolicy{}
	default:
		return &DefaultPolicy{}
	}
}

// DefaultPolicy is the balanced default cleanup policy
type DefaultPolicy struct{}

func (p *DefaultPolicy) Name() string {
	return "default"
}

func (p *DefaultPolicy) Evaluate(ws core.Workspace) (RemovalReason, bool) {
	// Skip pinned workspaces
	if ws.Flags.Pinned {
		return RemovalReason{}, false
	}

	// Skip locked workspaces
	if ws.Flags.Locked {
		return RemovalReason{}, false
	}

	// Remove expired ephemeral workspaces
	if ws.Flags.Ephemeral && ws.Ephemeral != nil {
		if time.Now().After(ws.Ephemeral.ExpiresAt) {
			return RemovalReason{
				Code:    "expired_ephemeral",
				Message: "Ephemeral workspace has expired",
			}, true
		}
	}

	// Remove workspaces idle for more than 30 days (not dirty)
	if ws.Activity.LastGitActivityAt != nil {
		idleTime := time.Since(*ws.Activity.LastGitActivityAt)
		if idleTime > 30*24*time.Hour && !ws.Status.Dirty {
			return RemovalReason{
				Code:    "idle_30d",
				Message: "Workspace idle for more than 30 days",
			}, true
		}
	}

	return RemovalReason{}, false
}

// ConservativePolicy only removes expired ephemeral workspaces
type ConservativePolicy struct{}

func (p *ConservativePolicy) Name() string {
	return "conservative"
}

func (p *ConservativePolicy) Evaluate(ws core.Workspace) (RemovalReason, bool) {
	// Skip pinned workspaces
	if ws.Flags.Pinned {
		return RemovalReason{}, false
	}

	// Skip locked workspaces
	if ws.Flags.Locked {
		return RemovalReason{}, false
	}

	// Only remove expired ephemeral workspaces
	if ws.Flags.Ephemeral && ws.Ephemeral != nil {
		if time.Now().After(ws.Ephemeral.ExpiresAt) {
			return RemovalReason{
				Code:    "expired_ephemeral",
				Message: "Ephemeral workspace has expired",
			}, true
		}
	}

	return RemovalReason{}, false
}

// AggressivePolicy removes stale workspaces more aggressively
type AggressivePolicy struct{}

func (p *AggressivePolicy) Name() string {
	return "aggressive"
}

func (p *AggressivePolicy) Evaluate(ws core.Workspace) (RemovalReason, bool) {
	// Skip pinned workspaces
	if ws.Flags.Pinned {
		return RemovalReason{}, false
	}

	// Skip locked workspaces (even in aggressive mode)
	if ws.Flags.Locked {
		return RemovalReason{}, false
	}

	// Remove expired ephemeral workspaces
	if ws.Flags.Ephemeral && ws.Ephemeral != nil {
		if time.Now().After(ws.Ephemeral.ExpiresAt) {
			return RemovalReason{
				Code:    "expired_ephemeral",
				Message: "Ephemeral workspace has expired",
			}, true
		}
	}

	// Remove workspaces idle for more than 7 days
	if ws.Activity.LastGitActivityAt != nil {
		idleTime := time.Since(*ws.Activity.LastGitActivityAt)
		if idleTime > 7*24*time.Hour {
			return RemovalReason{
				Code:    "idle_7d",
				Message: "Workspace idle for more than 7 days",
			}, true
		}
	}

	return RemovalReason{}, false
}
