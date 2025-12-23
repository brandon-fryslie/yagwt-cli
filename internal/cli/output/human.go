package output

import (
	"fmt"
	"strings"
	"time"

	"github.com/bmf/yagwt/internal/core"
	"github.com/bmf/yagwt/internal/errors"
)

type humanFormatter struct {
	quiet bool
}

func (f *humanFormatter) FormatWorkspaces(workspaces []core.Workspace) string {
	if len(workspaces) == 0 {
		return "No workspaces found."
	}

	var b strings.Builder

	// Header
	b.WriteString(formatTableHeader([]string{"NAME", "BRANCH/COMMIT", "PATH", "STATUS"}))
	b.WriteString("\n")

	// Rows
	for _, ws := range workspaces {
		name := ws.Name
		if ws.IsPrimary {
			name += " (primary)"
		}
		if ws.Flags.Pinned {
			name += " [P]"
		}
		if ws.Flags.Locked {
			name += " [L]"
		}
		if ws.Flags.Ephemeral {
			name += " [E]"
		}
		if ws.Flags.Broken {
			name += " [BROKEN]"
		}

		target := ws.Target.Short
		if ws.Status.Detached {
			target = "detached @ " + target
		}

		status := formatStatus(ws.Status)

		b.WriteString(fmt.Sprintf("%-30s %-30s %-40s %s\n",
			truncate(name, 30),
			truncate(target, 30),
			truncate(ws.Path, 40),
			status,
		))
	}

	// Legend
	if !f.quiet {
		b.WriteString("\nLegend: [P]=Pinned [L]=Locked [E]=Ephemeral\n")
	}

	return b.String()
}

func (f *humanFormatter) FormatWorkspace(workspace core.Workspace) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Workspace: %s\n", workspace.Name))
	b.WriteString(fmt.Sprintf("  ID:         %s\n", workspace.ID))
	b.WriteString(fmt.Sprintf("  Path:       %s\n", workspace.Path))
	b.WriteString(fmt.Sprintf("  Branch:     %s\n", workspace.Target.Short))
	if workspace.Target.Upstream != "" {
		b.WriteString(fmt.Sprintf("  Upstream:   %s\n", workspace.Target.Upstream))
	}
	b.WriteString(fmt.Sprintf("  HEAD:       %s\n", workspace.Target.HeadSHA[:7]))
	b.WriteString(fmt.Sprintf("  Status:     %s\n", formatStatus(workspace.Status)))

	// Flags
	var flags []string
	if workspace.IsPrimary {
		flags = append(flags, "primary")
	}
	if workspace.Flags.Pinned {
		flags = append(flags, "pinned")
	}
	if workspace.Flags.Locked {
		flags = append(flags, "locked")
	}
	if workspace.Flags.Ephemeral {
		flags = append(flags, "ephemeral")
	}
	if workspace.Flags.Broken {
		flags = append(flags, "broken")
	}
	if len(flags) > 0 {
		b.WriteString(fmt.Sprintf("  Flags:      %s\n", strings.Join(flags, ", ")))
	}

	// Ephemeral info
	if workspace.Ephemeral != nil {
		b.WriteString(fmt.Sprintf("  Expires:    %s (%s)\n",
			workspace.Ephemeral.ExpiresAt.Format("2006-01-02 15:04"),
			formatDuration(time.Until(workspace.Ephemeral.ExpiresAt)),
		))
	}

	// Activity
	if workspace.Activity.LastOpenedAt != nil {
		b.WriteString(fmt.Sprintf("  Last Used:  %s (%s ago)\n",
			workspace.Activity.LastOpenedAt.Format("2006-01-02 15:04"),
			formatDuration(time.Since(*workspace.Activity.LastOpenedAt)),
		))
	}

	return b.String()
}

func (f *humanFormatter) FormatWorkspacePath(workspace core.Workspace) string {
	// Just return the path for scripting use
	return workspace.Path
}

func (f *humanFormatter) FormatCleanupPlan(plan core.CleanupPlan) string {
	var b strings.Builder

	if len(plan.Actions) == 0 {
		b.WriteString("No workspaces to clean up.\n")
	} else {
		b.WriteString(fmt.Sprintf("Cleanup plan: %d workspace(s) to remove\n\n", len(plan.Actions)))

		// Header
		b.WriteString(formatTableHeader([]string{"NAME", "REASON", "STATUS"}))
		b.WriteString("\n")

		// Rows
		for _, action := range plan.Actions {
			status := "clean"
			if action.Workspace.Status.Dirty {
				status = fmt.Sprintf("dirty (will %s)", action.OnDirty)
			}

			b.WriteString(fmt.Sprintf("%-30s %-40s %s\n",
				truncate(action.Workspace.Name, 30),
				truncate(action.Reason, 40),
				status,
			))
		}
	}

	// Warnings
	if len(plan.Warnings) > 0 {
		b.WriteString("\nWarnings:\n")
		for _, warning := range plan.Warnings {
			b.WriteString(fmt.Sprintf("  - %s\n", warning.Message))
		}
	}

	if len(plan.Actions) > 0 && !f.quiet {
		b.WriteString("\nRun with --apply to execute this plan.\n")
	}

	return b.String()
}

