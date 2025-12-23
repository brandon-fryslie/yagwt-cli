# YAGWT Architecture

**Last Updated**: 2025-12-18
**Version**: 1.0

---

## System Overview

YAGWT is a three-layer system that separates concerns for maximum testability and extensibility:

```
┌──────────────────────────────────────────────────────────────────────┐
│                         USER INTERFACES                              │
│                                                                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                │
│  │     CLI     │  │  TUI        │  │  IDE Plugin │  (future)      │
│  │  (primary)  │  │  (future)   │  │  (future)   │                │
│  └─────────────┘  └─────────────┘  └─────────────┘                │
└────────────────────────────┬─────────────────────────────────────────┘
                             │
                             │ All interfaces invoke
                             │ the same core engine
                             ▼
┌──────────────────────────────────────────────────────────────────────┐
│                          CORE ENGINE                                 │
│                    (Pure business logic)                             │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                    WorkspaceManager                          │  │
│  │  • List/filter workspaces                                    │  │
│  │  • Create/remove workspaces                                  │  │
│  │  • Modify workspace metadata (rename, pin, lock)             │  │
│  │  • Resolve selectors → workspace references                  │  │
│  │  • Evaluate cleanup policies                                 │  │
│  │  • Detect and repair broken workspaces                       │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  All operations return structured Result types (no printing)        │
└────────────────────────────┬─────────────────────────────────────────┘
                             │
                             │ Uses
                             ▼
┌──────────────────────────────────────────────────────────────────────┐
│                      INFRASTRUCTURE LAYER                            │
│                                                                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                │
│  │     Git     │  │  Metadata   │  │   Config    │                │
│  │   Wrapper   │  │   Storage   │  │   Loader    │                │
│  └─────────────┘  └─────────────┘  └─────────────┘                │
│                                                                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                │
│  │    Lock     │  │   Filter    │  │    Hooks    │                │
│  │   Manager   │  │   Engine    │  │  Executor   │                │
│  └─────────────┘  └─────────────┘  └─────────────┘                │
└────────────────────────────┬─────────────────────────────────────────┘
                             │
                             │ Executes
                             ▼
                      ┌──────────────┐
                      │ git commands │
                      │ (subprocess) │
                      └──────────────┘
```

---

## Component Details

### 1. CLI Layer (cmd/yagwt, internal/cli)

**Package structure**:
```
cmd/yagwt/
  main.go                    # Entry point, cobra root command setup
internal/cli/
  commands/
    ls.go                    # List workspaces command
    new.go                   # Create workspace command
    rm.go                    # Remove workspace command
    clean.go                 # Cleanup command
    doctor.go                # Repair command
    show.go                  # Show workspace details
    ... (all other commands)
  output/
    formatter.go             # Formatter interface
    json.go                  # JSON formatter
    porcelain.go             # Porcelain formatter
    table.go                 # Human-readable table formatter
  prompt/
    interactive.go           # User prompts and confirmations
  flags.go                   # Common flag definitions
  errors.go                  # Error display with hints
```

**Responsibilities**:
- Parse command-line arguments using cobra
- Validate flags and arguments
- Invoke core engine operations
- Format results for output (JSON, porcelain, table)
- Handle interactive prompts (unless --no-prompt)
- Display errors with actionable hints
- Set exit codes based on operation results

**Key interfaces**:
```go
// Formatter converts engine results to output
type Formatter interface {
    FormatWorkspaces(workspaces []core.Workspace) string
    FormatError(err error) string
}

// Prompter handles user interaction
type Prompter interface {
    Confirm(message string) bool
    Select(message string, options []string) (int, error)
}
```

---

### 2. Core Engine (internal/core)

**Package structure**:
```
internal/core/
  engine.go                  # WorkspaceManager implementation
  workspace.go               # Workspace type and methods
  result.go                  # Result types for operations
  selector.go                # Selector resolution
  cleanup.go                 # Cleanup policy evaluation
  doctor.go                  # Repair operations
  create.go                  # Workspace creation logic
  remove.go                  # Workspace removal logic
  modify.go                  # Rename, move, pin, lock operations
  errors.go                  # Structured error types
```

