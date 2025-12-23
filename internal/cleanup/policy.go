package cleanup

import (
	"time"
)

// Workspace represents a workspace for cleanup evaluation
// This is a minimal interface to avoid import cycles
type Workspace interface {
	GetID() string
	GetName() string
	GetFlags() Flags
	GetEphemeral() *EphemeralInfo
	GetActivity() Activity
	GetStatus() Status
}

// Flags represents workspace flags
type Flags interface {
	IsPinned() bool
	IsLocked() bool
	IsEphemeral() bool
}

// EphemeralInfo represents TTL information
type EphemeralInfo struct {
	TTLSeconds int
	ExpiresAt  time.Time
}

// Activity represents workspace activity
type Activity interface {
	GetLastGitActivityAt() *time.Time
}

// Status represents workspace status
type Status interface {
	IsDirty() bool
	HasConflicts() bool
	GetAhead() int
}

// Policy defines a cleanup policy
type Policy interface {
	Name() string
	Evaluate(ws Workspace) (RemovalReason, bool)
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

func (p *DefaultPolicy) Evaluate(ws Workspace) (RemovalReason, bool) {
	flags := ws.GetFlags()

	// Skip pinned workspaces
	if flags.IsPinned() {
		return RemovalReason{}, false
	}

	// Skip locked workspaces
	if flags.IsLocked() {
		return RemovalReason{}, false
	}

	// Remove expired ephemeral workspaces
	if flags.IsEphemeral() {
		ephemeral := ws.GetEphemeral()
		if ephemeral != nil && time.Now().After(ephemeral.ExpiresAt) {
			return RemovalReason{
				Code:    "expired_ephemeral",
				Message: "Ephemeral workspace has expired",
			}, true
		}
	}

	// Remove workspaces idle for more than 30 days (not dirty)
	activity := ws.GetActivity()
	lastActivity := activity.GetLastGitActivityAt()
	if lastActivity != nil {
		idleTime := time.Since(*lastActivity)
		status := ws.GetStatus()
		if idleTime > 30*24*time.Hour && !status.IsDirty() {
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

func (p *ConservativePolicy) Evaluate(ws Workspace) (RemovalReason, bool) {
	flags := ws.GetFlags()

	// Skip pinned workspaces
	if flags.IsPinned() {
		return RemovalReason{}, false
	}

	// Skip locked workspaces
	if flags.IsLocked() {
		return RemovalReason{}, false
	}

	// Only remove expired ephemeral workspaces
	if flags.IsEphemeral() {
		ephemeral := ws.GetEphemeral()
		if ephemeral != nil && time.Now().After(ephemeral.ExpiresAt) {
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

func (p *AggressivePolicy) Evaluate(ws Workspace) (RemovalReason, bool) {
	flags := ws.GetFlags()

	// Skip pinned workspaces
	if flags.IsPinned() {
		return RemovalReason{}, false
	}

	// Skip locked workspaces (even in aggressive mode)
	if flags.IsLocked() {
		return RemovalReason{}, false
	}

	// Remove expired ephemeral workspaces
	if flags.IsEphemeral() {
		ephemeral := ws.GetEphemeral()
		if ephemeral != nil && time.Now().After(ephemeral.ExpiresAt) {
			return RemovalReason{
				Code:    "expired_ephemeral",
				Message: "Ephemeral workspace has expired",
			}, true
		}
	}

	// Remove workspaces idle for more than 7 days
	activity := ws.GetActivity()
	lastActivity := activity.GetLastGitActivityAt()
	if lastActivity != nil {
		idleTime := time.Since(*lastActivity)
		if idleTime > 7*24*time.Hour {
			return RemovalReason{
				Code:    "idle_7d",
				Message: "Workspace idle for more than 7 days",
			}, true
		}
	}

	return RemovalReason{}, false
}
