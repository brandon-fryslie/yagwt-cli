package core

import (
	"time"
)

// Workspace represents a git worktree with metadata
type Workspace struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Path      string         `json:"path"`
	IsPrimary bool           `json:"isPrimary"`
	Target    Target         `json:"target"`
	Flags     WorkspaceFlags `json:"flags"`
	Ephemeral *EphemeralInfo `json:"ephemeral,omitempty"`
	Activity  ActivityInfo   `json:"activity"`
	Status    StatusInfo     `json:"status"`
}

// Target represents the ref a workspace is tracking
type Target struct {
	Type     string `json:"type"` // "branch" or "commit"
	Ref      string `json:"ref"`  // Full ref or SHA
	Short    string `json:"short"`
	Upstream string `json:"upstream,omitempty"`
	HeadSHA  string `json:"headSha"`
}

// WorkspaceFlags contains boolean flags for workspace state
type WorkspaceFlags struct {
	Pinned    bool `json:"pinned"`
	Ephemeral bool `json:"ephemeral"`
	Locked    bool `json:"locked"`
	Broken    bool `json:"broken"`
}

// EphemeralInfo contains TTL information for ephemeral workspaces
type EphemeralInfo struct {
	TTLSeconds int       `json:"ttlSeconds"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

// ActivityInfo tracks workspace usage
type ActivityInfo struct {
	LastOpenedAt      *time.Time `json:"lastOpenedAt,omitempty"`
	LastGitActivityAt *time.Time `json:"lastGitActivityAt,omitempty"`
}

// StatusInfo contains git status information
type StatusInfo struct {
	Dirty     bool   `json:"dirty"`
	Conflicts bool   `json:"conflicts"`
	Ahead     int    `json:"ahead"`
	Behind    int    `json:"behind"`
	Branch    string `json:"branch,omitempty"`
	Detached  bool   `json:"detached"`
}
