package core

import (
	"path/filepath"
	"testing"
)

func TestParseSelector(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType SelectorType
		wantVal  string // Empty means check against input or skip
	}{
		// ID selectors
		{
			name:     "id selector",
			input:    "id:abc123",
			wantType: SelectorID,
			wantVal:  "abc123",
		},
		{
			name:     "id selector with UUID",
			input:    "id:550e8400-e29b-41d4-a716-446655440000",
			wantType: SelectorID,
			wantVal:  "550e8400-e29b-41d4-a716-446655440000",
		},

		// Name selectors
		{
			name:     "name selector",
			input:    "name:my-feature",
			wantType: SelectorName,
			wantVal:  "my-feature",
		},
		{
			name:     "name selector with complex name",
			input:    "name:feature/auth-2024",
			wantType: SelectorName,
			wantVal:  "feature/auth-2024",
		},

		// Path selectors
		{
			name:     "path selector absolute",
			input:    "path:/abs/path/to/workspace",
			wantType: SelectorPath,
			wantVal:  "/abs/path/to/workspace",
		},
		{
			name:     "path selector relative",
			input:    "path:relative/path",
			wantType: SelectorPath,
			// wantVal is absolute, will be checked differently
		},

		// Branch selectors
		{
			name:     "branch selector",
			input:    "branch:feature/x",
			wantType: SelectorBranch,
			wantVal:  "feature/x",
		},
		{
			name:     "branch selector with main",
			input:    "branch:main",
			wantType: SelectorBranch,
			wantVal:  "main",
		},

		// Bare selectors
		{
			name:     "bare selector simple",
			input:    "my-workspace",
			wantType: SelectorBare,
			wantVal:  "my-workspace",
		},
		{
			name:     "bare selector that looks like path",
			input:    "./relative",
			wantType: SelectorBare,
			// wantVal is normalized path
		},
		{
			name:     "bare selector with slash",
			input:    "feature/auth",
			wantType: SelectorBare,
			// wantVal is normalized path
		},
		{
			name:     "bare selector current dir",
			input:    ".",
			wantType: SelectorBare,
			// wantVal is normalized path
		},

		// Edge cases
		{
			name:     "empty colon not a selector",
			input:    ":",
			wantType: SelectorBare,
			wantVal:  ":",
		},
		{
			name:     "unknown prefix",
			input:    "unknown:value",
			wantType: SelectorBare,
			wantVal:  "unknown:value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSelector(tt.input)

			if got.Type != tt.wantType {
				t.Errorf("ParseSelector(%q).Type = %v, want %v", tt.input, got.Type, tt.wantType)
			}

			// For path-like selectors, just check that the value is absolute
			if tt.wantVal == "" {
				if tt.wantType == SelectorPath || (tt.wantType == SelectorBare && (filepath.IsAbs(tt.input) || tt.input == "." || tt.input[0] == '.')) {
					if !filepath.IsAbs(got.Value) {
						t.Errorf("ParseSelector(%q).Value = %q, expected absolute path", tt.input, got.Value)
					}
				}
			} else if got.Value != tt.wantVal {
				t.Errorf("ParseSelector(%q).Value = %q, want %q", tt.input, got.Value, tt.wantVal)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantAbs bool // Should result be absolute?
	}{
		{
			name:    "absolute path unchanged",
			input:   "/absolute/path",
			wantAbs: true,
		},
		{
			name:    "relative path becomes absolute",
			input:   "relative/path",
			wantAbs: true,
		},
		{
			name:    "current dir becomes absolute",
			input:   ".",
			wantAbs: true,
		},
		{
			name:    "parent dir becomes absolute",
			input:   "..",
			wantAbs: true,
		},
		{
			name:    "path with dots cleaned",
			input:   "/path/./to/../workspace",
			wantAbs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizePath(tt.input)

			if tt.wantAbs && !filepath.IsAbs(got) {
				t.Errorf("normalizePath(%q) = %q, expected absolute path", tt.input, got)
			}

			// Check that path is cleaned (no . or ..)
			if filepath.Clean(got) != got {
				t.Errorf("normalizePath(%q) = %q, not cleaned", tt.input, got)
			}
		})
	}
}