**Key types**:
```go
// Workspace represents a git worktree with metadata
type Workspace struct {
    ID          string           // wsp_<uuid>
    Name        string           // User-assigned alias
    Path        string           // Absolute path
    IsPrimary   bool             // Is this the original checkout?
    Target      Target           // Branch or commit
    Flags       WorkspaceFlags   // pinned, ephemeral, locked, broken
    Ephemeral   *EphemeralInfo   // TTL and expiry (if ephemeral)
    Activity    ActivityInfo     // Last opened, last git activity
    Status      StatusInfo       // Dirty, conflicts, ahead/behind
}

// WorkspaceManager is the main interface for workspace operations
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

// Selector identifies a workspace
type Selector struct {
    Type  SelectorType  // ID, Name, Path, Branch
    Value string
}

// Result wraps operation outcomes
type Result struct {
    Workspace *Workspace
    Warnings  []Warning
    Errors    []Error
}
```

**Responsibilities**:
- Implement all workspace lifecycle operations
- Coordinate between git wrapper, metadata storage, and config
- Resolve selectors to workspace references
- Evaluate cleanup policies
- Detect and repair broken workspaces
- Return structured results (never print, never prompt)
- Handle git errors gracefully

**Critical design principle**: The core engine has ZERO side effects beyond filesystem changes (git operations, metadata writes). No printing, no prompting, no reading environment variables directly (config layer does this).

---

### 3. Git Wrapper (internal/git)

**Package structure**:
```
internal/git/
  repo.go                    # Repository interface
  worktree.go                # Worktree operations
  status.go                  # Status parsing
  refs.go                    # Reference resolution
  subprocess.go              # Git subprocess execution
  errors.go                  # Git error types
```

**Key interfaces**:
```go
// Repository provides git operations
type Repository interface {
    // Worktree operations
    ListWorktrees() ([]Worktree, error)
    AddWorktree(path, ref string, opts AddOptions) error
    RemoveWorktree(path string, force bool) error

    // Status operations
    GetStatus(path string) (Status, error)

    // Reference operations
    ResolveRef(ref string) (string, error)  // Returns full SHA
    GetBranch(ref string) (Branch, error)

    // Repository info
    Root() string
    GitDir() string
}

// Worktree represents git worktree list output
type Worktree struct {
    Path   string
    HEAD   string  // SHA
    Branch string  // Empty if detached
}

// Status represents git status output
type Status struct {
    Dirty      bool
    Conflicts  bool
    Branch     string
    Detached   bool
    Ahead      int
    Behind     int
}
```

**Responsibilities**:
- Execute git commands as subprocesses
- Parse git output (worktree list, status, refs)
- Handle git errors (non-zero exit codes, stderr)
- Detect repository root and git directory
- Validate git version compatibility

**Implementation notes**:
- Use `exec.Command` for subprocess execution
- Capture stdout and stderr separately
- Parse output using regex or string parsing
- Convert git errors to structured error types

---

### 4. Metadata Storage (internal/metadata)

**Package structure**:
```
internal/metadata/
  store.go                   # MetadataStore interface
  json.go                    # JSON file storage implementation
  index.go                   # Reverse index (path → ID)
  migrate.go                 # Schema migration (future)
```

**Key interfaces**:
```go
// MetadataStore persists workspace metadata
type MetadataStore interface {
    // Read operations (lock-free)
    Load() (Metadata, error)
    Get(id string) (WorkspaceMetadata, error)
    FindByName(name string) (WorkspaceMetadata, error)
    FindByPath(path string) (WorkspaceMetadata, error)

    // Write operations (requires lock)
    Save(metadata Metadata) error
    Set(id string, meta WorkspaceMetadata) error
    Delete(id string) error

    // Index operations
    RebuildIndex() error
}

// Metadata is the top-level structure persisted to disk
type Metadata struct {
    SchemaVersion int
    Workspaces    map[string]WorkspaceMetadata  // ID → metadata
    Index         Index                          // Reverse lookups
}

// WorkspaceMetadata is per-workspace persistent data
type WorkspaceMetadata struct {
    ID          string
    Name        string
    Path        string
    Flags       WorkspaceFlags
    Ephemeral   *EphemeralInfo
    Activity    ActivityInfo
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// Index provides reverse lookups
type Index struct {
    ByPath   map[string]string  // path → ID
    ByName   map[string]string  // name → ID
    ByBranch map[string][]string // branch → []ID
}
```

