# YAGWT - Yet Another Git Worktree Manager

A powerful, production-grade CLI tool for managing Git worktrees with lifecycle management, metadata tracking, and intelligent cleanup policies.

## Overview

YAGWT transforms Git worktrees into first-class **workspaces** with rich metadata, lifecycle states, and automation. Stop juggling branches with `git stash` - work on multiple branches simultaneously with zero friction.

### Key Features

- **Workspace Lifecycle**: Track workspaces as pinned, ephemeral (auto-expire), locked, or broken
- **Safe Operations**: Never lose uncommitted work without explicit consent (multiple safety strategies)
- **Intelligent Cleanup**: Policy-driven cleanup of idle/expired workspaces
- **Machine-Readable Output**: Stable JSON and porcelain formats for scripting and IDE integration
- **Flexible Selectors**: Reference workspaces by ID, name, path, or branch
- **Hook System**: Execute custom scripts at lifecycle events (create, remove, open)
- **Doctor/Repair**: Detect and fix broken workspaces automatically

## Getting Started

### Prerequisites

- **Git**: Version 2.30 or newer
- **Go**: Version 1.21+ (for building from source)
- **macOS or Linux**: Windows support is planned

### Installation

#### Homebrew (macOS/Linux)

```bash
brew tap bmf/yagwt
brew install yagwt
```

#### From Source

```bash
git clone https://github.com/bmf/yagwt.git
cd yagwt
just build
sudo cp bin/yagwt /usr/local/bin/
```

### Quick Start

```bash
# List all workspaces in current repo
yagwt ls

# Create a workspace for feature branch
yagwt new feature/auth-improvements --name auth

# Create an ephemeral workspace (auto-expires in 7 days)
yagwt new bugfix/urgent --ephemeral

# Show workspace details
yagwt show auth

# Remove a workspace (safely handles uncommitted changes)
yagwt rm auth

# Cleanup idle workspaces (dry-run first)
yagwt clean --plan
yagwt clean --apply

# Detect and repair broken workspaces
yagwt doctor --plan
yagwt doctor --apply
```

## Mental Model

A repository has **workspaces** instead of just worktrees. A workspace is:

- A git worktree (path + HEAD target + index/working tree)
- Plus metadata (ID, name, flags, activity tracking)
- Plus lifecycle state (pinned, ephemeral, locked, broken)

### Workspace Types

- **Primary**: The original checkout (typically `main` or `master` branch)
- **Secondary**: Additional workspaces created for parallel work

### Lifecycle States

- **Active**: Currently in use
- **Idle**: Not recently accessed, candidate for cleanup
- **Pinned**: Protected from automatic cleanup
- **Ephemeral**: Auto-expires after TTL (default 7 days)
- **Locked**: Protected from removal/modification
- **Broken**: Workspace where git state is inconsistent (needs repair)

## Command Reference

### Read Operations

```bash
# List workspaces
yagwt ls [--all] [--filter=EXPR] [--format=FORMAT] [--fields=FIELDS]

# Show workspace details
yagwt show <selector>

# Print workspace path only
yagwt path <selector>

# Find workspaces with ref checked out
yagwt resolve <ref>

# Show status summary
yagwt status [<selector>]
```

### Create/Ensure

```bash
# Create new workspace
yagwt new <ref> [--name=NAME] [--dir=PATH] [--ephemeral] [--pin] [--ttl=DURATION]

# Idempotent create (returns existing if already exists)
yagwt ensure <branch>
```

### Modify

```bash
# Rename workspace
yagwt rename <selector> <new-name>

# Pin/unpin (protect from cleanup)
yagwt pin <selector>
yagwt unpin <selector>

# Mark as ephemeral (auto-expire)
yagwt ephemeral <selector> [--ttl=DURATION]
yagwt permanent <selector>

# Move to different directory
yagwt move <selector> --dir=<path>

# Lock/unlock (prevent removal)
yagwt lock <selector>
yagwt unlock <selector>
```

### Remove/Cleanup/Repair

```bash
# Remove workspace
yagwt rm <selector> [--delete-branch] [--on-dirty=STRATEGY]

# Cleanup idle/expired workspaces
yagwt clean [--policy=POLICY] [--plan] [--apply] [--max=N]

# Detect and repair broken workspaces
yagwt doctor [--plan] [--apply] [--forget-missing]
```

### Selectors

Commands accept flexible selectors to identify workspaces:

```bash
id:wsp_01HZX...        # By workspace ID (stable, unique)
name:auth              # By name (alias)
path:/abs/path         # By absolute or relative path
branch:feature/x       # By branch name

# Bare selectors try resolution order: id → name → path → branch
yagwt show auth        # Resolves "auth" automatically
```

### Common Flags

All commands support these flags:

- `--repo <path>`: Repository root (default: auto-detect from current directory)
- `--json`: Output in JSON format (machine-readable, schema versioned)
- `--porcelain`: Output in stable porcelain format (tab-separated)
- `--quiet`: Suppress non-essential output
- `--no-prompt`: Never ask questions (fail if input required)
- `--yes` / `-y`: Automatically answer yes to all prompts

## Configuration

YAGWT discovers configuration files in this precedence order:

