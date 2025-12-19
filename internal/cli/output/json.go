package output

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bmf/yagwt/internal/core"
	"github.com/bmf/yagwt/internal/errors"
)

type jsonFormatter struct{}

// JSON schema version for machine consumers
const schemaVersion = 1

type jsonOutput struct {
	SchemaVersion int         `json:"schemaVersion"`
	Data          interface{} `json:"data"`
	Error         interface{} `json:"error,omitempty"`
}

type jsonWorkspace struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Path      string                 `json:"path"`
	IsPrimary bool                   `json:"isPrimary"`
	Target    jsonTarget             `json:"target"`
	Flags     jsonFlags              `json:"flags"`
	Ephemeral *jsonEphemeral         `json:"ephemeral,omitempty"`
	Activity  jsonActivity           `json:"activity"`
	Status    jsonStatus             `json:"status"`
}

type jsonTarget struct {
	Type     string `json:"type"`
	Ref      string `json:"ref"`
	Short    string `json:"short"`
	HeadSHA  string `json:"headSHA"`
	Upstream string `json:"upstream,omitempty"`
}

type jsonFlags struct {
	Pinned    bool `json:"pinned"`
	Ephemeral bool `json:"ephemeral"`
	Locked    bool `json:"locked"`
	Broken    bool `json:"broken"`
}

type jsonEphemeral struct {
	TTLSeconds int    `json:"ttlSeconds"`
	ExpiresAt  string `json:"expiresAt"`
}

type jsonActivity struct {
	LastOpenedAt      *string `json:"lastOpenedAt"`
	LastGitActivityAt *string `json:"lastGitActivityAt"`
}

type jsonStatus struct {
	Dirty     bool   `json:"dirty"`
	Conflicts bool   `json:"conflicts"`
	Ahead     int    `json:"ahead"`
	Behind    int    `json:"behind"`
	Branch    string `json:"branch"`
	Detached  bool   `json:"detached"`
}

type jsonCleanupPlan struct {
	Actions  []jsonRemovalAction `json:"actions"`
	Warnings []jsonWarning       `json:"warnings"`
}

type jsonRemovalAction struct {
	Workspace jsonWorkspace `json:"workspace"`
	Reason    string        `json:"reason"`
	OnDirty   string        `json:"onDirty"`
}

type jsonWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type jsonDoctorReport struct {
	BrokenWorkspaces []jsonWorkspace `json:"brokenWorkspaces"`
	Repairs          []jsonRepair    `json:"repairs"`
	Warnings         []jsonWarning   `json:"warnings"`
}

type jsonRepair struct {
	WorkspaceID string `json:"workspaceId"`
	Issue       string `json:"issue"`
	Fix         string `json:"fix"`
	Applied     bool   `json:"applied"`
}

type jsonVersion struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

type jsonError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
	Hints   []jsonHint             `json:"hints,omitempty"`
}

type jsonHint struct {
	Message string `json:"message"`
	Command string `json:"command,omitempty"`
}

func (f *jsonFormatter) FormatWorkspaces(workspaces []core.Workspace) string {
	jsonWs := make([]jsonWorkspace, len(workspaces))
	for i, ws := range workspaces {
		jsonWs[i] = convertWorkspace(ws)
	}

	output := jsonOutput{
		SchemaVersion: schemaVersion,
		Data:          jsonWs,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"schemaVersion": %d, "error": "failed to marshal JSON: %s"}`, schemaVersion, err)
	}

	return string(data)
}

func (f *jsonFormatter) FormatWorkspace(workspace core.Workspace) string {
	jsonWs := convertWorkspace(workspace)

	output := jsonOutput{
		SchemaVersion: schemaVersion,
		Data:          jsonWs,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"schemaVersion": %d, "error": "failed to marshal JSON: %s"}`, schemaVersion, err)
	}

	return string(data)
}

func (f *jsonFormatter) FormatWorkspacePath(workspace core.Workspace) string {
	// Even in JSON mode, path command returns just the path for scripting
	return workspace.Path
}

