package filter

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bmf/yagwt/internal/core"
	"github.com/bmf/yagwt/internal/errors"
)

// Filter represents a filter that can match workspaces
type Filter interface {
	Match(ws core.Workspace) bool
}

// FilterExpr represents a composite filter expression
type FilterExpr struct {
	Filters []Filter
	Logic   string // "and" or "or"
}

// Match evaluates the filter expression against a workspace
func (f *FilterExpr) Match(ws core.Workspace) bool {
	if len(f.Filters) == 0 {
		return true
	}

	if f.Logic == "or" {
		// OR logic: at least one filter must match
		for _, filter := range f.Filters {
			if filter.Match(ws) {
				return true
			}
		}
		return false
	}

	// AND logic (default): all filters must match
	for _, filter := range f.Filters {
		if !filter.Match(ws) {
			return false
		}
	}
	return true
}

// FlagFilter filters by workspace flags
type FlagFilter struct {
	Flag string
}

func (f *FlagFilter) Match(ws core.Workspace) bool {
	switch f.Flag {
	case "pinned":
		return ws.Flags.Pinned
	case "ephemeral":
		return ws.Flags.Ephemeral
	case "locked":
		return ws.Flags.Locked
	case "broken":
		return ws.Flags.Broken
	default:
		return false
	}
}

// StatusFilter filters by workspace status
type StatusFilter struct {
	Status string
}

func (f *StatusFilter) Match(ws core.Workspace) bool {
	switch f.Status {
	case "dirty":
		return ws.Status.Dirty
	case "clean":
		return !ws.Status.Dirty
	case "conflicts":
		return ws.Status.Conflicts
	default:
		return false
	}
}

// TargetFilter filters by target type
type TargetFilter struct {
	Type string
}

func (f *TargetFilter) Match(ws core.Workspace) bool {
	switch f.Type {
	case "branch":
		return ws.Target.Type == "branch"
	case "detached":
		return ws.Target.Type == "commit" || ws.Status.Detached
	default:
		return false
	}
}

// ActivityFilter filters by activity conditions
type ActivityFilter struct {
	Condition string // e.g., "idle>30d", "active<1h"
}

func (f *ActivityFilter) Match(ws core.Workspace) bool {
	// Parse condition: "idle>30d" or "active<1h"
	if strings.HasPrefix(f.Condition, "idle>") {
		durationStr := strings.TrimPrefix(f.Condition, "idle>")
		duration, err := parseDuration(durationStr)
		if err != nil {
			return false
		}

		// Check if workspace is idle (no git activity) for longer than duration
		if ws.Activity.LastGitActivityAt == nil {
			// No activity recorded, consider very idle
			return true
		}

		idleTime := time.Since(*ws.Activity.LastGitActivityAt)
		return idleTime > duration
	}

	if strings.HasPrefix(f.Condition, "active<") {
		durationStr := strings.TrimPrefix(f.Condition, "active<")
		duration, err := parseDuration(durationStr)
		if err != nil {
			return false
		}

		// Check if workspace was active within duration
		if ws.Activity.LastGitActivityAt == nil {
			// No activity recorded
			return false
		}

		idleTime := time.Since(*ws.Activity.LastGitActivityAt)
		return idleTime < duration
	}

	return false
}

// NameFilter filters by workspace name pattern
type NameFilter struct {
	Pattern string
}

func (f *NameFilter) Match(ws core.Workspace) bool {
	matched, err := filepath.Match(f.Pattern, ws.Name)
	if err != nil {
		return false
	}
	return matched
}

// BranchFilter filters by branch pattern
type BranchFilter struct {
	Pattern string
}

func (f *BranchFilter) Match(ws core.Workspace) bool {
	if ws.Target.Type != "branch" {
		return false
	}

	// Match against short branch name
	matched, err := filepath.Match(f.Pattern, ws.Target.Short)
	if err != nil {
		return false
	}
	return matched
}

