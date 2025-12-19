# Project Specification: YAGWT (Yet Another Git Worktree Manager)

**Generated**: 2025-12-18
**Last Updated**: 2025-12-18
**Scenario**: New Project
**Status**: Active

---

## 1. Project Overview

### Purpose

YAGWT (Yet Another Git Worktree Manager) is a production-grade command-line tool that provides a powerful, intuitive interface for managing Git worktrees. It solves the friction developers face when working with multiple branches simultaneously by providing:

- A mental model centered around "workspaces" (worktrees with metadata and lifecycle management)
- Safe, policy-driven operations that never lose uncommitted work without explicit consent
- Machine-readable output (--json, --porcelain) for scripting and IDE integration
- Intelligent cleanup and maintenance (automatic ephemeral workspace expiry, doctor/repair)

Unlike raw `git worktree` commands, YAGWT tracks workspace metadata, provides lifecycle management (pinned, ephemeral, locked states), and offers powerful selection and filtering capabilities.

### Target Users

1. **Individual developers** who regularly work across multiple branches (feature development, bug fixes, code review)
2. **Development teams** who need shared conventions and policies for workspace management
3. **IDE/tool developers** who want stable, machine-readable interfaces for worktree integration
4. **DevOps/automation scripts** that need programmatic worktree management

### Core Goals

1. **Never lose work**: Provide explicit, safe handling of uncommitted changes with multiple strategies (fail, stash, patch, wip-commit)
2. **Stable automation interfaces**: Maintain backwards-compatible --json/--porcelain output with schema versioning
3. **Zero-friction multi-branch workflow**: Enable instant switching between branches via workspaces without stashing/unstashing
4. **Intelligent cleanup**: Automatically identify and offer removal of idle/ephemeral workspaces with configurable policies
5. **Production-grade reliability**: Handle edge cases (broken workspaces, concurrent operations, git errors) gracefully

### Success Criteria

**User Metrics**:
- Adoption by 1000+ users within 6 months of public release
- Positive feedback on "safety" and "ease of use" from early adopters
- Integration requests from IDE plugin developers

**Technical Metrics**:
- 90%+ test coverage for core engine
- Zero data loss bugs reported in production
- <50ms p95 latency for list operations (up to 100 workspaces)
- Stable JSON schema with zero breaking changes for v1.x releases

**Business Metrics**:
- Public release via Homebrew + GitHub Releases
- Clear documentation enabling new users to be productive within 5 minutes
- Active community contributions (issues, PRs, discussions)

---

## 2. Architecture

### System Overview

YAGWT uses a three-layer architecture that separates concerns:

1. **Core Engine**: Pure library implementing workspace lifecycle operations with no I/O side effects (printing, prompting). Returns structured results that calling code can interpret.

2. **CLI Wrapper**: Thin presentation layer that invokes the core engine, handles user interaction (prompts, confirmations), formats output for humans or machines, and manages error display.

3. **Optional Daemon** (future): Long-lived background service for performance optimization (caching git state) and file-watch event handling.

This separation enables:
- Reusable core logic for multiple interfaces (CLI, TUI, IDE plugins)
- Testability (core engine has zero external dependencies beyond git)
- Clear boundaries between business logic and presentation

### Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     CLI Layer (cmd/yagwt)                    │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐            │
│  │  Command   │  │  Output    │  │  User      │            │
│  │  Parsing   │  │ Formatting │  │ Interaction│            │
│  └────────────┘  └────────────┘  └────────────┘            │
└────────────────────────┬────────────────────────────────────┘
                         │ Calls
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                Core Engine (internal/core)                   │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐            │
│  │ Workspace  │  │ Lifecycle  │  │  Cleanup   │            │
│  │   CRUD     │  │ Management │  │  Policies  │            │
│  └────────────┘  └────────────┘  └────────────┘            │
│  ┌────────────┐  ┌────────────┐                            │
│  │  Selector  │  │   Doctor   │                            │
│  │  Resolver  │  │   Repair   │                            │
│  └────────────┘  └────────────┘                            │
└────────────────────────┬────────────────────────────────────┘
                         │ Uses
                         ▼
