package output

import (
	"github.com/bmf/yagwt/internal/core"
)

// Formatter handles output formatting for different modes
type Formatter interface {
	// Workspace formatting
	FormatWorkspaces(workspaces []core.Workspace) string
	FormatWorkspace(workspace core.Workspace) string
	FormatWorkspacePath(workspace core.Workspace) string

	// Cleanup formatting
	FormatCleanupPlan(plan core.CleanupPlan) string

	// Doctor formatting
	FormatDoctorReport(report core.DoctorReport) string

	// Error formatting
	FormatError(err error) string

	// Version formatting
	FormatVersion(version, commit, date string) string

	// Success messages
	FormatSuccess(message string) string
}

// OutputMode represents different output modes
type OutputMode string

const (
	ModeHuman     OutputMode = "human"
	ModeJSON      OutputMode = "json"
	ModePorcelain OutputMode = "porcelain"
)

// NewFormatter creates a formatter for the specified mode
func NewFormatter(mode OutputMode, quiet bool) Formatter {
	switch mode {
	case ModeJSON:
		return &jsonFormatter{}
	case ModePorcelain:
		return &porcelainFormatter{}
	default:
		return &humanFormatter{quiet: quiet}
	}
}