func (f *humanFormatter) FormatDoctorReport(report core.DoctorReport) string {
	var b strings.Builder

	if len(report.BrokenWorkspaces) == 0 && len(report.Repairs) == 0 {
		b.WriteString("All workspaces are healthy.\n")
		return b.String()
	}

	// Broken workspaces
	if len(report.BrokenWorkspaces) > 0 {
		b.WriteString(fmt.Sprintf("Found %d broken workspace(s):\n\n", len(report.BrokenWorkspaces)))
		for _, ws := range report.BrokenWorkspaces {
			b.WriteString(fmt.Sprintf("  - %s (ID: %s)\n", ws.Name, ws.ID))
			b.WriteString(fmt.Sprintf("    Path: %s\n", ws.Path))
		}
		b.WriteString("\n")
	}

	// Repairs
	if len(report.Repairs) > 0 {
		b.WriteString(fmt.Sprintf("Repairs: %d issue(s)\n\n", len(report.Repairs)))
		for _, repair := range report.Repairs {
			status := "pending"
			if repair.Applied {
				status = "applied"
			}

			b.WriteString(fmt.Sprintf("  [%s] %s\n", status, repair.Issue))
			b.WriteString(fmt.Sprintf("    Fix: %s\n", repair.Fix))
			if repair.WorkspaceID != "" {
				b.WriteString(fmt.Sprintf("    Workspace ID: %s\n", repair.WorkspaceID))
			}
		}
	}

	// Warnings
	if len(report.Warnings) > 0 {
		b.WriteString("\nWarnings:\n")
		for _, warning := range report.Warnings {
			b.WriteString(fmt.Sprintf("  - %s\n", warning.Message))
		}
	}

	return b.String()
}

func (f *humanFormatter) FormatError(err error) string {
	var b strings.Builder

	// Check if it's a structured error
	if yerr, ok := err.(*errors.Error); ok {
		b.WriteString(fmt.Sprintf("Error: %s\n", yerr.Message))

		// Show details if present
		if len(yerr.Details) > 0 && !f.quiet {
			b.WriteString("\nDetails:\n")
			for key, value := range yerr.Details {
				b.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
			}
		}

		// Show hints if present
		if len(yerr.Hints) > 0 {
			b.WriteString("\nHints:\n")
			for _, hint := range yerr.Hints {
				b.WriteString(fmt.Sprintf("  - %s\n", hint.Message))
				if hint.Command != "" {
					b.WriteString(fmt.Sprintf("    Try: %s\n", hint.Command))
				}
			}
		}
	} else {
		// Generic error
		b.WriteString(fmt.Sprintf("Error: %s\n", err.Error()))
	}

	return b.String()
}

func (f *humanFormatter) FormatVersion(version, commit, date string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("yagwt version %s\n", version))
	if commit != "" && commit != "unknown" {
		b.WriteString(fmt.Sprintf("Commit: %s\n", commit))
	}
	if date != "" && date != "unknown" {
		b.WriteString(fmt.Sprintf("Built: %s\n", date))
	}
	return b.String()
}

func (f *humanFormatter) FormatSuccess(message string) string {
	// Don't output success messages when quiet flag is set
	if f.quiet {
		return ""
	}
	return message + "\n"
}

// Helper functions

func formatTableHeader(headers []string) string {
	return strings.Join(headers, " | ")
}

func formatStatus(status core.StatusInfo) string {
	var parts []string

	if status.Dirty {
		parts = append(parts, "dirty")
	}
	if status.Conflicts {
		parts = append(parts, "conflicts")
	}
	if status.Ahead > 0 {
		parts = append(parts, fmt.Sprintf("ahead %d", status.Ahead))
	}
	if status.Behind > 0 {
		parts = append(parts, fmt.Sprintf("behind %d", status.Behind))
	}

	if len(parts) == 0 {
		return "clean"
	}

	return strings.Join(parts, ", ")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = -d
	}

	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	}
	if hours > 0 {
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	if minutes > 0 {
		if minutes == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	}
	return "just now"
}