┌─────────────────────────────────────────────────────────────┐
│              Support Modules (internal/*)                    │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐            │
│  │    Git     │  │  Metadata  │  │   Config   │            │
│  │  Wrapper   │  │  Storage   │  │   Loader   │            │
│  └────────────┘  └────────────┘  └────────────┘            │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐            │
│  │   Lock     │  │   Filter   │  │   Hooks    │            │
│  │  Manager   │  │   Engine   │  │  Executor  │            │
│  └────────────┘  └────────────┘  └────────────┘            │
└────────────────────────┬────────────────────────────────────┘
                         │ Invokes
                         ▼
                  ┌──────────────┐
                  │  git command │
                  │  (subprocess)│
                  └──────────────┘
```

### Component Responsibilities

#### CLI Layer (cmd/yagwt, internal/cli)
- **Purpose**: User-facing interface for YAGWT
- **Key responsibilities**:
  - Parse command-line arguments and flags
  - Invoke core engine operations
  - Format results for human consumption (tables, colors) or machines (JSON, porcelain)
  - Manage interactive prompts and confirmations
  - Handle --no-prompt, --yes, --quiet modes
- **Dependencies**: Core engine, output formatters
- **Exposes**: Binary executable `yagwt`

#### Core Engine (internal/core)
- **Purpose**: Pure business logic for workspace lifecycle management
- **Key responsibilities**:
  - List workspaces with filtering
  - Create, rename, move, remove workspaces
  - Resolve selectors (name, path, branch, id) to workspace references
  - Evaluate cleanup policies and generate removal plans
  - Detect and repair broken workspaces (doctor)
  - Apply lifecycle flags (pin, ephemeral, lock)
- **Dependencies**: Git wrapper, metadata storage, config, lock manager
- **Exposes**: WorkspaceManager interface with methods returning structured Result types

#### Git Wrapper (internal/git)
- **Purpose**: Abstraction over git subprocess invocation
- **Key responsibilities**:
  - Execute git commands with proper error handling
  - Parse git worktree list output
  - Parse git status output (dirty state, ahead/behind)
  - Validate git repository roots
  - Handle git errors with structured error types
- **Dependencies**: None (stdlib only)
- **Exposes**: Repository interface

#### Metadata Storage (internal/metadata)
- **Purpose**: Persistent storage for workspace metadata
- **Key responsibilities**:
  - Read/write `<gitDir>/yagwt/meta.json` atomically
  - Maintain reverse index (path → workspace ID)
  - Track workspace flags (pinned, ephemeral, locked, broken)
  - Store aliases, TTL/expiry, activity timestamps
  - Assign stable workspace IDs
- **Dependencies**: Lock manager (for atomic writes)
- **Exposes**: MetadataStore interface

#### Config Loader (internal/config)
- **Purpose**: Configuration file discovery and parsing
- **Key responsibilities**:
  - Locate config files (CLI flag → repo → user → system)
  - Parse TOML config files
  - Merge configs with precedence rules
  - Provide default values
  - Validate config schema
- **Dependencies**: None
- **Exposes**: Config struct

#### Lock Manager (internal/lock)
- **Purpose**: Concurrency control for metadata operations
- **Key responsibilities**:
  - Acquire/release advisory file locks (flock/fcntl)
  - Support configurable timeouts
  - Handle lock contention gracefully
  - Enable lock-free reads, exclusive writes
- **Dependencies**: None
- **Exposes**: Lock interface

#### Filter Engine (internal/filter)
- **Purpose**: Parse and evaluate workspace filters
- **Key responsibilities**:
  - Parse filter expressions (flag:pinned, status:dirty, activity:idle>30d)
  - Evaluate filters against workspace objects
  - Support AND/OR logic
- **Dependencies**: None
- **Exposes**: Filter type with Match(workspace) bool method

#### Hook Executor (internal/hooks)
- **Purpose**: Execute user-defined hooks at lifecycle events
- **Key responsibilities**:
  - Discover hook scripts (config + <repoRoot>/.yagwt/hooks/)
  - Execute hooks with environment variables (YAGWT_*)
  - Handle hook failures appropriately
  - Support hooks: post-create, pre-remove, post-remove, post-open
- **Dependencies**: None
- **Exposes**: HookExecutor interface

#### Output Formatters (internal/output)
- **Purpose**: Format workspace data for different output modes
- **Key responsibilities**:
  - Generate human-readable tables with colors
  - Generate JSON output matching schema
  - Generate porcelain output (tab-separated stable format)
  - Respect --quiet, --json, --porcelain flags
- **Dependencies**: None
- **Exposes**: Formatter interface

### Data Flow

**Example: User creates a workspace for feature branch**

1. User runs: `yagwt new feature/auth-improvements --name auth --ephemeral`
2. CLI parses flags, validates arguments
3. CLI invokes: `coreEngine.CreateWorkspace(CreateOptions{Target: "feature/auth-improvements", Name: "auth", Ephemeral: true})`
4. Core engine:
   a. Acquires lock via LockManager
   b. Loads metadata via MetadataStore
   c. Resolves target ref via GitWrapper
   d. Determines workspace path using config.workspace.rootStrategy
   e. Invokes: `git worktree add <path> <ref>` via GitWrapper
   f. Generates stable workspace ID
   g. Stores workspace metadata (name, flags, TTL) via MetadataStore
   h. Executes post-create hook via HookExecutor
   i. Releases lock
   j. Returns: Result{Workspace: {...}, Warnings: []}
5. CLI formats result using OutputFormatter
6. CLI prints: "Created workspace 'auth' at /path/to/workspace (expires in 7 days)"

**Example: Cleanup identifies idle workspaces**

1. User runs: `yagwt clean --policy=default --plan`
2. CLI invokes: `coreEngine.Cleanup(CleanupOptions{Policy: "default", DryRun: true})`
3. Core engine:
   a. Loads metadata (lock-free read)
   b. Lists git worktrees via GitWrapper
   c. Evaluates cleanup policy against each workspace:
      - Expired ephemeral workspaces → RemovalAction{Reason: "Expired", OnDirty: "fail"}
      - Idle >30d + not pinned + clean → RemovalAction{Reason: "Idle", OnDirty: "fail"}
   d. Returns: Result{Actions: [...], Warnings: []}
4. CLI formats actions as a plan (what would be removed)
5. User reviews plan, runs: `yagwt clean --policy=default --apply` to execute

---

## 3. Technology Stack

### Language: Go

**Rationale**:
1. **Single-binary distribution**: Go produces static binaries with zero runtime dependencies, perfect for CLI tool distribution via Homebrew
2. **Cross-platform support**: Native support for macOS (arm64, x86_64) and Linux (x86_64, arm64) without conditional compilation complexity
3. **Excellent subprocess handling**: First-class support for executing and parsing git commands
4. **Fast compilation**: Rapid build times enable fast development iteration
5. **Strong stdlib**: File locking (fcntl), JSON parsing, TOML parsing (via library), filesystem operations all well-supported
6. **Concurrency**: Goroutines enable future daemon implementation without major refactoring

**Alternatives considered**:
- **Rust**: Excellent performance and safety, but slower compilation and steeper learning curve. Go's simplicity is more appropriate for a CLI tool.
- **Python**: Easier for scripting, but distribution (single binary) and performance (startup time) are inferior.
- **Shell (bash)**: Too complex for the scale of this project; testing and error handling would be painful.

### Framework(s): N/A

**Rationale**: Go's stdlib is sufficient for this CLI tool. No heavy framework needed.

### Key Libraries & Dependencies

- **github.com/spf13/cobra**: Command-line interface framework with subcommands, flags, help generation. Industry standard for Go CLIs.
- **github.com/spf13/viper**: Configuration management (TOML parsing, file discovery, precedence). Integrates seamlessly with cobra.
- **github.com/BurntSushi/toml**: TOML parser for config files (viper dependency).
- **github.com/google/uuid**: Stable workspace ID generation (wsp_* IDs).
- **github.com/fatih/color**: Terminal color output for human-readable formatting.
- **github.com/olekukonko/tablewriter**: ASCII table formatting for workspace lists.
- **github.com/stretchr/testify**: Testing assertions and mocking (assert, require, mock packages).
- **golang.org/x/sys/unix**: Low-level file locking (flock, fcntl) for concurrency control.

### Database: None

**Rationale**: Workspace metadata is stored as JSON in `<gitDir>/yagwt/meta.json`. This is simple, human-readable, and sufficient for expected scale (hundreds of workspaces per repo max). No need for database overhead.

### Infrastructure

**Development**:
- Local Go installation (Go 1.21+)
- Standard go toolchain (go build, go test, go mod)
- Just for task automation (build, test, lint, format)

**Production**:
- **Distribution**: GitHub Releases (binaries attached to git tags)
- **Package manager**: Homebrew formula (tap: bmf/yagwt) for easy installation on macOS/Linux
- **Platforms**: macOS (arm64, x86_64), Linux (x86_64, arm64)
- **Hosting**: GitHub (source code, issues, releases)

**Rationale**: This is a CLI tool that runs locally on developer machines. No servers, databases, or cloud services needed. GitHub provides everything (source hosting, CI/CD via Actions, release distribution).

---

## 4. Development Workflow

### Version Control

- **Git workflow**: Feature branches merged to `master` via pull requests
- **Commit messages**: Conventional commits (feat:, fix:, docs:, test:, refactor:)
- **Branch protection**: None initially (single developer), add PR reviews when team grows
- **Rationale**: Simple workflow appropriate for early-stage project. Feature branches isolate work, conventional commits enable automated changelog generation.

### Package Management

- **Tool**: go modules (go.mod, go.sum)
- **Lock files**: go.sum committed to repo for reproducible builds
- **Rationale**: Go modules are the standard. No alternative tools needed.

### Code Quality Tools

- **Linting**: golangci-lint (aggregates staticcheck, govet, errcheck, etc.)
- **Formatting**: gofmt (standard Go formatter)
- **Type checking**: Built into Go compiler
- **Pre-commit hooks**: None initially (manual linting), add later if needed

**Configuration files**: .golangci.yml for linter settings

**Rationale**: Go's tooling is excellent out of the box. golangci-lint catches most issues, gofmt ensures consistency.

### Testing Approach

- **Framework**: Go stdlib testing package + testify for assertions
- **Test levels**:
  - **Unit tests**: 90%+ coverage for core engine (all business logic)
  - **Integration tests**: Git wrapper, metadata storage, lock manager (test against real git repos in temp dirs)
  - **Golden tests**: JSON output schema validation (ensure --json output matches documented schema exactly)
  - **End-to-end tests**: Full CLI command execution in isolated test repos
- **Mocking strategy**:
  - Mock git wrapper for core engine tests (fast, isolated)
  - Use real git repos in temp directories for integration tests
  - Never mock filesystem or file locking (test real behavior)
- **Coverage goals**: 90% for core engine, 80% overall
- **CI execution**: All tests on every push (GitHub Actions)

**Philosophy**:
- Write tests alongside code (not strictly TDD, but test-aware design)
- Focus on testing real behavior, not implementation details
- Golden tests are critical for --json stability (machine consumers depend on schema)

### CI/CD

- **Platform**: GitHub Actions
- **Pipeline stages**:
  1. **Lint**: golangci-lint
  2. **Test**: go test ./... with coverage report
  3. **Build**: Cross-compile for all target platforms (macOS arm64/x86_64, Linux x86_64/arm64)
  4. **Release** (on git tags): Attach binaries to GitHub Release
- **Deployment triggers**:
  - Lint + test on every push/PR
  - Build on every push to master
  - Release on git tags matching v* (e.g., v1.0.0)

**Rationale**: GitHub Actions is free for public repos, well-integrated with GitHub Releases. Simple YAML workflow files are easy to maintain.

---

## 5. Architecture Decisions

### ADR-001: Choose Go Over Rust or Python
**Date**: 2025-12-18
**Status**: Accepted

**Context**: Need to choose implementation language for a cross-platform CLI tool with git subprocess integration and single-binary distribution.

**Decision**: Use Go

**Options Considered**:
1. **Go**: Mature stdlib, fast compilation, easy cross-compilation, single binary
2. **Rust**: Best performance, memory safety, but slower compilation and steeper learning curve
3. **Python**: Easy scripting, but distribution (PyInstaller) is clunky and startup time is slow

**Rationale**:
- Single-binary distribution is critical for Homebrew packaging and user experience
- Go's stdlib has excellent subprocess handling (critical for git integration)
- Compilation speed enables rapid iteration
- Go's simplicity matches project complexity (this is not a systems programming project where Rust's safety guarantees are essential)

**Consequences**:
- **Positive**: Fast builds, easy distribution, strong stdlib
- **Negative**: Manual memory management (but Go's GC handles this well), less compile-time safety than Rust
- **Risks**: None significant

---

### ADR-002: Three-Layer Architecture (Engine, CLI, Daemon)
**Date**: 2025-12-18
**Status**: Accepted

**Context**: Need architecture that supports current CLI use case and future expansions (TUI, IDE plugins, daemon).

**Decision**: Separate core engine (pure library) from CLI wrapper from optional daemon.

**Options Considered**:
1. **Monolithic CLI**: All logic in cmd/yagwt main package
2. **Two layers**: Engine + CLI
3. **Three layers**: Engine + CLI + Daemon

**Rationale**:
- Core engine with zero I/O side effects is highly testable (no mocking prompts, no capturing stdout)
- Multiple interfaces can reuse engine (TUI, IDE plugins)
- Daemon is future work, but architecture accommodates it without refactoring

**Consequences**:
- **Positive**: Clear boundaries, testable core, extensible
- **Negative**: Slightly more boilerplate (engine returns structured results, CLI formats them)
- **Risks**: Over-engineering if we never build TUI/plugins (mitigated: current CLI benefits from testability)

---

### ADR-003: JSON Schema Versioning for Stability
**Date**: 2025-12-18
**Status**: Accepted

**Context**: Machine consumers (scripts, IDE plugins) depend on stable --json output. Breaking changes are painful.

**Decision**: Include schemaVersion in all JSON responses. Major version bumps (v1 → v2) indicate breaking changes. Minor/patch versions maintain backwards compatibility.

**Options Considered**:
1. **No versioning**: Hope we never break compatibility
2. **Schema version in JSON**: schemaVersion field
3. **API version in flag**: --json-version=1

**Rationale**:
- schemaVersion in JSON enables consumers to detect schema and adapt
- Tied to YAGWT release version: v1.x.x → schemaVersion 1, v2.x.x → schemaVersion 2
- Golden tests enforce schema stability within major version

**Consequences**:
- **Positive**: Machine consumers can trust stability, clear breaking change signaling
- **Negative**: Constraints on future JSON changes (must maintain backwards compat within major version)
- **Risks**: None

---

### ADR-004: Selector System for Workspace Addressing
**Date**: 2025-12-18
**Status**: Accepted

**Context**: Users need flexible ways to reference workspaces (by name, path, branch, ID).

**Decision**: Implement selector syntax: `id:wsp_123`, `name:auth`, `path:/abs/path`, `branch:feature/x`. Bare values resolved in order: id → name → path → branch. Ambiguity fails in non-interactive mode with E_AMBIGUOUS (exit 2).

**Options Considered**:
1. **Only by name**: Too limiting (what if user doesn't set names?)
2. **Only by path**: Requires full paths (annoying)
3. **Selector system**: Flexible, explicit when needed

**Rationale**:
- Users want convenience ("yagwt rm auth" when they've named it)
- Users need precision when ambiguous ("yagwt rm path:/abs/path" when multiple workspaces target same branch)
- Scripts need stability (always use id: selectors)

**Consequences**:
- **Positive**: Flexible UX, unambiguous in scripts
- **Negative**: Complexity in resolver logic
- **Risks**: Ambiguity frustration (mitigated: clear error messages with hints)

---

### ADR-005: Stable Exit Codes for Scripting
**Date**: 2025-12-18
**Status**: Accepted

**Context**: Scripts need to detect specific failure modes (not found, ambiguous selector, safety refusal).

**Decision**: Define stable exit codes:
- 0: success
- 1: generic failure
- 2: invalid usage / ambiguous selector
- 3: safety refusal (dirty + non-interactive)
- 4: partial success (batch operations)
- 5: not found

**Options Considered**:
1. **Always exit 0 or 1**: Simple, but scripts can't distinguish failure modes
2. **Stable exit codes**: More complex, but enables scripting

**Rationale**:
- Scripts can handle "not found" differently than "ambiguous selector"
- Safety refusals (exit 3) can be handled with --on-dirty flag
- Documented in --help and manual

**Consequences**:
- **Positive**: Scriptable, clear failure modes
- **Negative**: Must maintain exit code stability (breaking change if we change meanings)
- **Risks**: None

---

### ADR-006: Advisory File Locking for Concurrency
**Date**: 2025-12-18
**Status**: Accepted

**Context**: Multiple yagwt processes might run concurrently (e.g., user runs `yagwt ls` while another `yagwt rm` is in progress).

**Decision**: Use advisory file locking (flock on Linux, fcntl on macOS) for metadata writes. Reads are lock-free. Lock file: `<gitDir>/yagwt/lock`.

**Options Considered**:
1. **No locking**: Risk of corrupted metadata
2. **Advisory file locking**: Standard Unix approach
3. **Distributed lock (e.g., Redis)**: Overkill for local tool

**Rationale**:
- Metadata corruption is unacceptable
- Advisory locks are lightweight and standard
- Lock-free reads enable fast list operations even during writes

**Consequences**:
- **Positive**: Safe concurrent access, minimal performance impact
- **Negative**: Requires platform-specific code (golang.org/x/sys/unix)
- **Risks**: Lock contention (mitigated: configurable timeouts, fast operations)

---

### ADR-007: TOML Configuration Files
**Date**: 2025-12-18
**Status**: Accepted

**Context**: Need human-readable, editable configuration format.

**Decision**: Use TOML for config files (workspace.rootStrategy, cleanup policies, hooks).

**Options Considered**:
1. **JSON**: Machine-readable, but poor for human editing (no comments)
2. **YAML**: Human-readable, but complex spec (indentation sensitivity)
3. **TOML**: Human-readable, simple, supports comments

**Rationale**:
- TOML is designed for config files (comments, clear sections)
- Well-supported in Go ecosystem (viper has built-in TOML support)
- Simpler than YAML (no indentation traps)

**Consequences**:
- **Positive**: Human-readable, good library support
- **Negative**: Less common than JSON/YAML (but simple enough to learn)
- **Risks**: None

---

### ADR-008: Metadata Storage in <gitDir>/yagwt/
**Date**: 2025-12-18
**Status**: Accepted

**Context**: Need persistent storage for workspace metadata (IDs, aliases, flags, TTL).

**Decision**: Store metadata in `<gitDir>/yagwt/meta.json`. Use JSON for human-readability.

**Options Considered**:
1. **SQLite database**: Structured queries, but overkill for simple data
2. **JSON file**: Simple, human-readable, no dependencies
3. **Git notes**: Clever, but fragile (notes can be lost, complex to manage)

**Rationale**:
- JSON is simple, human-readable, and sufficient for expected scale
- Atomic writes (write to temp file, rename) prevent corruption
- File locking ensures concurrency safety

**Consequences**:
- **Positive**: Simple, debuggable (users can inspect meta.json), no database dependencies
- **Negative**: Must load entire file for every operation (acceptable for <1000 workspaces)
- **Risks**: Scale limit (mitigated: workspaces rarely exceed hundreds per repo)

---

### ADR-009: Policy-Driven Cleanup vs Manual Pruning
**Date**: 2025-12-18
**Status**: Accepted

**Context**: Users need to clean up stale workspaces, but policies vary (aggressive vs conservative).

**Decision**: Implement policy-driven cleanup with built-in policies (default, conservative, aggressive) and support for custom policies in config. Always show plan before applying (--plan flag, --apply to execute).

**Options Considered**:
1. **Manual only**: User specifies each workspace to remove
2. **Automatic aggressive**: Remove everything idle >7d
3. **Policy-driven with plan**: Flexible, safe

**Rationale**:
- Different users have different needs (personal repos vs team repos)
- Plan-then-apply prevents accidental data loss
- Policies are configurable in config files

**Consequences**:
- **Positive**: Flexible, safe, auditable
- **Negative**: Requires policy evaluation logic
- **Risks**: None

---

### ADR-010: Aggressive Roadmap (No Intermediate Implementations)
**Date**: 2025-12-18
**Status**: Accepted

**Context**: User explicitly requested aggressive roadmap that goes straight to full functionality, no intermediate implementations.

**Decision**: Implement complete command surface in initial release. No MVP with reduced functionality.

**Options Considered**:
1. **MVP approach**: Start with basic ls, new, rm commands
2. **Aggressive full implementation**: All commands from day one

**Rationale**:
- User wants complete solution immediately
- Architecture supports full implementation without phasing
- All commands are interconnected (cleanup depends on flags, flags depend on metadata, etc.)

**Consequences**:
- **Positive**: Complete functionality from v1.0
- **Negative**: Longer time to first release (acceptable per user)
- **Risks**: Larger initial implementation burden (mitigated: clear spec)

---

## 6. Implementation Roadmap

**IMPORTANT**: This is an aggressive roadmap with NO intermediate implementations. All phases are part of the initial v1.0 release. Phases indicate implementation order and dependency, not separate releases.

### Phase 1: Foundation (Core Infrastructure)

**Goal**: Establish core infrastructure and git integration

**Deliverables**:
- Go module initialization (github.com/bmf/yagwt)
- Project structure (cmd/, internal/ packages)
- Git wrapper (Repository interface, worktree listing, status parsing)
- Metadata storage (MetadataStore interface, JSON read/write, atomic operations)
- Lock manager (advisory file locking, timeout handling)
- Config loader (TOML parsing, precedence, defaults)
- Error model (structured errors with codes, hints)
- Exit code handling

**Acceptance Criteria**:
- Git wrapper can list worktrees, parse status, handle errors
- Metadata can be written/read atomically with locking
- Config files can be loaded from multiple locations with precedence
- All infrastructure has 90%+ test coverage

**Complexity**: Large (establishes entire foundation)

**Dependencies**: None

---

### Phase 2: Core Engine (Business Logic)

**Goal**: Implement workspace lifecycle operations

**Deliverables**:
- WorkspaceManager interface
- List workspaces (with git worktree integration + metadata merging)
- Create workspace (git worktree add + metadata creation, ID assignment)
- Remove workspace (git worktree remove + metadata cleanup)
- Rename, move, pin/unpin, lock/unlock operations
- Selector resolver (id:, name:, path:, branch:, bare value resolution)
- Workspace flags (pinned, ephemeral, locked, broken)
- TTL/expiry tracking for ephemeral workspaces
- Activity tracking (lastOpenedAt, lastGitActivityAt)

**Acceptance Criteria**:
- All CRUD operations work correctly
- Selectors resolve unambiguously or fail with E_AMBIGUOUS
- Metadata is synced with git worktree state
- 90%+ test coverage for core engine
- Integration tests with real git repos in temp directories

**Complexity**: Large (core business logic)

**Dependencies**: Phase 1 complete

---

### Phase 3: Advanced Features (Cleanup, Doctor, Filters)

**Goal**: Implement intelligent maintenance operations

**Deliverables**:
- Cleanup policy engine (default, conservative, aggressive policies)
- Cleanup plan generation (identify removal candidates)
- Cleanup execution (remove workspaces per policy)
- Doctor/repair (detect broken workspaces, prune stale metadata, forget missing)
- Filter engine (parse and evaluate filter expressions)
- Filter language support (flag:, status:, target:, activity:)
- On-dirty strategies (fail, stash, patch, wip-commit, force)

**Acceptance Criteria**:
- Cleanup policies correctly identify removal candidates
- Doctor detects and repairs broken workspaces
- Filters correctly match workspaces
- On-dirty strategies preserve uncommitted work safely
- 90%+ test coverage

**Complexity**: Large (complex policy evaluation logic)

**Dependencies**: Phase 2 complete

---

### Phase 4: CLI Commands (User Interface)

**Goal**: Implement complete command surface

**Deliverables**:
- Cobra command structure with all subcommands
- Read commands: ls, show, path, resolve, status
- Create/ensure commands: new, ensure
- Modify commands: rename, pin/unpin, ephemeral/permanent, move, lock/unlock
- Remove/cleanup/repair commands: rm, clean, doctor
- Flag parsing for all commands (common flags: --repo, --json, --porcelain, --quiet, --no-prompt, --yes)
- Command-specific flags (e.g., rm --delete-branch, clean --policy)
- Selector parsing and validation
- Interactive prompts and confirmations
- Error display with hints

**Acceptance Criteria**:
- All commands invoke core engine correctly
- Flags are parsed and validated
- Prompts work in interactive mode, skipped in non-interactive
- Error messages are clear with actionable hints
- E2E tests for all commands

**Complexity**: Large (complete CLI surface)

**Dependencies**: Phase 3 complete

---

### Phase 5: Output Formatting (Machine Interfaces)

**Goal**: Implement stable machine-readable output

**Deliverables**:
- JSON output formatter (schemaVersion, stable schema)
- Porcelain output formatter (tab-separated, stable columns)
- Human-readable table formatter (colors, formatting)
- Quiet mode (suppress non-essential output)
- Golden tests for JSON schema stability
- Documentation of JSON schema and porcelain format

**Acceptance Criteria**:
- --json output matches documented schema exactly
- Golden tests enforce schema stability
- --porcelain output is stable and parseable
- Human output is readable and helpful
- All output modes tested

**Complexity**: Medium (mostly formatting logic)

**Dependencies**: Phase 4 complete

---

### Phase 6: Hooks and Extensibility

**Goal**: Enable user customization via hooks

**Deliverables**:
- Hook executor (discover, execute, handle failures)
- Hook types: post-create, pre-remove, post-remove, post-open
- Environment variables: YAGWT_REPO_ROOT, YAGWT_WORKSPACE_ID, YAGWT_WORKSPACE_PATH, YAGWT_TARGET_REF, YAGWT_WORKSPACE_NAME, YAGWT_OPERATION
- Hook configuration in config files
- Hook discovery in <repoRoot>/.yagwt/hooks/
- Error handling for hook failures

**Acceptance Criteria**:
- Hooks execute at correct lifecycle events
- Environment variables are set correctly
- Hook failures are reported clearly
- Integration tests with real hook scripts

**Complexity**: Medium (subprocess execution, environment setup)

**Dependencies**: Phase 5 complete

---

### Phase 7: Documentation and Polish

**Goal**: Prepare for public release

**Deliverables**:
- Comprehensive README.md with examples
- Manual pages (man yagwt, man yagwt-ls, etc.)
- JSON schema documentation (stable contracts)
- Porcelain format documentation
- Configuration file examples
- Homebrew formula
- GitHub Actions CI/CD pipeline
- Cross-platform builds (macOS arm64/x86_64, Linux x86_64/arm64)
- Release automation (attach binaries to GitHub Releases)
- Changelog generation

**Acceptance Criteria**:
- README enables new users to be productive in 5 minutes
- All commands documented in --help and man pages
- Homebrew formula successfully installs from tap
- CI/CD pipeline runs on every push
- Releases are automated

**Complexity**: Medium (documentation and automation)

**Dependencies**: Phase 6 complete

---

### Phase 8: Testing and Hardening

**Goal**: Ensure production-grade reliability

**Deliverables**:
- 90%+ test coverage for core engine
- 80%+ test coverage overall
- Integration tests for all git operations
- E2E tests for all CLI commands
- Golden tests for JSON schema
- Concurrency tests (multiple processes)
- Error scenario tests (broken repos, missing metadata, lock contention)
- Performance tests (<50ms p95 for list operations)

**Acceptance Criteria**:
- All tests pass on all target platforms
- Coverage goals met
- No known bugs
- Performance goals met

**Complexity**: Large (comprehensive testing)

**Dependencies**: Phase 7 complete

---

### Release: v1.0.0

**Goal**: Public release via Homebrew + GitHub Releases

**Deliverables**:
- Tag v1.0.0
- GitHub Release with binaries for all platforms
- Homebrew formula published to tap
- Announcement (GitHub Discussions, social media)

**Acceptance Criteria**:
- Users can install via `brew install bmf/yagwt/yagwt`
- All functionality works on macOS (arm64, x86_64) and Linux (x86_64, arm64)
- Zero known critical bugs

---

## 7. Future Considerations

### Deferred Decision: TUI (Terminal User Interface)

**Current approach**: CLI only, no interactive UI beyond prompts

**When to revisit**:
- When user feedback indicates desire for visual interface
- When core engine is stable (v1.x released and hardened)

**Upgrade path**:
1. Implement TUI in separate package (internal/tui) using bubbles/bubbletea
2. Reuse core engine (already designed for this)
3. Add `yagwt tui` subcommand to launch interactive mode
4. Estimated effort: Medium (2-3 weeks)

---

### Deferred Decision: Long-lived Daemon

**Current approach**: Each yagwt invocation is short-lived, parses git state fresh

**When to revisit**:
- When list performance degrades (repos with >100 workspaces)
- When users request file-watch integration (detect external worktree changes)

**Upgrade path**:
1. Implement daemon in separate binary (cmd/yagwtd)
2. Daemon caches git state, invalidates on file-watch events
3. CLI communicates with daemon via Unix socket
4. Fallback to direct mode if daemon not running
5. Estimated effort: Large (3-4 weeks)

**Risks**: Daemon complexity (startup, shutdown, error recovery). Mitigation: Make daemon purely optional (CLI works without it).

---

### Deferred Decision: IntelliJ Plugin

**Current approach**: CLI only, no IDE integration

**When to revisit**:
- When v1.0 is released and stable
- When users request IDE integration
- When JSON output is proven stable

**Upgrade path**:
1. IntelliJ plugin (separate repo) invokes yagwt with --json flag
2. Parses JSON responses (schema versioned)
3. Integrates with IDE UI (tool window, actions)
4. Estimated effort: Large (4+ weeks, requires Java/Kotlin expertise)

---

### Deferred Decision: Web UI

**Current approach**: CLI only

**When to revisit**:
- If team workflows require visual workspace management
- If remote development becomes primary use case

**Upgrade path**:
1. Implement REST API server (cmd/yagwt-server) using core engine
2. Build web UI (separate project) consuming API
3. Authentication/authorization for team use
4. Estimated effort: Large (4+ weeks)

**Risks**: Adds server hosting complexity. Mitigation: Keep CLI as primary interface, web UI as optional addon.

---

### Deferred Decision: Windows Support

**Current approach**: macOS and Linux only

**When to revisit**:
- When users on Windows request support (monitor GitHub issues)
- When core functionality is stable on macOS/Linux

**Upgrade path**:
1. Test git wrapper on Windows (git.exe subprocess)
2. Implement Windows file locking (different API than Unix)
3. Test on Windows (GitHub Actions supports Windows runners)
4. Add Windows builds to CI/CD
5. Estimated effort: Medium (2 weeks)

**Risks**: Git behavior differences on Windows. Mitigation: Extensive testing.

---

### Deferred Decision: Remote Worktree Support

**Current approach**: All workspaces on local filesystem

**When to revisit**:
- If remote development (containers, SSH) becomes common use case
- If users request ability to manage workspaces on remote machines

**Upgrade path**:
1. Implement remote backend (SSH, container exec)
2. Add --remote flag to commands
3. Proxy git operations to remote machine
4. Estimated effort: Large (3-4 weeks)

**Risks**: Complexity, latency, authentication. Mitigation: Keep local as primary, remote as advanced feature.

---

## 8. Open Questions & Risks

### Open Questions

1. **Workspace ID format**: Use wsp_<uuid> or wsp_<ulid>?
   - **Impact**: IDs are visible to users in --json output, stored in metadata
   - **Who decides**: Developer (research UUIDs vs ULIDs for CLI use case)
   - **Deadline**: Before Phase 2 (ID generation implementation)

2. **Default cleanup policy**: How aggressive should "default" policy be?
   - **Impact**: Users might accidentally lose work if too aggressive
   - **Who decides**: User feedback (start conservative, tune based on feedback)
   - **Deadline**: Before Phase 3 (policy implementation)

3. **Hook failure handling**: Should hook failures abort operations or just warn?
   - **Impact**: Affects reliability vs flexibility tradeoff
   - **Who decides**: Developer (research git hook behavior for consistency)
   - **Deadline**: Before Phase 6 (hook implementation)

### Risks

1. **Git compatibility**: git worktree behavior might differ across versions
   - **Likelihood**: Medium (git is stable, but edge cases exist)
   - **Impact**: High (core functionality depends on git)
   - **Mitigation**: Test against multiple git versions (2.30+), document minimum version
   - **Owner**: Developer

2. **Concurrency bugs**: File locking might not work perfectly on all platforms
   - **Likelihood**: Low (advisory locks are standard Unix feature)
   - **Impact**: High (metadata corruption is unacceptable)
   - **Mitigation**: Extensive concurrency testing, atomic file operations
   - **Owner**: Developer

3. **Performance**: Listing 100+ workspaces might be slow
   - **Likelihood**: Medium (git worktree list is fast, but metadata merging adds overhead)
   - **Impact**: Medium (users tolerate some latency for lists)
   - **Mitigation**: Performance testing, optimize hot paths, consider daemon for caching (future)
   - **Owner**: Developer

4. **Schema evolution**: JSON schema might need breaking changes
   - **Likelihood**: Medium (hard to predict all future needs)
   - **Impact**: High (breaking changes anger machine consumers)
   - **Mitigation**: Schema versioning, careful design, community feedback before v1.0
   - **Owner**: Developer

---

## Appendix A: Glossary

**Workspace**: A git worktree with associated metadata (ID, name, flags, activity timestamps). The fundamental abstraction YAGWT manages.

**Primary workspace**: The original checkout (what you get with `git clone`). Typically the "main" or "master" branch.

**Secondary workspace**: Additional worktrees created with `git worktree add` (via YAGWT `new` command).

**Selector**: A way to reference a workspace in commands. Syntax: `id:<uuid>`, `name:<alias>`, `path:<path>`, `branch:<branch>`. Bare values resolved automatically.

**Ephemeral workspace**: Workspace with TTL (time-to-live). Automatically expires after inactivity period (default 7 days). Used for temporary work.

**Pinned workspace**: Workspace protected from automatic cleanup. Used for long-lived work.

**Locked workspace**: Workspace where destructive operations (rm, move) are blocked. Used to prevent accidental deletion.

**Broken workspace**: Workspace where git state is inconsistent (e.g., worktree path deleted externally). Detected by `doctor` command.

**Policy**: Rule set for cleanup operations. Built-in policies: default, conservative, aggressive. Custom policies in config files.

**Hook**: User-defined script executed at workspace lifecycle events (post-create, pre-remove, post-remove, post-open).

**On-dirty strategy**: How to handle uncommitted changes during removal. Options: fail, stash, patch, wip-commit, force.

**Golden test**: Test that compares actual output against known-good "golden" file. Used to enforce JSON schema stability.

**ADR**: Architecture Decision Record. Documents major decisions with rationale, alternatives, and consequences.

---

## Appendix B: References

- [Git worktree documentation](https://git-scm.com/docs/git-worktree)
- [Cobra CLI framework](https://github.com/spf13/cobra)
- [Conventional commits specification](https://www.conventionalcommits.org/)
- [Semantic versioning](https://semver.org/)
- [Go modules reference](https://go.dev/ref/mod)
- [golangci-lint linters](https://golangci-lint.run/usage/linters/)

---

## Generation History

**Generated**: 2025-12-18
**Agent**: project-architect
**Scenario**: New Project
**Handoff**: Recommended next step: Review spec, then proceed to scaffolding
