package git

import (
	"testing"
)

// Unit tests for parsing functions

func TestParseWorktreeList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Worktree
	}{
		{
			name: "single worktree with branch",
			input: `worktree /path/to/repo
HEAD abc123def456
branch refs/heads/main

`,
			expected: []Worktree{
				{
					Path:   "/path/to/repo",
					HEAD:   "abc123def456",
					Branch: "main",
				},
			},
		},
		{
			name: "detached worktree",
			input: `worktree /path/to/detached
HEAD def456abc123
detached

`,
			expected: []Worktree{
				{
					Path:   "/path/to/detached",
					HEAD:   "def456abc123",
					Branch: "",
				},
			},
		},
		{
			name: "locked worktree",
			input: `worktree /path/to/locked
HEAD 123abc456def
branch refs/heads/feature
locked reason: testing

`,
			expected: []Worktree{
				{
					Path:   "/path/to/locked",
					HEAD:   "123abc456def",
					Branch: "feature",
					Locked: true,
				},
			},
		},
		{
			name: "prunable worktree",
			input: `worktree /path/to/prunable
HEAD 456def789abc
branch refs/heads/old-branch
prunable gitdir file points to non-existent location

`,
			expected: []Worktree{
				{
					Path:     "/path/to/prunable",
					HEAD:     "456def789abc",
					Branch:   "old-branch",
					Prunable: true,
				},
			},
		},
		{
			name: "multiple worktrees",
			input: `worktree /path/to/main
HEAD abc123
branch refs/heads/main

worktree /path/to/feature-a
HEAD def456
branch refs/heads/feature-a

worktree /path/to/feature-b
HEAD 789ghi
branch refs/heads/feature-b

`,
			expected: []Worktree{
				{Path: "/path/to/main", HEAD: "abc123", Branch: "main"},
				{Path: "/path/to/feature-a", HEAD: "def456", Branch: "feature-a"},
				{Path: "/path/to/feature-b", HEAD: "789ghi", Branch: "feature-b"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseWorktreeList([]byte(tt.input))
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Fatalf("Expected %d worktrees, got %d", len(tt.expected), len(result))
			}

			for i, expected := range tt.expected {
				actual := result[i]
				if actual.Path != expected.Path {
					t.Errorf("Worktree %d: expected path %q, got %q", i, expected.Path, actual.Path)
				}
				if actual.HEAD != expected.HEAD {
					t.Errorf("Worktree %d: expected HEAD %q, got %q", i, expected.HEAD, actual.HEAD)
				}
				if actual.Branch != expected.Branch {
					t.Errorf("Worktree %d: expected branch %q, got %q", i, expected.Branch, actual.Branch)
				}
				if actual.Locked != expected.Locked {
					t.Errorf("Worktree %d: expected locked=%v, got %v", i, expected.Locked, actual.Locked)
				}
				if actual.Prunable != expected.Prunable {
					t.Errorf("Worktree %d: expected prunable=%v, got %v", i, expected.Prunable, actual.Prunable)
				}
			}
		})
	}
}

func TestParseStatusV2(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Status
	}{
		{
			name: "clean branch",
			input: `# branch.oid abc123def456
# branch.head main
# branch.upstream origin/main
# branch.ab +0 -0
`,
			expected: Status{
				Branch:  "main",
				Dirty:   false,
				Ahead:   0,
				Behind:  0,
				Detached: false,
			},
		},
		{
			name: "dirty with modified files",
			input: `# branch.oid abc123
# branch.head feature
1 .M N... 100644 100644 100644 abc def file.txt
? untracked.txt
`,
			expected: Status{
				Branch: "feature",
				Dirty:  true,
			},
		},
		{
			name: "detached HEAD",
			input: `# branch.oid abc123
# branch.head (detached)
`,
			expected: Status{
				Detached: true,
				Dirty:    false,
			},
		},
		{
			name: "ahead and behind",
			input: `# branch.oid abc123
# branch.head feature
# branch.upstream origin/feature
# branch.ab +3 -2
`,
			expected: Status{
				Branch: "feature",
				Ahead:  3,
				Behind: 2,
			},
		},
		{
			name: "conflicts",
			input: `# branch.oid abc123
# branch.head main
u UU N... 100644 100644 100644 100644 abc def ghi conflicted.txt
`,
			expected: Status{
				Branch:    "main",
				Dirty:     true,
				Conflicts: true,
			},
		},
		{
			name: "untracked files only",
			input: `# branch.oid abc123
# branch.head main
? new-file.txt
? another-file.go
`,
			expected: Status{
				Branch: "main",
				Dirty:  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseStatusV2([]byte(tt.input))
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.Branch != tt.expected.Branch {
				t.Errorf("Expected branch %q, got %q", tt.expected.Branch, result.Branch)
			}
			if result.Dirty != tt.expected.Dirty {
				t.Errorf("Expected dirty=%v, got %v", tt.expected.Dirty, result.Dirty)
			}
			if result.Conflicts != tt.expected.Conflicts {
				t.Errorf("Expected conflicts=%v, got %v", tt.expected.Conflicts, result.Conflicts)
			}
			if result.Detached != tt.expected.Detached {
				t.Errorf("Expected detached=%v, got %v", tt.expected.Detached, result.Detached)
			}
			if result.Ahead != tt.expected.Ahead {
				t.Errorf("Expected ahead=%d, got %d", tt.expected.Ahead, result.Ahead)
			}
			if result.Behind != tt.expected.Behind {
				t.Errorf("Expected behind=%d, got %d", tt.expected.Behind, result.Behind)
			}
		})
	}
}