**Responsibilities**:
- Read metadata from `<gitDir>/yagwt/meta.json`
- Write metadata atomically (temp file + rename)
- Maintain reverse index for fast lookups
- Handle missing/corrupted metadata files gracefully
- Coordinate with lock manager for atomic writes

**File format** (JSON):
```json
{
  "schemaVersion": 1,
  "workspaces": {
    "wsp_01HZX...": {
      "id": "wsp_01HZX...",
      "name": "feature-auth",
      "path": "/abs/path/to/workspace",
      "flags": {
        "pinned": false,
        "ephemeral": true,
        "locked": false,
        "broken": false
      },
      "ephemeral": {
        "ttlSeconds": 604800,
        "expiresAt": "2025-12-25T10:00:00Z"
      },
      "activity": {
        "lastOpenedAt": "2025-12-18T10:00:00Z",
        "lastGitActivityAt": "2025-12-18T11:30:00Z"
      },
      "createdAt": "2025-12-18T09:00:00Z",
      "updatedAt": "2025-12-18T11:30:00Z"
    }
  },
  "index": {
    "byPath": { "/abs/path/to/workspace": "wsp_01HZX..." },
    "byName": { "feature-auth": "wsp_01HZX..." },
    "byBranch": { "refs/heads/feature/auth": ["wsp_01HZX..."] }
  }
}
```

---

### 5. Config Loader (internal/config)

**Package structure**:
```
internal/config/
  config.go                  # Config type and loader
  defaults.go                # Default values
  validate.go                # Config validation
  policies.go                # Cleanup policy definitions
```

**Key types**:
```go
// Config is the full configuration
type Config struct {
    Workspace WorkspaceConfig
    Cleanup   CleanupConfig
    Hooks     HooksConfig
}

// WorkspaceConfig controls workspace creation
type WorkspaceConfig struct {
    RootStrategy string  // "sibling" or "inside"
    RootDir      string  // ".workspaces" (if inside)
    NameTemplate string  // "{branch}" or custom
}

// CleanupConfig defines cleanup policies
type CleanupConfig struct {
    Policies map[string]CleanupPolicy
}

// CleanupPolicy defines rules for cleanup
type CleanupPolicy struct {
    RemoveEphemeral bool
    IdleThreshold   time.Duration
    RespectPinned   bool
    OnDirty         string  // fail, stash, patch, wip-commit
}

// HooksConfig defines hook scripts
type HooksConfig struct {
    PostCreate  string
    PreRemove   string
    PostRemove  string
    PostOpen    string
}
```

**Responsibilities**:
- Discover config files (CLI flag → repo → user → system)
- Parse TOML config files
- Merge configs with precedence rules
- Provide default values
- Validate config values

**Config file locations** (precedence):
1. `--config <path>` (CLI flag)
2. `<repoRoot>/.yagwt/config.toml` (repo-specific)
3. macOS: `~/Library/Application Support/yagwt/config.toml` OR `~/.config/yagwt/config.toml`
4. Linux: `$XDG_CONFIG_HOME/yagwt/config.toml` OR `~/.config/yagwt/config.toml`

**Example config.toml**:
```toml
[workspace]
rootStrategy = "sibling"  # or "inside"
rootDir = ".workspaces"   # only used if rootStrategy = "inside"
nameTemplate = "{branch}"

[cleanup.policies.default]
removeEphemeral = true
idleThreshold = "30d"
respectPinned = true
onDirty = "fail"

[cleanup.policies.aggressive]
removeEphemeral = true
idleThreshold = "7d"
respectPinned = false
onDirty = "stash"

[hooks]
postCreate = ".yagwt/hooks/post-create"
preRemove = ".yagwt/hooks/pre-remove"
```

---

### 6. Lock Manager (internal/lock)

**Package structure**:
```
internal/lock/
  lock.go                    # Lock interface
  flock.go                   # File locking implementation (Unix)
```

**Key interfaces**:
```go
// Lock provides concurrency control
type Lock interface {
    Acquire(timeout time.Duration) error
    Release() error
}

// FileLock is a file-based advisory lock
type FileLock struct {
    path string
    fd   int
}
```

**Responsibilities**:
- Acquire advisory file locks (flock on Linux, fcntl on macOS)
- Release locks
- Handle timeouts gracefully
- Support both blocking and non-blocking acquisition