// ParseFilter parses a filter expression string
func ParseFilter(expr string) (Filter, error) {
	if expr == "" {
		return &FilterExpr{}, nil
	}

	// Check for OR logic (pipe separator)
	if strings.Contains(expr, "|") {
		parts := strings.Split(expr, "|")
		var filters []Filter
		for _, part := range parts {
			f, err := parseSingleFilter(strings.TrimSpace(part))
			if err != nil {
				return nil, err
			}
			filters = append(filters, f)
		}
		return &FilterExpr{
			Filters: filters,
			Logic:   "or",
		}, nil
	}

	// Check for AND logic (comma separator)
	if strings.Contains(expr, ",") {
		parts := strings.Split(expr, ",")
		var filters []Filter
		for _, part := range parts {
			f, err := parseSingleFilter(strings.TrimSpace(part))
			if err != nil {
				return nil, err
			}
			filters = append(filters, f)
		}
		return &FilterExpr{
			Filters: filters,
			Logic:   "and",
		}, nil
	}

	// Single filter
	return parseSingleFilter(expr)
}

// parseSingleFilter parses a single filter (e.g., "flag:pinned")
func parseSingleFilter(expr string) (Filter, error) {
	parts := strings.SplitN(expr, ":", 2)
	if len(parts) != 2 {
		return nil, errors.NewError(errors.ErrConfig, "invalid filter syntax").
			WithDetail("filter", expr).
			WithHint("Use format 'type:value' (e.g., flag:pinned)", "")
	}

	filterType := parts[0]
	filterValue := parts[1]

	switch filterType {
	case "flag":
		validFlags := map[string]bool{
			"pinned":    true,
			"ephemeral": true,
			"locked":    true,
			"broken":    true,
		}
		if !validFlags[filterValue] {
			return nil, errors.NewError(errors.ErrConfig, "invalid flag filter value").
				WithDetail("value", filterValue).
				WithHint("Valid flags: pinned, ephemeral, locked, broken", "")
		}
		return &FlagFilter{Flag: filterValue}, nil

	case "status":
		validStatuses := map[string]bool{
			"dirty":     true,
			"clean":     true,
			"conflicts": true,
		}
		if !validStatuses[filterValue] {
			return nil, errors.NewError(errors.ErrConfig, "invalid status filter value").
				WithDetail("value", filterValue).
				WithHint("Valid statuses: dirty, clean, conflicts", "")
		}
		return &StatusFilter{Status: filterValue}, nil

	case "target":
		validTargets := map[string]bool{
			"branch":   true,
			"detached": true,
		}
		if !validTargets[filterValue] {
			return nil, errors.NewError(errors.ErrConfig, "invalid target filter value").
				WithDetail("value", filterValue).
				WithHint("Valid targets: branch, detached", "")
		}
		return &TargetFilter{Type: filterValue}, nil

	case "activity":
		// Validate activity condition format
		if !strings.HasPrefix(filterValue, "idle>") && !strings.HasPrefix(filterValue, "active<") {
			return nil, errors.NewError(errors.ErrConfig, "invalid activity filter condition").
				WithDetail("value", filterValue).
				WithHint("Use format 'idle>30d' or 'active<1h'", "")
		}
		return &ActivityFilter{Condition: filterValue}, nil

	case "name":
		if filterValue == "" {
			return nil, errors.NewError(errors.ErrConfig, "name filter cannot be empty").
				WithHint("Use a name pattern (e.g., name:feature-*)", "")
		}
		return &NameFilter{Pattern: filterValue}, nil

	case "branch":
		if filterValue == "" {
			return nil, errors.NewError(errors.ErrConfig, "branch filter cannot be empty").
				WithHint("Use a branch pattern (e.g., branch:main)", "")
		}
		return &BranchFilter{Pattern: filterValue}, nil

	default:
		return nil, errors.NewError(errors.ErrConfig, "unknown filter type").
			WithDetail("type", filterType).
			WithHint("Valid types: flag, status, target, activity, name, branch", "")
	}
}

// parseDuration parses duration strings like "30d", "1h", "45m"
func parseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	// Extract number and unit
	var numStr string
	var unit string

	for i, c := range s {
		if c >= '0' && c <= '9' {
			numStr += string(c)
		} else {
			unit = s[i:]
			break
		}
	}

	if numStr == "" {
		return 0, fmt.Errorf("invalid duration: no number")
	}

	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("invalid duration number: %v", err)
	}

	switch unit {
	case "d":
		return time.Duration(num) * 24 * time.Hour, nil
	case "h":
		return time.Duration(num) * time.Hour, nil
	case "m":
		return time.Duration(num) * time.Minute, nil
	case "s":
		return time.Duration(num) * time.Second, nil
	default:
		return 0, fmt.Errorf("invalid duration unit: %s (use d, h, m, or s)", unit)
	}
}
