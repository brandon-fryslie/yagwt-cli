package core

import (
	"path/filepath"
	"strings"
)

// SelectorType indicates how a selector identifies a workspace
type SelectorType int

const (
	SelectorBare SelectorType = iota
	SelectorID
	SelectorName
	SelectorPath
	SelectorBranch
)

// Selector identifies a workspace in commands
type Selector struct {
	Type  SelectorType
	Value string
}

// ParseSelector parses a selector string into a typed Selector
// Syntax:
//   id:<uuid>       - Match by ID
//   name:<alias>    - Match by name
//   path:<path>     - Match by path (absolute or relative)
//   branch:<branch> - Match by branch
//   <bare>          - Try to resolve as: id → name → path → branch
func ParseSelector(s string) Selector {
	// Check for typed selectors with prefix
	if idx := strings.Index(s, ":"); idx > 0 {
		prefix := s[:idx]
		value := s[idx+1:]

		switch prefix {
		case "id":
			return Selector{
				Type:  SelectorID,
				Value: value,
			}
		case "name":
			return Selector{
				Type:  SelectorName,
				Value: value,
			}
		case "path":
			return Selector{
				Type:  SelectorPath,
				Value: normalizePath(value),
			}
		case "branch":
			return Selector{
				Type:  SelectorBranch,
				Value: value,
			}
		}
	}

	// If it looks like a path (contains / or is .), normalize it
	if strings.Contains(s, "/") || s == "." || strings.HasPrefix(s, ".") {
		return Selector{
			Type:  SelectorBare,
			Value: normalizePath(s),
		}
	}

	// Default to bare selector
	return Selector{
		Type:  SelectorBare,
		Value: s,
	}
}

// normalizePath converts a path to absolute form
func normalizePath(path string) string {
	// Clean the path first
	path = filepath.Clean(path)

	// Convert to absolute if not already
	if !filepath.IsAbs(path) {
		if absPath, err := filepath.Abs(path); err == nil {
			path = absPath
		}
	}

	return path
}