**Lock file location**: `<gitDir>/yagwt/lock`

**Locking strategy**:
- **Read operations** (list, get, resolve): No lock required (eventually consistent reads)
- **Write operations** (create, remove, modify): Acquire exclusive lock → read current state → modify → write → release lock

**Implementation notes**:
- Use `golang.org/x/sys/unix` for flock/fcntl
- Default timeout: 3 seconds
- Lock file is created if missing
- Lock is automatically released on process exit (OS guarantee)

---

### 7. Filter Engine (internal/filter)

**Package structure**:
```
internal/filter/
  parser.go                  # Parse filter expressions
  eval.go                    # Evaluate filters against workspaces
  builtins.go                # Built-in filter functions
```

**Key types**:
```go
// Filter represents a parsed filter expression
type Filter interface {
    Match(workspace core.Workspace) bool
}

// Examples of filter types
type FlagFilter struct {
    Flag  string  // "pinned", "ephemeral", "locked", "broken"
    Value bool
}

type StatusFilter struct {
    Key   string  // "dirty", "conflicts"
    Value bool
}

type ActivityFilter struct {
    Key      string        // "idle"
    Operator string        // ">", "<", "="
    Duration time.Duration
}

type AndFilter struct {
    Filters []Filter
}

type OrFilter struct {
    Filters []Filter
}
```

**Filter syntax**:
```
flag:pinned              # Pinned workspaces
flag:ephemeral           # Ephemeral workspaces
status:dirty             # Dirty workspaces
status:broken            # Broken workspaces
target:branch=feature/x  # Workspaces targeting specific branch
activity:idle>30d        # Idle for >30 days
name:auth*               # Name glob matching

# Combining filters (AND by default)
flag:ephemeral status:clean activity:idle>7d

# OR with comma
flag:pinned,flag:locked  # Pinned OR locked
```

**Responsibilities**:
- Parse filter expressions into AST
- Evaluate filters against workspace objects
- Support built-in filters (flag, status, target, activity, name)
- Support AND/OR logic

---

### 8. Hook Executor (internal/hooks)

**Package structure**:
```
internal/hooks/
  executor.go                # Hook execution
  discover.go                # Hook script discovery
```

**Key interfaces**:
```go
// HookExecutor executes lifecycle hooks
type HookExecutor interface {
    Execute(hook HookType, context HookContext) error
}

// HookType identifies the hook
type HookType string

const (
    PostCreate HookType = "post-create"
    PreRemove  HookType = "pre-remove"
    PostRemove HookType = "post-remove"
    PostOpen   HookType = "post-open"
)

// HookContext provides data to hook scripts
type HookContext struct {
    RepoRoot      string
    WorkspaceID   string
    WorkspacePath string
    WorkspaceName string
    TargetRef     string
    Operation     string
}
```

**Responsibilities**:
- Discover hook scripts (config + `<repoRoot>/.yagwt/hooks/`)
- Execute hooks with environment variables
- Handle hook failures (abort operation? warn only?)
- Support multiple hooks for same lifecycle event

**Hook discovery order** (first match wins):
1. Config file setting (e.g., `hooks.postCreate = "/path/to/script"`)
2. `<repoRoot>/.yagwt/hooks/<hook-name>` (executable file)

**Environment variables passed to hooks**:
- `YAGWT_REPO_ROOT`: Repository root path
- `YAGWT_WORKSPACE_ID`: Workspace ID (wsp_*)
- `YAGWT_WORKSPACE_PATH`: Absolute workspace path
- `YAGWT_WORKSPACE_NAME`: Workspace name (alias)
- `YAGWT_TARGET_REF`: Target reference (branch or commit)
- `YAGWT_OPERATION`: Operation name (create, remove, open)

**Hook failure handling**:
- **pre-remove**: Failure aborts operation
- **post-create, post-remove, post-open**: Failure is logged as warning, operation continues

---

## Data Flow Examples

### Example 1: List Workspaces with Filter

