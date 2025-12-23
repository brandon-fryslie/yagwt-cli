# Enhanced Developer Workflows Proposal

## The Problem

Developers using git worktrees face friction beyond just managing the worktrees themselves. The current YAGWT implementation handles workspace lifecycle beautifully, but there are adjacent workflow problems that cause developers to context-switch between tools:

1. **Branch Discovery Pain**: When reviewing code or checking out a colleague's work, developers must first identify the branch name, then create a workspace. This requires switching between GitHub/GitLab/email and the terminal.

2. **Dependency Setup Overhead**: After creating a new workspace, developers manually run the same setup commands every time: `npm install`, `bundle install`, database migrations, etc. This wastes 2-5 minutes per workspace.

3. **Lost Context Between Sessions**: When returning to a workspace after days/weeks, developers forget what they were working on. Git branch names rarely capture enough context, and commit messages might not exist yet for WIP work.

4. **Workspace Discovery is Primitive**: `yagwt ls` shows workspaces, but finding "the workspace where I was fixing that auth bug" requires remembering the name or path you chose weeks ago.

5. **No Cross-Workspace Awareness**: Developers often need to compare files or cherry-pick commits between workspaces, but there's no built-in support for these workflows.

## The Vision

YAGWT becomes the **intelligent hub** for multi-branch development. Instead of just managing workspace lifecycle, it understands your workflow patterns and eliminates repetitive work. Developers spend less time on mechanics and more time on creative problem-solving.

When you create a workspace, it's **ready to use immediately** - dependencies installed, environment configured. When you list workspaces, you see **rich context** about what you were doing. When you search, you find workspaces **by the problem they solve**, not just the branch name.

## Selected Ideas

### Idea 1: Smart Workspace Creation from URLs

**User Story**: As a developer reviewing a pull request, I want to create a workspace directly from the PR URL so that I can start reviewing code in under 10 seconds.

**The Experience**:

A developer receives a Slack message: "Can you review https://github.com/acme/app/pull/1234?"

They run:
```bash
yagwt new pr#1234
```

YAGWT:
1. Fetches PR metadata via GitHub API (branch name, title, description)
2. Creates workspace named `pr-1234-fix-auth-timeout`
3. Checks out the PR branch
4. Adds PR title + description to workspace notes
5. Optionally runs post-create hooks (install dependencies, start services)

Within seconds, they're reviewing code in a properly configured workspace. The workspace name is meaningful, and the context (PR description) is preserved.

**Why This Matters**:

Eliminates 4-5 manual steps and context switches:
- Finding the branch name from the PR
- Typing `yagwt new feature/fix-auth-timeout-issue-1234`
- Opening the PR in browser to remember what it's about
- Copy-pasting context into notes somewhere

**Success Looks Like**:
- Developer creates workspace from PR URL in one command
- Workspace name auto-generated from PR metadata (human-readable)
- PR context (title, description, labels) stored in workspace metadata
- `yagwt show <workspace>` displays PR context
- Works with GitHub, GitLab, Bitbucket APIs

**Extensions**:
- `yagwt new issue#456` - create from issue
- `yagwt new commit/abc123def` - create from specific commit
- `yagwt new https://gitlab.com/...` - full URL support

---

### Idea 2: Automatic Workspace Setup with Templates

**User Story**: As a developer, I want my workspaces to be **immediately usable** after creation so that I don't waste 2-5 minutes running setup commands every time.

**The Experience**:

A developer maintains a `.yagwt/setup-template.sh`:
```bash
#!/bin/bash
# Runs automatically for all new workspaces
npm install
cp .env.example .env
./scripts/db-migrate.sh
```

When they create a workspace:
```bash
yagwt new feature/new-dashboard
```

YAGWT:
1. Creates the worktree
2. Detects setup template script
3. Executes it in the new workspace directory
4. Captures output (shown in `yagwt show`)
5. Marks workspace as "ready" or "setup-failed"

They can immediately start coding - dependencies installed, environment configured.

**Why This Matters**:

Developers create workspaces frequently (daily for some). Manual setup creates friction:
- "I'll just stick with my current branch instead of creating a workspace" (defeats the purpose)
- Lost time: 2-5 min/workspace × 10 workspaces/week = 30-50 min/week wasted
- Error-prone: forgetting a step breaks things later

Automation removes this friction entirely.

**Success Looks Like**:
- Template script runs automatically on workspace creation
- Output captured and accessible via `yagwt show <workspace>`
- Failed setup clearly indicated (workspace flagged, error message saved)
- Multiple templates supported (production, testing, minimal)
- Async execution supported for long-running setup (with status tracking)

