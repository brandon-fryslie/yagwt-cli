# Project Structure Cache
**Confidence**: FRESH
**Date**: 2025-12-18

## Project Layout

```
yagwt-cli/
├── cmd/yagwt/              # Main CLI entry point
├── internal/
│   ├── cli/commands/       # CLI command handlers (minimal)
│   ├── config/             # TOML config loader
│   ├── core/               # Core types (errors, workspace, selector, engine)
│   ├── git/                # Git wrapper (worktree operations)
│   ├── lock/               # Advisory file locking
│   └── metadata/           # JSON metadata storage
├── pkg/                    # Public packages (empty currently)
├── testdata/               # Test fixtures
└── .agent_planning/        # Planning and evaluation docs
    ├── phase1-foundation/  # Phase 1 specific plans
    └── eval-cache/         # Evaluation cache (this)
```

## Package Responsibilities

### internal/git
- Git worktree operations via exec
- Parsing porcelain output (worktree list, status v2)
- Repository detection and validation

### internal/metadata
- JSON-based metadata storage at `<gitdir>/yagwt/meta.json`
- O(1) indexed lookups (by ID, name, path)
- Atomic writes (temp + rename pattern)

### internal/lock
- Advisory file locking using unix.Flock
- Timeout support with exponential backoff
- Concurrent access coordination

### internal/config
- TOML config parsing with github.com/pelletier/go-toml/v2
- Multi-source precedence (explicit > repo > user)
- Validation for rootStrategy and onDirty values

### internal/core
- Error model with wrapping and serialization
- Workspace and selector types
- WorkspaceManager engine (skeleton in Phase 1)

## Test Infrastructure

### Test Commands
- `just test` - Run all tests
- `just test-coverage` - Run with race detector and coverage
- `just build` - Compile binary

### Test Structure
- Unit tests: `*_test.go` in same package
- Integration tests: `integration_test.go` for git operations
- Test utilities: setupTestRepo(), runGit() helpers

### Coverage Measurement
- Per-package coverage via `go test -cover`
- Detailed coverage via `go tool cover -func`
- Race detection via `go test -race`

## Key Patterns

### Error Handling
- Custom error type with codes, details, hints
- Error wrapping preserves underlying error
- Exit code mapping for CLI

### Atomic Operations
- Metadata: write to temp file, then rename
- Lock: flock with timeout and exponential backoff

### Git Integration
- All git operations via exec.Command
- Porcelain output parsing for stability
- Path resolution handles symlinks (macOS /var vs /private/var)