```
User: yagwt ls --filter="flag:ephemeral status:clean"

1. CLI parses flags and filter expression
2. CLI invokes: engine.List(ListOptions{Filter: "flag:ephemeral status:clean"})
3. Core engine:
   a. Loads metadata (lock-free read)
   b. Lists git worktrees via git wrapper
   c. Merges git state + metadata → []Workspace
   d. Parses filter expression via filter engine
   e. Filters workspaces: w.Flags.Ephemeral && !w.Status.Dirty
   f. Returns: Result{Workspaces: [...]}
4. CLI formats workspaces as table
5. CLI prints table to stdout
```

### Example 2: Create Ephemeral Workspace

```
User: yagwt new feature/auth --name auth --ephemeral --ttl 7d

1. CLI parses flags
2. CLI invokes: engine.Create(CreateOptions{
     Target: "feature/auth",
     Name: "auth",
     Ephemeral: true,
     TTL: 7*24*time.Hour,
   })
3. Core engine:
   a. Acquires lock
   b. Loads metadata
   c. Resolves target ref "feature/auth" → refs/heads/feature/auth via git wrapper
   d. Determines workspace path:
      - If config.workspace.rootStrategy = "sibling":
        path = <repoParent>/feature-auth
      - If "inside":
        path = <repoRoot>/.workspaces/feature-auth
   e. Generates workspace ID: wsp_01HZX...
   f. Invokes: git worktree add <path> refs/heads/feature/auth
   g. Creates metadata:
      {
        id: wsp_01HZX...,
        name: "auth",
        path: <path>,
        flags: {ephemeral: true},
        ephemeral: {ttlSeconds: 604800, expiresAt: now+7d},
        createdAt: now,
      }
   h. Saves metadata atomically
   i. Executes post-create hook (if configured)
   j. Releases lock
   k. Returns: Result{Workspace: {...}}
4. CLI formats result
5. CLI prints: "Created workspace 'auth' at <path> (expires in 7 days)"
```

### Example 3: Cleanup with Policy

```
User: yagwt clean --policy=default --plan

1. CLI parses flags
2. CLI invokes: engine.Cleanup(CleanupOptions{
     Policy: "default",
     DryRun: true,
   })
3. Core engine:
   a. Loads metadata (lock-free)
   b. Lists git worktrees
   c. Loads cleanup policy "default" from config
   d. Evaluates policy against each workspace:
      - Expired ephemeral? → Remove (if clean or policy.onDirty allows)
      - Idle >30d + not pinned + clean? → Remove
      - Broken? → Warn (doctor should handle)
   e. Generates removal plan: []RemovalAction
   f. Returns: Result{Plan: [...]}
4. CLI formats plan as table:
   | Workspace | Reason    | On Dirty | Will Delete Branch? |
   |-----------|-----------|----------|---------------------|
   | old-feat  | Idle 45d  | fail     | No                  |
   | temp-fix  | Expired   | fail     | No                  |
5. CLI prints plan + instructions: "Run with --apply to execute"

User: yagwt clean --policy=default --apply

(same flow, but DryRun=false, engine actually removes workspaces)
```

### Example 4: Remove Workspace with Uncommitted Changes

```
User: yagwt rm auth

1. CLI parses selector "auth"
2. CLI invokes: engine.Remove(Selector{Type: Bare, Value: "auth"}, RemoveOptions{})
3. Core engine:
   a. Acquires lock
   b. Resolves selector "auth" → workspace wsp_01HZX... (by name)
   c. Gets git status via git wrapper
   d. Status shows dirty=true (uncommitted changes)
   e. RemoveOptions.OnDirty not set → defaults to "fail"
   f. Returns: Error{Code: E_DIRTY, Message: "Workspace has uncommitted changes"}
4. CLI formats error with hints:
   "Error: Workspace 'auth' has uncommitted changes

   Hints:
   - To stash changes: yagwt rm auth --on-dirty=stash
   - To save as patch: yagwt rm auth --on-dirty=patch
   - To commit as WIP: yagwt rm auth --on-dirty=wip-commit
   - To discard: yagwt rm auth --on-dirty=force --yes"
5. CLI exits with code 1

User: yagwt rm auth --on-dirty=stash

(same flow, but engine stashes changes before removing workspace)
```

---

## Concurrency Model

### Read Operations (Lock-Free)
- **Commands**: ls, show, path, resolve, status
- **Strategy**: Read metadata without lock (eventually consistent)
- **Rationale**: Reads are frequent, contention-free performance is critical

