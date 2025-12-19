package output

import (
	"fmt"
	"strings"

	"github.com/bmf/yagwt/internal/core"
	"github.com/bmf/yagwt/internal/errors"
)

type porcelainFormatter struct{}

// Porcelain format uses tab-separated values with stable column order
// No headers, no formatting, suitable for scripting

func (f *porcelainFormatter) FormatWorkspaces(workspaces []core.Workspace) string {
	var b strings.Builder

	// Format: id\tname\tpath\tbranch\tstatus\tflags
	for _, ws := range workspaces {
		var flags []string
		if ws.IsPrimary {
			flags = append(flags, "primary")
		}
		if ws.Flags.Pinned {
			flags = append(flags, "pinned")
		}
		if ws.Flags.Locked {
			flags = append(flags, "locked")
		}
		if ws.Flags.Ephemeral {
			flags = append(flags, "ephemeral")
		}
		if ws.Flags.Broken {
			flags = append(flags, "broken")
		}

		status := "clean"
		if ws.Status.Dirty {
			status = "dirty"
		}
		if ws.Status.Conflicts {
			status = "conflicts"
		}

		b.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\n",
			ws.ID,
			ws.Name,
			ws.Path,
			ws.Target.Short,
			status,
			strings.Join(flags, ","),
		))
	}

	return b.String()
}

func (f *porcelainFormatter) FormatWorkspace(workspace core.Workspace) string {
	// Single workspace: same format as list but one line
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

	status := "clean"
	if workspace.Status.Dirty {
		status = "dirty"
	}
	if workspace.Status.Conflicts {
		status = "conflicts"
	}

	return fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\n",
		workspace.ID,
		workspace.Name,
		workspace.Path,
		workspace.Target.Short,
		status,
		strings.Join(flags, ","),
	)
}

func (f *porcelainFormatter) FormatWorkspacePath(workspace core.Workspace) string {
	// Just return the path for scripting use
	return workspace.Path
}

func (f *porcelainFormatter) FormatCleanupPlan(plan core.CleanupPlan) string {
	var b strings.Builder

	// Format: workspace_id\tworkspace_name\treason\tstatus
	for _, action := range plan.Actions {
		status := "clean"
		if action.Workspace.Status.Dirty {
			status = "dirty"
		}

		b.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\n",
			action.Workspace.ID,
			action.Workspace.Name,
			action.Reason,
			status,
		))
	}

	return b.String()
}

func (f *porcelainFormatter) FormatDoctorReport(report core.DoctorReport) string {
	var b strings.Builder

	// Format: type\tworkspace_id\tissue\tfix\tstatus
	for _, repair := range report.Repairs {
		status := "pending"
		if repair.Applied {
			status = "applied"
		}

		b.WriteString(fmt.Sprintf("repair\t%s\t%s\t%s\t%s\n",
			repair.WorkspaceID,
			repair.Issue,
			repair.Fix,
			status,
		))
	}

	for _, warning := range report.Warnings {
		b.WriteString(fmt.Sprintf("warning\t\t%s\t\t\n",
			warning.Message,
		))
	}

	return b.String()
}

func (f *porcelainFormatter) FormatError(err error) string {
	// Format: error_code\tmessage
	if yerr, ok := err.(*errors.Error); ok {
		return fmt.Sprintf("%s\t%s\n", string(yerr.Code), yerr.Message)
	}
	return fmt.Sprintf("unknown\t%s\n", err.Error())
}

func (f *porcelainFormatter) FormatVersion(version, commit, date string) string {
	// Format: version\tcommit\tdate
	return fmt.Sprintf("%s\t%s\t%s\n", version, commit, date)
}

func (f *porcelainFormatter) FormatSuccess(message string) string {
	// Format: success\tmessage
	return fmt.Sprintf("success\t%s\n", message)
}
