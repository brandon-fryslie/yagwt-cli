package core

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/bmf/yagwt/internal/config"
	"github.com/bmf/yagwt/internal/git"
	"github.com/bmf/yagwt/internal/lock"
	"github.com/bmf/yagwt/internal/metadata"
	"github.com/google/uuid"
)

// WorkspaceManager is the main interface for workspace lifecycle operations
type WorkspaceManager interface {
	// Read operations (lock-free)
	List(opts ListOptions) ([]Workspace, error)
	Get(selector Selector) (Workspace, error)
	Resolve(ref string) ([]Workspace, error)

	// Write operations (acquire lock)
	Create(opts CreateOptions) (Workspace, error)
	Remove(selector Selector, opts RemoveOptions) error
	Rename(selector Selector, newName string) error
	Move(selector Selector, newPath string) error
	Pin(selector Selector) error
	Unpin(selector Selector) error
	Lock(selector Selector) error
	Unlock(selector Selector) error

	// Maintenance operations
	Cleanup(opts CleanupOptions) (CleanupPlan, error)
	Doctor(opts DoctorOptions) (DoctorReport, error)
}

// ListOptions specifies parameters for listing workspaces
type ListOptions struct {
	Filter string
	All    bool
	Fields []string
}

// CreateOptions specifies parameters for creating a workspace
type CreateOptions struct {
	Target    string
	Name      string
	Dir       string
	Base      string
	NewBranch bool
	Existing  bool
	Detached  bool
	Ephemeral bool
	TTL       time.Duration
	Pin       bool
	Checkout  bool
}

// RemoveOptions specifies parameters for removing a workspace
type RemoveOptions struct {
	DeleteBranch bool
	KeepBranch   bool
	OnDirty      string // fail, stash, patch, wip-commit, force
	PatchDir     string
	WipMessage   string
	NoPrompt     bool
}

// CleanupOptions specifies parameters for cleanup operations
type CleanupOptions struct {
	Policy  string
	DryRun  bool
	OnDirty string
	Max     int
}

// DoctorOptions specifies parameters for repair operations
type DoctorOptions struct {
	DryRun        bool
	ForgetMissing bool
}

// CleanupPlan describes what cleanup would do
type CleanupPlan struct {
	Actions  []RemovalAction
	Warnings []Warning
}

// RemovalAction describes a workspace to be removed
type RemovalAction struct {
	Workspace Workspace
	Reason    string
	OnDirty   string
}

// DoctorReport describes repair results
type DoctorReport struct {
	BrokenWorkspaces []Workspace
	Repairs          []Repair
	Warnings         []Warning
}

// Repair describes a repair operation
type Repair struct {
	WorkspaceID string
	Issue       string
	Fix         string
	Applied     bool
}

// Warning represents a non-fatal issue
type Warning struct {
	Code    string
	Message string
}

// engine implements WorkspaceManager interface
type engine struct {
	repo     git.Repository
	store    metadata.Store
	config   *config.Config
	lockMgr  lock.Manager
	lockPath string
}

// NewEngine creates a new WorkspaceManager from a path
func NewEngine(path string) (WorkspaceManager, error) {
	// Find and initialize git repository
	repo, err := git.NewRepository(path)
	if err != nil {
		return nil, err
	}

	// Create metadata store
	store, err := metadata.NewStore(repo.GitDir())
	if err != nil {
		return nil, err
	}

	// Load configuration
	cfg, err := config.Load(repo.Root(), "")
	if err != nil {
		return nil, err
	}

	// Create lock manager
	lockMgr := lock.NewManager()
	lockPath := filepath.Join(repo.GitDir(), "yagwt", "lock")

	return &engine{
		repo:     repo,
		store:    store,
		config:   cfg,
		lockMgr:  lockMgr,
		lockPath: lockPath,
	}, nil
}

// NewEngineWithDeps creates a new WorkspaceManager with injected dependencies (for testing)
func NewEngineWithDeps(repo git.Repository, store metadata.Store, cfg *config.Config, lockMgr lock.Manager) WorkspaceManager {
	lockPath := filepath.Join(repo.GitDir(), "yagwt", "lock")
	return &engine{
		repo:     repo,
		store:    store,
		config:   cfg,
		lockMgr:  lockMgr,
		lockPath: lockPath,
	}
}