### Write Operations (Exclusive Lock)
- **Commands**: new, rm, rename, move, pin/unpin, lock/unlock, clean --apply, doctor --apply
- **Strategy**:
  1. Acquire exclusive lock (with timeout)
  2. Read current state (metadata + git)
  3. Validate operation (selector resolution, dirty check)
  4. Execute operation (git command + metadata update)
  5. Write metadata atomically (temp file + rename)
  6. Execute hooks
  7. Release lock
- **Rationale**: Prevents concurrent modifications from corrupting metadata

### Lock Contention Handling
- Default timeout: 3 seconds
- If timeout exceeded: Error "Failed to acquire lock (another yagwt process is running)"
- User can override: `--lock-timeout=10s`

---

## Error Handling Strategy

### Structured Error Model

All errors follow this structure:

```go
type Error struct {
    Code    ErrorCode
    Message string
    Details map[string]interface{}
    Hints   []Hint
}

type Hint struct {
    Message string
    Command string  // Optional: suggested command to fix
}
```

### Error Codes

- `E_DIRTY`: Workspace has uncommitted changes
- `E_NOT_FOUND`: Workspace/ref not found
- `E_AMBIGUOUS`: Selector matches multiple workspaces
- `E_GIT`: Git subprocess error
- `E_POLICY`: Cleanup policy violation
- `E_LOCKED`: Workspace is locked
- `E_BROKEN`: Workspace is broken
- `E_CONFLICT`: Operation conflicts with current state
- `E_TIMEOUT`: Lock acquisition timeout
- `E_CONFIG`: Invalid configuration

### Error Display

CLI formats errors with:
1. Error message (human-readable)
2. Details (if helpful)
3. Hints (actionable suggestions)

Example:
```
Error: Workspace 'auth' has uncommitted changes (E_DIRTY)

Details:
- Modified files: 3
- Untracked files: 1

Hints:
- To stash changes: yagwt rm auth --on-dirty=stash
- To save as patch: yagwt rm auth --on-dirty=patch --patch-dir=/tmp/patches
- To discard: yagwt rm auth --on-dirty=force --yes (WARNING: data loss)
```

---

## Testing Strategy

### Unit Tests (90% coverage for core engine)
- Test all business logic in isolation
- Mock git wrapper, metadata store, config loader
- Focus on edge cases (ambiguous selectors, expired ephemerals, broken workspaces)

### Integration Tests
- Test against real git repositories in temp directories
- Test git wrapper operations (add, remove, status)
- Test metadata storage (atomic writes, concurrent access)
- Test lock manager (concurrent processes)

### Golden Tests (JSON schema stability)
- Capture --json output for various commands
- Compare against known-good golden files
- Fail on schema changes (forces explicit versioning)

### End-to-End Tests
- Full CLI command execution in isolated test repos
- Test user workflows (create → modify → cleanup → remove)
- Test interactive prompts (with simulated input)
- Test error scenarios (dirty workspaces, missing refs)

### Performance Tests
- Measure list operation latency (up to 100 workspaces)
- Target: <50ms p95
- Test concurrent operations (lock contention)

---

## Deployment Architecture

### Build Process

```
1. Developer pushes to GitHub (master branch)
2. GitHub Actions triggers:
   a. Lint: golangci-lint
   b. Test: go test ./... -race -coverprofile=coverage.txt
   c. Build: Cross-compile for all platforms:
      - darwin/amd64, darwin/arm64
      - linux/amd64, linux/arm64
   d. Archive binaries: yagwt-<version>-<os>-<arch>.tar.gz
3. On git tag (v*):
   a. Create GitHub Release
   b. Attach binaries
   c. Generate changelog from conventional commits
```

### Distribution

**GitHub Releases**:
- Binaries attached to each release
- Users download and install manually

**Homebrew**:
```ruby
# Formula: bmf/yagwt/yagwt.rb
class Yagwt < Formula
  desc "Yet Another Git Worktree Manager"
  homepage "https://github.com/bmf/yagwt"
  url "https://github.com/bmf/yagwt/releases/download/v1.0.0/yagwt-1.0.0-darwin-arm64.tar.gz"
  sha256 "..."

  def install
    bin.install "yagwt"
  end
end
```

Users install:
```bash
brew tap bmf/yagwt
brew install yagwt
```