func (f *jsonFormatter) FormatCleanupPlan(plan core.CleanupPlan) string {
	jsonPlan := jsonCleanupPlan{
		Actions:  make([]jsonRemovalAction, len(plan.Actions)),
		Warnings: make([]jsonWarning, len(plan.Warnings)),
	}

	for i, action := range plan.Actions {
		jsonPlan.Actions[i] = jsonRemovalAction{
			Workspace: convertWorkspace(action.Workspace),
			Reason:    action.Reason,
			OnDirty:   action.OnDirty,
		}
	}

	for i, warning := range plan.Warnings {
		jsonPlan.Warnings[i] = jsonWarning{
			Code:    warning.Code,
			Message: warning.Message,
		}
	}

	output := jsonOutput{
		SchemaVersion: schemaVersion,
		Data:          jsonPlan,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"schemaVersion": %d, "error": "failed to marshal JSON: %s"}`, schemaVersion, err)
	}

	return string(data)
}

func (f *jsonFormatter) FormatDoctorReport(report core.DoctorReport) string {
	jsonReport := jsonDoctorReport{
		BrokenWorkspaces: make([]jsonWorkspace, len(report.BrokenWorkspaces)),
		Repairs:          make([]jsonRepair, len(report.Repairs)),
		Warnings:         make([]jsonWarning, len(report.Warnings)),
	}

	for i, ws := range report.BrokenWorkspaces {
		jsonReport.BrokenWorkspaces[i] = convertWorkspace(ws)
	}

	for i, repair := range report.Repairs {
		jsonReport.Repairs[i] = jsonRepair{
			WorkspaceID: repair.WorkspaceID,
			Issue:       repair.Issue,
			Fix:         repair.Fix,
			Applied:     repair.Applied,
		}
	}

	for i, warning := range report.Warnings {
		jsonReport.Warnings[i] = jsonWarning{
			Code:    warning.Code,
			Message: warning.Message,
		}
	}

	output := jsonOutput{
		SchemaVersion: schemaVersion,
		Data:          jsonReport,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"schemaVersion": %d, "error": "failed to marshal JSON: %s"}`, schemaVersion, err)
	}

	return string(data)
}

func (f *jsonFormatter) FormatError(err error) string {
	var jsonErr jsonError

	// Check if it's a structured error
	if yerr, ok := err.(*errors.Error); ok {
		jsonErr = jsonError{
			Code:    string(yerr.Code),
			Message: yerr.Message,
			Details: yerr.Details,
		}

		if len(yerr.Hints) > 0 {
			jsonErr.Hints = make([]jsonHint, len(yerr.Hints))
			for i, hint := range yerr.Hints {
				jsonErr.Hints[i] = jsonHint{
					Message: hint.Message,
					Command: hint.Command,
				}
			}
		}
	} else {
		// Generic error
		jsonErr = jsonError{
			Code:    "unknown",
			Message: err.Error(),
		}
	}

	output := jsonOutput{
		SchemaVersion: schemaVersion,
		Error:         jsonErr,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"schemaVersion": %d, "error": {"code": "marshal_error", "message": "%s"}}`, schemaVersion, err)
	}

	return string(data)
}

func (f *jsonFormatter) FormatVersion(version, commit, date string) string {
	ver := jsonVersion{
		Version: version,
		Commit:  commit,
		Date:    date,
	}

	output := jsonOutput{
		SchemaVersion: schemaVersion,
		Data:          ver,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"schemaVersion": %d, "error": "failed to marshal JSON: %s"}`, schemaVersion, err)
	}

	return string(data)
}

func (f *jsonFormatter) FormatSuccess(message string) string {
	output := jsonOutput{
		SchemaVersion: schemaVersion,
		Data: map[string]string{
			"message": message,
		},
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"schemaVersion": %d, "error": "failed to marshal JSON: %s"}`, schemaVersion, err)
	}

	return string(data)
}

// Helper to convert core.Workspace to jsonWorkspace
func convertWorkspace(ws core.Workspace) jsonWorkspace {
	jsonWs := jsonWorkspace{
		ID:        ws.ID,
		Name:      ws.Name,
		Path:      ws.Path,
		IsPrimary: ws.IsPrimary,
		Target: jsonTarget{
			Type:     ws.Target.Type,
			Ref:      ws.Target.Ref,
			Short:    ws.Target.Short,
			HeadSHA:  ws.Target.HeadSHA,
			Upstream: ws.Target.Upstream,
		},
		Flags: jsonFlags{
			Pinned:    ws.Flags.Pinned,
			Ephemeral: ws.Flags.Ephemeral,
			Locked:    ws.Flags.Locked,
			Broken:    ws.Flags.Broken,
		},
		Activity: jsonActivity{
			LastOpenedAt:      formatTimePtr(ws.Activity.LastOpenedAt),
			LastGitActivityAt: formatTimePtr(ws.Activity.LastGitActivityAt),
		},
		Status: jsonStatus{
			Dirty:     ws.Status.Dirty,
			Conflicts: ws.Status.Conflicts,
			Ahead:     ws.Status.Ahead,
			Behind:    ws.Status.Behind,
			Branch:    ws.Status.Branch,
			Detached:  ws.Status.Detached,
		},
	}

	if ws.Ephemeral != nil {
		jsonWs.Ephemeral = &jsonEphemeral{
			TTLSeconds: ws.Ephemeral.TTLSeconds,
			ExpiresAt:  ws.Ephemeral.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return jsonWs
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02T15:04:05Z07:00")
	return &s
}