1. `--config <path>` (CLI flag)
2. `<repoRoot>/.yagwt/config.toml` (team/repo-specific)
3. `~/.config/yagwt/config.toml` (user-specific)
4. Built-in defaults

### Example config.toml

```toml
[workspace]
rootStrategy = "sibling"  # or "inside"
rootDir = ".workspaces"   # only used if rootStrategy = "inside"
nameTemplate = "{branch}" # or custom like "{ticket}-{slug}"

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

## Safety Features

### Handling Uncommitted Changes

YAGWT never loses uncommitted work without explicit consent. When removing a workspace with uncommitted changes, choose a strategy:

- `--on-dirty=fail`: Abort (default, safest)
- `--on-dirty=stash`: Stash changes before removal
- `--on-dirty=patch`: Save changes as patch file to `--patch-dir`
- `--on-dirty=wip-commit`: Commit as WIP with `--wip-message`
- `--on-dirty=force`: Discard changes (requires `--yes`, dangerous!)

### Branch Deletion

Branch deletion requires explicit `--delete-branch` flag. Never automatic.

## Machine-Readable Output

YAGWT provides stable interfaces for scripting and IDE integration:

### JSON Output

```bash
yagwt ls --json
```

```json
{
  "schemaVersion": 1,
  "repo": {
    "root": "/path/to/repo",
    "gitDir": "/path/to/repo/.git"
  },
  "ok": true,
  "result": {
    "workspaces": [
      {
        "id": "wsp_01HZX...",
        "name": "feature-auth",
        "path": "/abs/path",
        "isPrimary": false,
        "target": {
          "type": "branch",
          "ref": "refs/heads/feature/auth",
          "short": "feature/auth",
          "headSha": "abc123..."
        },
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
        "status": {
          "dirty": false,
          "conflicts": false,
          "ahead": 2,
          "behind": 0,
          "branch": "feature/auth",
          "detached": false
        }
      }
    ]
  }
}
```

Schema is versioned and backwards-compatible within major versions.

### Porcelain Output

```bash
yagwt ls --porcelain
```

Tab-separated output (stable, parseable):

```
wsp_01HZX...	feature-auth	/abs/path	branch	feature/auth	false	true	false	false
```

## Exit Codes

YAGWT uses stable exit codes for scripting:

- `0`: Success
- `1`: Generic failure
- `2`: Invalid usage / ambiguous selector / bad flags
- `3`: Safety refusal (dirty + non-interactive + no safe policy)
- `4`: Partial success (batch operations)
- `5`: Not found

## Hooks

Execute custom scripts at lifecycle events:

- `post-create`: After workspace creation
- `pre-remove`: Before workspace removal (can abort)
- `post-remove`: After workspace removal
- `post-open`: After workspace opened (future)

Hooks receive environment variables:

- `YAGWT_REPO_ROOT`: Repository root path
- `YAGWT_WORKSPACE_ID`: Workspace ID
- `YAGWT_WORKSPACE_PATH`: Absolute workspace path
- `YAGWT_WORKSPACE_NAME`: Workspace name
- `YAGWT_TARGET_REF`: Target reference
- `YAGWT_OPERATION`: Operation name

Example hook (`.yagwt/hooks/post-create`):

```bash
#!/bin/bash
# Open workspace in IDE after creation
code "$YAGWT_WORKSPACE_PATH"
```

## Filter Language

Filter workspaces with simple expressions:

```bash
# Single filters
yagwt ls --filter="flag:pinned"
yagwt ls --filter="flag:ephemeral"
yagwt ls --filter="status:dirty"
yagwt ls --filter="activity:idle>30d"

# Combined (AND by default)
yagwt ls --filter="flag:ephemeral status:clean activity:idle>7d"

# OR with comma
yagwt ls --filter="flag:pinned,flag:locked"
```

## Development

### Building

```bash
just build
```

### Testing

```bash
just test
just test-coverage
```

### Linting

```bash
just lint
just fmt
```

### Release

```bash
just release v1.0.0
```

## Project Structure

```
yagwt/
├── cmd/yagwt/           # CLI entry point
├── internal/
│   ├── core/            # Core engine (business logic)
│   ├── cli/             # CLI commands and output
│   ├── git/             # Git wrapper
│   ├── metadata/        # Metadata storage
│   ├── config/          # Configuration
│   ├── lock/            # Concurrency control
│   ├── filter/          # Filter engine
│   └── hooks/           # Hook executor
├── testdata/            # Test fixtures
└── .agent_planning/     # Design documents
    ├── PROJECT_SPEC.md
    └── ARCHITECTURE.md
```

## Documentation

For detailed documentation:

- [PROJECT_SPEC.md](.agent_planning/PROJECT_SPEC.md) - Complete project specification
- [ARCHITECTURE.md](.agent_planning/ARCHITECTURE.md) - Architecture details and design decisions

## Contributing

Contributions are welcome! Please:

1. Read the [PROJECT_SPEC.md](.agent_planning/PROJECT_SPEC.md)
2. Open an issue to discuss major changes
3. Follow the existing code style
4. Add tests for new functionality
5. Ensure all tests pass: `just test`

## License

MIT License - see LICENSE file for details

## Acknowledgments

Built with inspiration from git-worktree and a desire for better developer workflows.