**Extensions**:
- Project-type detection (Node.js → npm install, Ruby → bundle install)
- Template marketplace/sharing (common setups for popular frameworks)
- Conditional templates based on branch type (feature/*, hotfix/*)

---

### Idea 3: Workspace Notes and Context Capture

**User Story**: As a developer, I want to capture **why I created this workspace** and **what I was doing** so that when I return days later, I remember instantly.

**The Experience**:

Developer creates a workspace for a complex bug:
```bash
yagwt new bugfix/memory-leak --note "Investigating memory leak in user sessions - repro steps in ticket #789"
```

Two weeks later, they run:
```bash
yagwt ls
```

Output shows:
```
NAME              BRANCH              LAST ACTIVE  NOTE
prod-deploy       release/v2.1        2 hours ago  Final testing before deploy
memory-leak       bugfix/memory-leak  14 days ago  Investigating memory leak in user sessions - repro...
```

They switch back:
```bash
yagwt show memory-leak
```

Sees:
```
Workspace: memory-leak
Branch: bugfix/memory-leak
Note: Investigating memory leak in user sessions - repro steps in ticket #789
Files changed: 3 modified, 1 new
Last commit: WIP: Add logging to session handler
```

They remember exactly what they were doing and pick up where they left off.

**Why This Matters**:

Context switching is expensive. When you have 5-10 active workspaces, remembering which one has the auth fix vs the database migration vs the refactor is hard. Branch names are technical, not contextual.

Notes make workspaces **self-documenting**. You capture intent when it's fresh, retrieve it when needed.

**Success Looks Like**:
- Notes captured via `--note` flag or interactive prompt
- Notes displayed in `yagwt ls` (truncated) and `yagwt show` (full)
- Notes editable via `yagwt note <workspace> "new note"`
- Notes searchable via `yagwt search "memory leak"`
- Notes optional (doesn't break existing workflows)

**Extensions**:
- Automatic note suggestions from commit messages
- Rich notes with Markdown formatting
- Notes synced with issue tracker (two-way)

---

### Idea 4: Workspace Search and Discovery

**User Story**: As a developer with 10+ workspaces, I want to **find workspaces by what they do**, not by arbitrary names I gave them weeks ago.

**The Experience**:

Developer remembers working on "something with authentication" but forgets the workspace name:

```bash
yagwt search auth
```

Results:
```
Found 3 workspaces matching "auth":

1. pr-1234-fix-auth-timeout (feature/fix-auth-timeout)
   Note: Fix timeout issue in OAuth flow
   Last active: 3 days ago
   Status: clean, 2 commits ahead

2. auth-refactor (refactor/auth-layer)
   Note: Simplify authentication middleware
   Last active: 2 weeks ago
   Status: dirty, WIP changes

3. oauth-migration (feature/oauth2-upgrade)
   Note: Migrate from OAuth 1.0 to 2.0
   Last active: 1 month ago
   Status: clean, merged to main
```

They quickly identify the right workspace and switch to it.

**Why This Matters**:

As workspace count grows, discovery becomes painful. `yagwt ls` shows everything, but finding the right one requires scrolling and visual scanning. Human memory works by association (I remember the **problem**, not the branch name).

Search enables **intent-based discovery**: find by what you were doing, not what you named it.

**Success Looks Like**:
- Full-text search across workspace names, notes, branch names, commit messages
- Fuzzy matching (typo-tolerant)
- Search results ranked by relevance and recency
- Fast (<100ms for 100 workspaces)
- Integrates seamlessly with existing commands

**Extensions**:
- Search by date: `yagwt search --since="2 weeks ago"`
- Search by file: `yagwt search --touched="auth.js"`
- Search by author: `yagwt search --commits-by="alice@example.com"`

---

## Ideas Considered But Not Selected

1. **Workspace Templates (Full Clone)**: Allow saving a workspace state as a template and recreating it later. Complex implementation, unclear value vs existing git stash/branch features.

2. **Built-in IDE Integration**: Auto-open workspaces in VSCode/IntelliJ. Better as a separate plugin using YAGWT's JSON output.

3. **Workspace Groups/Tags**: Organize workspaces into projects/epics. Adds complexity without proportional benefit - search + notes solve the organization problem more flexibly.

4. **Automatic Stale Branch Detection**: Flag workspaces where the branch has been merged/deleted remotely. Overlaps with existing cleanup policies, and `doctor` can handle broken state.

5. **Workspace Snapshots**: Save workspace state (uncommitted changes, stash, etc.) and restore later. Git stash already handles this well; duplication without clear improvement.

6. **Cross-Workspace File Comparison**: View diffs between files in different workspaces. Better served by IDE diff tools or `git diff branch1..branch2`.

7. **Workspace Statistics Dashboard**: Show metrics like average workspace lifetime, most active workspaces, etc. Fun but low practical value.

8. **Workspace Sharing**: Share workspace configuration with team. Configuration already shareable via `.yagwt/config.toml` in repo.

## Open Questions

1. **GitHub API Authentication**: How should users authenticate for PR/issue fetching?
   - Options: GitHub CLI (reuse existing auth), personal access token, OAuth app
   - Tradeoff: Ease of use vs security vs reliability

2. **Setup Template Security**: Arbitrary script execution is risky. How to make it safe?
   - Options: Require explicit opt-in per repo, sandboxing, template review/signing
   - Tradeoff: Security vs convenience

3. **Notes Storage**: Should notes be in metadata.json or separate files?
   - Options: Metadata (simple, atomic), separate files (Git-trackable, diffable)
   - Tradeoff: Simplicity vs team collaboration

4. **Search Index**: Should search build an index or scan on-demand?
   - Options: On-demand (simple, no maintenance), indexed (faster for 100+ workspaces)
   - Tradeoff: Simplicity vs performance at scale
