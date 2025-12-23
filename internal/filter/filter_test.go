package filter

import (
	"testing"
	"time"

	"github.com/bmf/yagwt/internal/core"
)

// Test helper to create test workspaces
func makeTestWorkspace(opts map[string]interface{}) core.Workspace {
	ws := core.Workspace{
		ID:   "test-id",
		Name: "test-workspace",
		Path: "/test/path",
		Target: core.Target{
			Type:  "branch",
			Short: "main",
		},
		Flags: core.WorkspaceFlags{},
		Status: core.StatusInfo{
			Dirty: false,
		},
		Activity: core.ActivityInfo{},
	}

	// Apply options
	if name, ok := opts["name"].(string); ok {
		ws.Name = name
	}
	if pinned, ok := opts["pinned"].(bool); ok {
		ws.Flags.Pinned = pinned
	}
	if ephemeral, ok := opts["ephemeral"].(bool); ok {
		ws.Flags.Ephemeral = ephemeral
	}
	if locked, ok := opts["locked"].(bool); ok {
		ws.Flags.Locked = locked
	}
	if broken, ok := opts["broken"].(bool); ok {
		ws.Flags.Broken = broken
	}
	if dirty, ok := opts["dirty"].(bool); ok {
		ws.Status.Dirty = dirty
	}
	if conflicts, ok := opts["conflicts"].(bool); ok {
		ws.Status.Conflicts = conflicts
	}
	if targetType, ok := opts["targetType"].(string); ok {
		ws.Target.Type = targetType
	}
	if detached, ok := opts["detached"].(bool); ok {
		ws.Status.Detached = detached
	}
	if branch, ok := opts["branch"].(string); ok {
		ws.Target.Short = branch
	}
	if lastActivity, ok := opts["lastActivity"].(time.Time); ok {
		ws.Activity.LastGitActivityAt = &lastActivity
	}

	return ws
}

func TestParseFlagFilter(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{"pinned", "flag:pinned", false},
		{"ephemeral", "flag:ephemeral", false},
		{"locked", "flag:locked", false},
		{"broken", "flag:broken", false},
		{"invalid flag", "flag:invalid", true},
		{"missing value", "flag:", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFilter(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFilter(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
			}
		})
	}
}

func TestParseStatusFilter(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{"dirty", "status:dirty", false},
		{"clean", "status:clean", false},
		{"conflicts", "status:conflicts", false},
		{"invalid status", "status:invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFilter(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFilter(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
			}
		})
	}
}

func TestParseTargetFilter(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{"branch", "target:branch", false},
		{"detached", "target:detached", false},
		{"invalid target", "target:tag", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFilter(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFilter(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
			}
		})
	}
}

func TestParseActivityFilter(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{"idle>30d", "activity:idle>30d", false},
		{"idle>7d", "activity:idle>7d", false},
		{"active<1h", "activity:active<1h", false},
		{"active<30m", "activity:active<30m", false},
		{"invalid condition", "activity:invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFilter(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFilter(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
			}
		})
	}
}

func TestParseNameFilter(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{"simple name", "name:feature-123", false},
		{"glob pattern", "name:feature-*", false},
		{"empty name", "name:", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFilter(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFilter(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
			}
		})
	}
}

func TestParseBranchFilter(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{"simple branch", "branch:main", false},
		{"glob pattern", "branch:feature/*", false},
		{"empty branch", "branch:", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFilter(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFilter(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
			}
		})
	}
}

func TestParseAndLogic(t *testing.T) {
	filter, err := ParseFilter("flag:pinned,status:clean")
	if err != nil {
		t.Fatalf("ParseFilter() error = %v", err)
	}

	expr, ok := filter.(*FilterExpr)
	if !ok {
		t.Fatalf("Expected FilterExpr, got %T", filter)
	}

	if expr.Logic != "and" {
		t.Errorf("Expected AND logic, got %q", expr.Logic)
	}

	if len(expr.Filters) != 2 {
		t.Errorf("Expected 2 filters, got %d", len(expr.Filters))
	}
}

func TestParseOrLogic(t *testing.T) {
	filter, err := ParseFilter("flag:pinned|flag:ephemeral")
	if err != nil {
		t.Fatalf("ParseFilter() error = %v", err)
	}

	expr, ok := filter.(*FilterExpr)
	if !ok {
		t.Fatalf("Expected FilterExpr, got %T", filter)
	}

	if expr.Logic != "or" {
		t.Errorf("Expected OR logic, got %q", expr.Logic)
	}

	if len(expr.Filters) != 2 {
		t.Errorf("Expected 2 filters, got %d", len(expr.Filters))
	}
}