// List returns all workspaces with merged git + metadata
func (e *engine) List(opts ListOptions) ([]Workspace, error) {
	// Get git worktrees
	worktrees, err := e.repo.ListWorktrees()
	if err != nil {
		return nil, err
	}

	// Load metadata
	meta, err := e.store.Load()
	if err != nil {
		return nil, err
	}

	// Build map of normalized path -> metadata for efficient lookup
	pathToMeta := make(map[string]metadata.WorkspaceMetadata)
	for _, ws := range meta.Workspaces {
		normalizedPath := normalizePath(ws.Path)
		pathToMeta[normalizedPath] = ws
	}

	// Merge worktrees with metadata
	var workspaces []Workspace
	seenPaths := make(map[string]bool)

	for i, wt := range worktrees {
		// Normalize worktree path for consistent comparison
		normalizedWtPath := normalizePath(wt.Path)
		seenPaths[normalizedWtPath] = true

		// Get metadata if available
		wsMeta, hasMeta := pathToMeta[normalizedWtPath]

		// Get git status
		status, err := e.repo.GetStatus(wt.Path)
		if err != nil {
			// If we can't get status, continue but mark as broken
			status = git.Status{}
		}

		// Determine if this is the primary workspace
		isPrimary := i == 0 // First worktree is typically primary

		// Build Target info
		target := Target{
			HeadSHA: wt.HEAD,
		}
		if wt.Branch != "" {
			target.Type = "branch"
			target.Ref = "refs/heads/" + wt.Branch
			target.Short = wt.Branch
			target.Upstream = "" // TODO: Get from git.GetBranch
		} else {
			target.Type = "commit"
			target.Ref = wt.HEAD
			target.Short = wt.HEAD[:7]
		}

		// Build Workspace
		ws := Workspace{
			ID:        "",
			Name:      "",
			Path:      wt.Path,
			IsPrimary: isPrimary,
			Target:    target,
			Flags: WorkspaceFlags{
				Pinned:    false,
				Ephemeral: false,
				Locked:    wt.Locked,
				Broken:    false,
			},
			Activity: ActivityInfo{},
			Status: StatusInfo{
				Dirty:     status.Dirty,
				Conflicts: status.Conflicts,
				Ahead:     status.Ahead,
				Behind:    status.Behind,
				Branch:    status.Branch,
				Detached:  status.Detached,
			},
		}

		// Merge metadata if available
		if hasMeta {
			ws.ID = wsMeta.ID
			ws.Name = wsMeta.Name

			// Merge flags
			if pinned, ok := wsMeta.Flags["pinned"]; ok {
				ws.Flags.Pinned = pinned
			}
			if ephemeral, ok := wsMeta.Flags["ephemeral"]; ok {
				ws.Flags.Ephemeral = ephemeral
			}
			if locked, ok := wsMeta.Flags["locked"]; ok {
				ws.Flags.Locked = locked
			}

			// Copy ephemeral info
			if wsMeta.Ephemeral != nil {
				ws.Ephemeral = &EphemeralInfo{
					TTLSeconds: wsMeta.Ephemeral.TTLSeconds,
					ExpiresAt:  wsMeta.Ephemeral.ExpiresAt,
				}
			}

			// Copy activity info
			ws.Activity = ActivityInfo{
				LastOpenedAt:      wsMeta.Activity.LastOpenedAt,
				LastGitActivityAt: wsMeta.Activity.LastGitActivityAt,
			}
		} else {
			// Generate a temporary ID for display (not persisted)
			ws.ID = "<no-metadata>"
			ws.Name = filepath.Base(wt.Path)
		}

		workspaces = append(workspaces, ws)
	}

	// Check for orphaned metadata (metadata without git worktree)
	for _, wsMeta := range meta.Workspaces {
		normalizedMetaPath := normalizePath(wsMeta.Path)
		if !seenPaths[normalizedMetaPath] {
			// Workspace metadata exists but git worktree is missing
			ws := Workspace{
				ID:   wsMeta.ID,
				Name: wsMeta.Name,
				Path: wsMeta.Path,
				Flags: WorkspaceFlags{
					Broken: true,
				},
			}

			// Copy other metadata
			if pinned, ok := wsMeta.Flags["pinned"]; ok {
				ws.Flags.Pinned = pinned
			}
			if ephemeral, ok := wsMeta.Flags["ephemeral"]; ok {
				ws.Flags.Ephemeral = ephemeral
			}

			if wsMeta.Ephemeral != nil {
				ws.Ephemeral = &EphemeralInfo{
					TTLSeconds: wsMeta.Ephemeral.TTLSeconds,
					ExpiresAt:  wsMeta.Ephemeral.ExpiresAt,
				}
			}

			ws.Activity = ActivityInfo{
				LastOpenedAt:      wsMeta.Activity.LastOpenedAt,
				LastGitActivityAt: wsMeta.Activity.LastGitActivityAt,
			}

			workspaces = append(workspaces, ws)
		}
	}

	return workspaces, nil
}