---

## Future Architecture Extensions

### TUI (Terminal User Interface)

Add new package:
```
internal/tui/
  app.go                     # Bubble Tea app
  views/
    list.go                  # Workspace list view
    detail.go                # Workspace detail view
  keybindings.go
```

TUI reuses core engine (same as CLI).

### Daemon (Background Service)

Add new binary:
```
cmd/yagwtd/
  main.go                    # Daemon entry point
internal/daemon/
  server.go                  # Unix socket server
  cache.go                   # Git state caching
  watch.go                   # File-watch integration
```

CLI communicates with daemon via Unix socket, falls back to direct mode if daemon not running.

### REST API (for Web UI)

Add new binary:
```
cmd/yagwt-server/
  main.go                    # HTTP server
internal/api/
  handlers.go                # HTTP handlers
  auth.go                    # Authentication
```

Web UI (separate project) consumes REST API.

---

## Appendix: File Layout

```
yagwt/
├── .github/
│   └── workflows/
│       ├── ci.yml                    # Lint, test, build on every push
│       └── release.yml               # Release on git tags
├── .golangci.yml                     # Linter configuration
├── .gitignore
├── Justfile                          # Task automation
├── README.md
├── go.mod
├── go.sum
├── .agent_planning/
│   ├── PROJECT_SPEC.md
│   └── ARCHITECTURE.md
├── cmd/
│   └── yagwt/
│       └── main.go                   # CLI entry point
├── internal/
│   ├── core/                         # Core engine
│   │   ├── engine.go
│   │   ├── workspace.go
│   │   ├── result.go
│   │   ├── selector.go
│   │   ├── cleanup.go
│   │   ├── doctor.go
│   │   ├── create.go
│   │   ├── remove.go
│   │   ├── modify.go
│   │   └── errors.go
│   ├── cli/                          # CLI commands and output
│   │   ├── commands/
│   │   │   ├── root.go
│   │   │   ├── ls.go
│   │   │   ├── new.go
│   │   │   ├── rm.go
│   │   │   ├── clean.go
│   │   │   ├── doctor.go
│   │   │   ├── show.go
│   │   │   └── ... (all commands)
│   │   ├── output/
│   │   │   ├── formatter.go
│   │   │   ├── json.go
│   │   │   ├── porcelain.go
│   │   │   └── table.go
│   │   ├── prompt/
│   │   │   └── interactive.go
│   │   ├── flags.go
│   │   └── errors.go
│   ├── git/                          # Git wrapper
│   │   ├── repo.go
│   │   ├── worktree.go
│   │   ├── status.go
│   │   ├── refs.go
│   │   ├── subprocess.go
│   │   └── errors.go
│   ├── metadata/                     # Metadata storage
│   │   ├── store.go
│   │   ├── json.go
│   │   └── index.go
│   ├── config/                       # Configuration
│   │   ├── config.go
│   │   ├── defaults.go
│   │   ├── validate.go
│   │   └── policies.go
│   ├── lock/                         # Concurrency control
│   │   ├── lock.go
│   │   └── flock.go
│   ├── filter/                       # Filter engine
│   │   ├── parser.go
│   │   ├── eval.go
│   │   └── builtins.go
│   └── hooks/                        # Hook executor
│       ├── executor.go
│       └── discover.go
├── pkg/                              # Public API (if exposing library)
│   └── yagwt/
│       └── client.go                 # Library interface for external consumers
└── testdata/                         # Test fixtures and golden files
    ├── repos/                        # Test git repositories
    ├── configs/                      # Test config files
    └── golden/                       # Golden JSON outputs
```

---

## Appendix: Key Design Principles

1. **Separation of Concerns**: Core engine (business logic) is independent of CLI (presentation)
2. **Testability**: Pure functions, dependency injection, mockable interfaces
3. **Safety**: Never lose uncommitted work without explicit user consent
4. **Stability**: Versioned schemas for machine consumers (JSON, porcelain)
5. **Extensibility**: Architecture supports future interfaces (TUI, daemon, IDE plugins)
6. **Simplicity**: Use stdlib and minimal dependencies, avoid over-engineering
7. **Clarity**: Structured errors with hints, helpful error messages
8. **Performance**: Lock-free reads, fast list operations, efficient metadata storage