func TestParseInvalidFilter(t *testing.T) {
	tests := []string{
		"invalid",
		"unknown:value",
		"flag",
		":",
	}

	for _, expr := range tests {
		t.Run(expr, func(t *testing.T) {
			_, err := ParseFilter(expr)
			if err == nil {
				t.Errorf("ParseFilter(%q) should return error", expr)
			}
		})
	}
}

func TestFlagFilterMatch(t *testing.T) {
	tests := []struct {
		name      string
		filter    string
		ws        core.Workspace
		wantMatch bool
	}{
		{
			name:      "pinned matches",
			filter:    "flag:pinned",
			ws:        makeTestWorkspace(map[string]interface{}{"pinned": true}),
			wantMatch: true,
		},
		{
			name:      "pinned doesn't match",
			filter:    "flag:pinned",
			ws:        makeTestWorkspace(map[string]interface{}{"pinned": false}),
			wantMatch: false,
		},
		{
			name:      "ephemeral matches",
			filter:    "flag:ephemeral",
			ws:        makeTestWorkspace(map[string]interface{}{"ephemeral": true}),
			wantMatch: true,
		},
		{
			name:      "locked matches",
			filter:    "flag:locked",
			ws:        makeTestWorkspace(map[string]interface{}{"locked": true}),
			wantMatch: true,
		},
		{
			name:      "broken matches",
			filter:    "flag:broken",
			ws:        makeTestWorkspace(map[string]interface{}{"broken": true}),
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := ParseFilter(tt.filter)
			if err != nil {
				t.Fatalf("ParseFilter() error = %v", err)
			}

			if got := filter.Match(tt.ws); got != tt.wantMatch {
				t.Errorf("Match() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func TestStatusFilterMatch(t *testing.T) {
	tests := []struct {
		name      string
		filter    string
		ws        core.Workspace
		wantMatch bool
	}{
		{
			name:      "dirty matches",
			filter:    "status:dirty",
			ws:        makeTestWorkspace(map[string]interface{}{"dirty": true}),
			wantMatch: true,
		},
		{
			name:      "clean matches",
			filter:    "status:clean",
			ws:        makeTestWorkspace(map[string]interface{}{"dirty": false}),
			wantMatch: true,
		},
		{
			name:      "conflicts matches",
			filter:    "status:conflicts",
			ws:        makeTestWorkspace(map[string]interface{}{"conflicts": true}),
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := ParseFilter(tt.filter)
			if err != nil {
				t.Fatalf("ParseFilter() error = %v", err)
			}

			if got := filter.Match(tt.ws); got != tt.wantMatch {
				t.Errorf("Match() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func TestTargetFilterMatch(t *testing.T) {
	tests := []struct {
		name      string
		filter    string
		ws        core.Workspace
		wantMatch bool
	}{
		{
			name:      "branch matches",
			filter:    "target:branch",
			ws:        makeTestWorkspace(map[string]interface{}{"targetType": "branch"}),
			wantMatch: true,
		},
		{
			name:      "detached matches commit type",
			filter:    "target:detached",
			ws:        makeTestWorkspace(map[string]interface{}{"targetType": "commit"}),
			wantMatch: true,
		},
		{
			name:      "detached matches detached status",
			filter:    "target:detached",
			ws:        makeTestWorkspace(map[string]interface{}{"detached": true}),
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := ParseFilter(tt.filter)
			if err != nil {
				t.Fatalf("ParseFilter() error = %v", err)
			}

			if got := filter.Match(tt.ws); got != tt.wantMatch {
				t.Errorf("Match() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func TestActivityFilterMatch(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		filter    string
		ws        core.Workspace
		wantMatch bool
	}{
		{
			name:      "idle>30d matches old activity",
			filter:    "activity:idle>30d",
			ws:        makeTestWorkspace(map[string]interface{}{"lastActivity": now.Add(-40 * 24 * time.Hour)}),
			wantMatch: true,
		},
		{
			name:      "idle>30d doesn't match recent activity",
			filter:    "activity:idle>30d",
			ws:        makeTestWorkspace(map[string]interface{}{"lastActivity": now.Add(-10 * 24 * time.Hour)}),
			wantMatch: false,
		},
		{
			name:      "active<1h matches recent activity",
			filter:    "activity:active<1h",
			ws:        makeTestWorkspace(map[string]interface{}{"lastActivity": now.Add(-30 * time.Minute)}),
			wantMatch: true,
		},
		{
			name:      "active<1h doesn't match old activity",
			filter:    "activity:active<1h",
			ws:        makeTestWorkspace(map[string]interface{}{"lastActivity": now.Add(-2 * time.Hour)}),
			wantMatch: false,
		},
		{
			name:      "idle>30d matches nil activity",
			filter:    "activity:idle>30d",
			ws:        makeTestWorkspace(map[string]interface{}{}),
			wantMatch: true,
		},
		{
			name:      "active<1h doesn't match nil activity",
			filter:    "activity:active<1h",
			ws:        makeTestWorkspace(map[string]interface{}{}),
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := ParseFilter(tt.filter)
			if err != nil {
				t.Fatalf("ParseFilter() error = %v", err)
			}

			if got := filter.Match(tt.ws); got != tt.wantMatch {
				t.Errorf("Match() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func TestNameFilterMatch(t *testing.T) {
	tests := []struct {
		name      string
		filter    string
		ws        core.Workspace
		wantMatch bool
	}{
		{
			name:      "exact match",
			filter:    "name:feature-123",
			ws:        makeTestWorkspace(map[string]interface{}{"name": "feature-123"}),
			wantMatch: true,
		},
		{
			name:      "glob match",
			filter:    "name:feature-*",
			ws:        makeTestWorkspace(map[string]interface{}{"name": "feature-123"}),
			wantMatch: true,
		},
		{
			name:      "glob no match",
			filter:    "name:feature-*",
			ws:        makeTestWorkspace(map[string]interface{}{"name": "bugfix-456"}),
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := ParseFilter(tt.filter)
			if err != nil {
				t.Fatalf("ParseFilter() error = %v", err)
			}

			if got := filter.Match(tt.ws); got != tt.wantMatch {
				t.Errorf("Match() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func TestBranchFilterMatch(t *testing.T) {
	tests := []struct {
		name      string
		filter    string
		ws        core.Workspace
		wantMatch bool
	}{
		{
			name:      "exact match",
			filter:    "branch:main",
			ws:        makeTestWorkspace(map[string]interface{}{"branch": "main"}),
			wantMatch: true,
		},
		{
			name:      "glob match",
			filter:    "branch:feature/*",
			ws:        makeTestWorkspace(map[string]interface{}{"branch": "feature/test"}),
			wantMatch: true,
		},
		{
			name:      "no match non-branch",
			filter:    "branch:main",
			ws:        makeTestWorkspace(map[string]interface{}{"targetType": "commit"}),
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := ParseFilter(tt.filter)
			if err != nil {
				t.Fatalf("ParseFilter() error = %v", err)
			}

			if got := filter.Match(tt.ws); got != tt.wantMatch {
				t.Errorf("Match() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func TestFilterExprAndLogic(t *testing.T) {
	filter, err := ParseFilter("flag:pinned,status:clean")
	if err != nil {
		t.Fatalf("ParseFilter() error = %v", err)
	}

	tests := []struct {
		name      string
		ws        core.Workspace
		wantMatch bool
	}{
		{
			name:      "both conditions match",
			ws:        makeTestWorkspace(map[string]interface{}{"pinned": true, "dirty": false}),
			wantMatch: true,
		},
		{
			name:      "only pinned",
			ws:        makeTestWorkspace(map[string]interface{}{"pinned": true, "dirty": true}),
			wantMatch: false,
		},
		{
			name:      "only clean",
			ws:        makeTestWorkspace(map[string]interface{}{"pinned": false, "dirty": false}),
			wantMatch: false,
		},
		{
			name:      "neither",
			ws:        makeTestWorkspace(map[string]interface{}{"pinned": false, "dirty": true}),
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filter.Match(tt.ws); got != tt.wantMatch {
				t.Errorf("Match() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func TestFilterExprOrLogic(t *testing.T) {
	filter, err := ParseFilter("flag:pinned|flag:ephemeral")
	if err != nil {
		t.Fatalf("ParseFilter() error = %v", err)
	}

	tests := []struct {
		name      string
		ws        core.Workspace
		wantMatch bool
	}{
		{
			name:      "both conditions match",
			ws:        makeTestWorkspace(map[string]interface{}{"pinned": true, "ephemeral": true}),
			wantMatch: true,
		},
		{
			name:      "only pinned",
			ws:        makeTestWorkspace(map[string]interface{}{"pinned": true, "ephemeral": false}),
			wantMatch: true,
		},
		{
			name:      "only ephemeral",
			ws:        makeTestWorkspace(map[string]interface{}{"pinned": false, "ephemeral": true}),
			wantMatch: true,
		},
		{
			name:      "neither",
			ws:        makeTestWorkspace(map[string]interface{}{"pinned": false, "ephemeral": false}),
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filter.Match(tt.ws); got != tt.wantMatch {
				t.Errorf("Match() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"30d", 30 * 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"1h", time.Hour, false},
		{"45m", 45 * time.Minute, false},
		{"30s", 30 * time.Second, false},
		{"invalid", 0, true},
		{"", 0, true},
		{"10x", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