// Get returns a single workspace by selector
func (e *engine) Get(selector Selector) (Workspace, error) {
	workspaces, err := e.Resolve(selectorToString(selector))
	if err != nil {
		return Workspace{}, err
	}

	if len(workspaces) == 0 {
		return Workspace{}, NewError(ErrNotFound, "workspace not found").
			WithDetail("selector", selectorToString(selector))
	}

	if len(workspaces) > 1 {
		return Workspace{}, NewError(ErrAmbiguous, "selector matches multiple workspaces").
			WithDetail("selector", selectorToString(selector)).
			WithDetail("count", len(workspaces)).
			WithHint("Use a more specific selector (id:, name:, or path:)", "")
	}

	return workspaces[0], nil
}

// Resolve resolves a selector to matching workspaces
func (e *engine) Resolve(ref string) ([]Workspace, error) {
	selector := ParseSelector(ref)

	// Get all workspaces
	allWorkspaces, err := e.List(ListOptions{})
	if err != nil {
		return nil, err
	}

	var matches []Workspace

	switch selector.Type {
	case SelectorID:
		// Direct ID match
		for _, ws := range allWorkspaces {
			if ws.ID == selector.Value {
				matches = append(matches, ws)
			}
		}

	case SelectorName:
		// Direct name match
		for _, ws := range allWorkspaces {
			if ws.Name == selector.Value {
				matches = append(matches, ws)
			}
		}

	case SelectorPath:
		// Direct path match (normalize both for comparison)
		normalizedSelectorPath := normalizePath(selector.Value)
		for _, ws := range allWorkspaces {
			normalizedWsPath := normalizePath(ws.Path)
			if normalizedWsPath == normalizedSelectorPath {
				matches = append(matches, ws)
			}
		}

	case SelectorBranch:
		// Match by branch
		for _, ws := range allWorkspaces {
			if ws.Target.Short == selector.Value || ws.Target.Ref == "refs/heads/"+selector.Value {
				matches = append(matches, ws)
			}
		}

	case SelectorBare:
		// Try in order: id → name → path → branch
		// First try ID
		for _, ws := range allWorkspaces {
			if ws.ID == selector.Value {
				matches = append(matches, ws)
			}
		}
		if len(matches) > 0 {
			return matches, nil
		}

		// Try name
		for _, ws := range allWorkspaces {
			if ws.Name == selector.Value {
				matches = append(matches, ws)
			}
		}
		if len(matches) > 0 {
			return matches, nil
		}

		// Try path
		normalizedSelectorValue := normalizePath(selector.Value)
		for _, ws := range allWorkspaces {
			normalizedWsPath := normalizePath(ws.Path)
			if normalizedWsPath == normalizedSelectorValue {
				matches = append(matches, ws)
			}
		}
		if len(matches) > 0 {
			return matches, nil
		}

		// Try branch
		for _, ws := range allWorkspaces {
			if ws.Target.Short == selector.Value || ws.Target.Ref == "refs/heads/"+selector.Value {
				matches = append(matches, ws)
			}
		}
	}

	return matches, nil
}

// Create creates a new workspace
func (e *engine) Create(opts CreateOptions) (Workspace, error) {
	// Acquire lock for write operation
	lck, err := e.lockMgr.NewLock(e.lockPath)
	if err != nil {
		return Workspace{}, err
	}

	if err := lck.Acquire(3 * time.Second); err != nil {
		return Workspace{}, err
	}
	defer lck.Release()

	// Determine workspace path
	wsPath := opts.Dir
	if wsPath == "" {
		// Derive path from config
		wsPath, err = e.deriveWorkspacePath(opts)
		if err != nil {
			return Workspace{}, err
		}
	}

	// Make path absolute
	if !filepath.IsAbs(wsPath) {
		wsPath, err = filepath.Abs(wsPath)
		if err != nil {
			return Workspace{}, WrapError(ErrConfig, "failed to make path absolute", err).
				WithDetail("path", wsPath)
		}
	}

	// Build git add options
	gitOpts := git.AddOptions{
		NewBranch: opts.NewBranch,
		Detach:    opts.Detached,
		Checkout:  opts.Checkout,
		Force:     false,
	}

	// Create git worktree
	if err := e.repo.AddWorktree(wsPath, opts.Target, gitOpts); err != nil {
		return Workspace{}, err
	}

	// Generate workspace ID
	wsID := uuid.New().String()

	// Determine workspace name
	wsName := opts.Name
	if wsName == "" {
		// Derive from target or path
		if opts.Target != "" && !opts.Detached {
			// Use branch name
			wsName = strings.TrimPrefix(opts.Target, "refs/heads/")
			wsName = strings.ReplaceAll(wsName, "/", "-")
		} else {
			// Use directory name
			wsName = filepath.Base(wsPath)
		}
	}

	// Create metadata
	now := time.Now()
	wsMeta := metadata.WorkspaceMetadata{
		ID:   wsID,
		Name: wsName,
		Path: wsPath,
		Flags: map[string]bool{
			"pinned":    opts.Pin,
			"ephemeral": opts.Ephemeral,
			"locked":    false,
		},
		Activity: metadata.ActivityMetadata{
			LastOpenedAt:      nil,
			LastGitActivityAt: &now,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Add ephemeral info if needed
	if opts.Ephemeral && opts.TTL > 0 {
		wsMeta.Ephemeral = &metadata.EphemeralMetadata{
			TTLSeconds: int(opts.TTL.Seconds()),
			ExpiresAt:  now.Add(opts.TTL),
		}
	}

	// Save metadata
	if err := e.store.Set(wsID, wsMeta); err != nil {
		// Try to clean up git worktree on metadata failure
		_ = e.repo.RemoveWorktree(wsPath, true)
		return Workspace{}, err
	}

	// Get fresh workspace state
	ws, err := e.Get(Selector{Type: SelectorID, Value: wsID})
	if err != nil {
		return Workspace{}, err
	}

	return ws, nil
}

// deriveWorkspacePath derives a workspace path from config and options
func (e *engine) deriveWorkspacePath(opts CreateOptions) (string, error) {
	repoRoot := e.repo.Root()

	// Determine directory name
	dirName := opts.Name
	if dirName == "" {
		if opts.Target != "" && !opts.Detached {
			// Use branch name
			dirName = strings.TrimPrefix(opts.Target, "refs/heads/")
			dirName = strings.ReplaceAll(dirName, "/", "-")
		} else {
			return "", NewError(ErrConfig, "cannot derive workspace path: name or target required")
		}
	}

	var wsPath string
	switch e.config.Workspace.RootStrategy {
	case "sibling":
		// Place workspace as sibling to repo root
		parentDir := filepath.Dir(repoRoot)
		wsPath = filepath.Join(parentDir, dirName)

	case "inside":
		// Place workspace inside repo
		rootDir := e.config.Workspace.RootDir
		if rootDir == "" {
			rootDir = ".workspaces"
		}
		wsPath = filepath.Join(repoRoot, rootDir, dirName)

	default:
		return "", NewError(ErrConfig, "invalid rootStrategy in config").
			WithDetail("strategy", e.config.Workspace.RootStrategy)
	}

	return wsPath, nil
}

// Remove removes a workspace
func (e *engine) Remove(selector Selector, opts RemoveOptions) error {
	// Acquire lock for write operation
	lck, err := e.lockMgr.NewLock(e.lockPath)
	if err != nil {
		return err
	}

	if err := lck.Acquire(3 * time.Second); err != nil {
		return err
	}
	defer lck.Release()

	// Resolve workspace
	ws, err := e.Get(selector)
	if err != nil {
		return err
	}

	// Check if pinned
	if ws.Flags.Pinned {
		return NewError(ErrLocked, "workspace is pinned").
			WithDetail("id", ws.ID).
			WithDetail("name", ws.Name).
			WithHint("Unpin the workspace first", "yagwt unpin "+ws.Name)
	}

	// Check if locked
	if ws.Flags.Locked {
		return NewError(ErrLocked, "workspace is locked").
			WithDetail("id", ws.ID).
			WithDetail("name", ws.Name).
			WithHint("Unlock the workspace first", "yagwt unlock "+ws.Name)
	}

	// Check if dirty
	onDirty := opts.OnDirty
	if onDirty == "" {
		onDirty = "fail"
	}

	if ws.Status.Dirty && onDirty == "fail" {
		return NewError(ErrDirty, "workspace has uncommitted changes").
			WithDetail("id", ws.ID).
			WithDetail("name", ws.Name).
			WithHint("Commit or stash changes, or use --on-dirty=force", "git -C "+ws.Path+" status")
	}

	// Remove git worktree
	force := onDirty == "force"
	if err := e.repo.RemoveWorktree(ws.Path, force); err != nil {
		return err
	}

	// Remove metadata
	if err := e.store.Delete(ws.ID); err != nil {
		return err
	}

	return nil
}

// Rename renames a workspace
func (e *engine) Rename(selector Selector, newName string) error {
	// Acquire lock
	lck, err := e.lockMgr.NewLock(e.lockPath)
	if err != nil {
		return err
	}

	if err := lck.Acquire(3 * time.Second); err != nil {
		return err
	}
	defer lck.Release()

	// Resolve workspace
	ws, err := e.Get(selector)
	if err != nil {
		return err
	}

	// Check for name conflict
	existing, _ := e.Resolve("name:" + newName)
	if len(existing) > 0 && existing[0].ID != ws.ID {
		return NewError(ErrConflict, "workspace with this name already exists").
			WithDetail("name", newName)
	}

	// Load metadata
	meta, err := e.store.Get(ws.ID)
	if err != nil {
		return err
	}

	// Update name
	meta.Name = newName
	meta.UpdatedAt = time.Now()

	// Save
	return e.store.Set(ws.ID, meta)
}

// Move moves a workspace (stub for Phase 3)
func (e *engine) Move(selector Selector, newPath string) error {
	return NewError(ErrConfig, "move operation not yet implemented")
}

// Pin pins a workspace
func (e *engine) Pin(selector Selector) error {
	return e.setFlag(selector, "pinned", true)
}

// Unpin unpins a workspace
func (e *engine) Unpin(selector Selector) error {
	return e.setFlag(selector, "pinned", false)
}

// Lock locks a workspace
func (e *engine) Lock(selector Selector) error {
	return e.setFlag(selector, "locked", true)
}

// Unlock unlocks a workspace
func (e *engine) Unlock(selector Selector) error {
	return e.setFlag(selector, "locked", false)
}

// setFlag sets a flag on a workspace
func (e *engine) setFlag(selector Selector, flag string, value bool) error {
	// Acquire lock
	lck, err := e.lockMgr.NewLock(e.lockPath)
	if err != nil {
		return err
	}

	if err := lck.Acquire(3 * time.Second); err != nil {
		return err
	}
	defer lck.Release()

	// Resolve workspace
	ws, err := e.Get(selector)
	if err != nil {
		return err
	}

	// Load metadata
	meta, err := e.store.Get(ws.ID)
	if err != nil {
		return err
	}

	// Update flag
	if meta.Flags == nil {
		meta.Flags = make(map[string]bool)
	}
	meta.Flags[flag] = value
	meta.UpdatedAt = time.Now()

	// Save
	return e.store.Set(ws.ID, meta)
}

// Cleanup is a stub for Phase 3
func (e *engine) Cleanup(opts CleanupOptions) (CleanupPlan, error) {
	return CleanupPlan{}, NewError(ErrConfig, "cleanup operation not yet implemented")
}

// Doctor is a stub for Phase 3
func (e *engine) Doctor(opts DoctorOptions) (DoctorReport, error) {
	return DoctorReport{}, NewError(ErrConfig, "doctor operation not yet implemented")
}

// selectorToString converts a Selector back to a string for error messages
func selectorToString(s Selector) string {
	switch s.Type {
	case SelectorID:
		return "id:" + s.Value
	case SelectorName:
		return "name:" + s.Value
	case SelectorPath:
		return "path:" + s.Value
	case SelectorBranch:
		return "branch:" + s.Value
	default:
		return s.Value
	}
}
