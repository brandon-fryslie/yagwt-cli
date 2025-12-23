package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// JSON-RPC types
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// MCP types
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResult struct {
	ProtocolVersion string            `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo        `json:"serverInfo"`
}

type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type ResourcesCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
	Subscribe   bool `json:"subscribe,omitempty"`
}

type Tool struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	InputSchema JSONSchema `json:"inputSchema"`
}

type JSONSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

type CallToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type CallToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Prompts
type Prompt struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Arguments   []PromptArg `json:"arguments,omitempty"`
}

type PromptArg struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Required    bool       `json:"required"`
}

type PromptsListResult struct {
	Prompts []Prompt `json:"prompts"`
}

type GetPromptResult struct {
	Description string   `json:"description"`
	Messages    []Message `json:"messages"`
}

type Message struct {
	Role string `json:"role"`
	Text Text   `json:"text"`
}

type Text struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Resources
type Resource struct {
	URI         string        `json:"uri"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	MimeType    string        `json:"mimeType,omitempty"`
}

type ResourcesListResult struct {
	Resources []Resource `json:"resources"`
}

type ReadResourceResult struct {
	Contents []ResourceContent `json:"contents"`
}

type ResourceContent struct {
	URI     string `json:"uri"`
	MimeType string `json:"mimeType"`
	Text    string `json:"text,omitempty"`
}

// Server
type Server struct {
	repoRoot string
}

func NewServer() *Server {
	// Find repo root from current directory
	root, err := findRepoRoot()
	if err != nil {
		root = "."
	}
	return &Server{repoRoot: root}
}

func findRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (s *Server) Run() {
	scanner := bufio.NewScanner(os.Stdin)
	// Increase buffer size for large messages
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.sendError(nil, -32700, "Parse error")
			continue
		}

		s.handleRequest(&req)
	}
}

func (s *Server) handleRequest(req *Request) {
	switch req.Method {
	case "initialize":
		s.sendResult(req.ID, InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: ServerCapabilities{
				Tools:     &ToolsCapability{},
				Prompts:   &PromptsCapability{},
				Resources: &ResourcesCapability{},
			},
			ServerInfo: ServerInfo{
				Name:    "yagwt-mcp",
				Version: "0.1.0",
			},
		})

	case "notifications/initialized":
		// No response needed for notifications

	case "tools/list":
		s.sendResult(req.ID, ToolsListResult{
			Tools: s.getTools(),
		})

	case "tools/call":
		var params CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendError(req.ID, -32602, "Invalid params")
			return
		}
		result := s.callTool(params.Name, params.Arguments)
		s.sendResult(req.ID, result)

	case "prompts/list":
		s.sendResult(req.ID, PromptsListResult{
			Prompts: s.getPrompts(),
		})

	case "prompts/get":
		var params struct {
			Name string `json:"name"`
			Arguments map[string]string `json:"arguments,omitempty"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendError(req.ID, -32602, "Invalid params")
			return
		}
		result := s.getPrompt(params.Name, params.Arguments)
		s.sendResult(req.ID, result)

	case "resources/list":
		s.sendResult(req.ID, ResourcesListResult{
			Resources: s.getResources(),
		})

	case "resources/read":
		var params struct {
			URI string `json:"uri"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendError(req.ID, -32602, "Invalid params")
			return
		}
		result := s.readResource(params.URI)
		s.sendResult(req.ID, result)

	default:
		s.sendError(req.ID, -32601, "Method not found: "+req.Method)
	}
}

func (s *Server) getTools() []Tool {
	return []Tool{
		{
			Name:        "worktree_create",
			Description: "Create a new git worktree for isolated parallel work. Returns the path to the new worktree.",
			InputSchema: JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"branch": {
						Type:        "string",
						Description: "Name for the new branch to create",
					},
					"base": {
						Type:        "string",
						Description: "Base branch to create from (default: current branch)",
					},
					"name": {
						Type:        "string",
						Description: "Name for the worktree (default: derived from branch)",
					},
				},
				Required: []string{"branch"},
			},
		},
		{
			Name:        "worktree_list",
			Description: "List all worktrees with their paths and branches",
			InputSchema: JSONSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name:        "worktree_remove",
			Description: "Remove a worktree and optionally its branch",
			InputSchema: JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"name": {
						Type:        "string",
						Description: "Name of the worktree to remove",
					},
					"delete_branch": {
						Type:        "string",
						Description: "Also delete the branch (true/false, default: false)",
					},
				},
				Required: []string{"name"},
			},
		},
		{
			Name:        "worktree_path",
			Description: "Get the absolute path to a worktree by name",
			InputSchema: JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"name": {
						Type:        "string",
						Description: "Name of the worktree",
					},
				},
				Required: []string{"name"},
			},
		},
	}
}

func (s *Server) getPrompts() []Prompt {
	return []Prompt{
		{
			Name:        "parallel-work",
			Description: "Set up parallel work by creating isolated worktrees for multiple features/tasks",
			Arguments: []PromptArg{
				{
					Name:        "tasks",
					Description: "List of tasks to work on in parallel (one per line, format: 'task_name: brief description')",
					Required:    true,
				},
				{
					Name:        "base",
					Description: "Base branch to create all worktrees from",
					Required:    false,
				},
			},
		},
		{
			Name:        "feature-work",
			Description: "Create a worktree for working on a new feature",
			Arguments: []PromptArg{
				{
					Name:        "feature",
					Description: "Name of the feature branch",
					Required:    true,
				},
				{
					Name:        "description",
					Description: "Brief description of what needs to be implemented",
					Required:    true,
				},
				{
					Name:        "dependencies",
					Description: "Any dependencies or prerequisites",
					Required:    false,
				},
			},
		},
		{
			Name:        "bug-fix",
			Description: "Create a worktree for fixing a bug",
			Arguments: []PromptArg{
				{
					Name:        "bug_id",
					Description: "Bug ID or ticket number",
					Required:    true,
				},
				{
					Name:        "description",
					Description: "Description of the bug and how to reproduce",
					Required:    true,
				},
				{
					Name:        "affected_files",
					Description: "Files known to be affected by the bug",
					Required:    false,
				},
			},
		},
	}
}

func (s *Server) getPrompt(name string, args map[string]string) GetPromptResult {
	var prompt string

	switch name {
	case "parallel-work":
		tasks := args["tasks"]
		base := args["base"]
		if base == "" {
			base = "current branch"
		}
		prompt = fmt.Sprintf(`I need to work on multiple tasks in parallel. Here's what needs to be done:

Tasks:
%s

Please:
1. Create a worktree for each task using the worktree_create tool
2. For each worktree, start a subagent to work on that task
3. Each subagent should work in isolation in its worktree
4. Report back when all tasks are complete or if any issues arise

Base branch: %s

Instructions for each subagent:
- Work independently in your assigned worktree
- Commit your work when done
- Report completion status back to the main thread
- Don't interfere with other parallel worktrees`, tasks, base)

	case "feature-work":
		feature := args["feature"]
		description := args["description"]
		dependencies := args["dependencies"]
		prompt = fmt.Sprintf(`Working on feature: %s

Description: %s

Dependencies: %s

Please:
1. Create a new worktree for this feature (use the feature name as the branch)
2. Implement the feature according to the description
3. Add tests if applicable
4. Commit the changes
5. Provide a summary of what was implemented

Make sure to work in isolation using the worktree management tools.`, feature, description, dependencies)

	case "bug-fix":
		bugID := args["bug_id"]
		description := args["description"]
		affectedFiles := args["affected_files"]
		prompt = fmt.Sprintf(`Fixing bug: %s

Description: %s

Affected files: %s

Please:
1. Create a worktree for this bug fix (use bugfix-%s as the branch name)
2. Reproduce the issue if possible
3. Fix the bug
4. Test the fix
5. Commit the changes with a clear commit message referencing the bug ID
6. Provide a summary of the fix

Work in isolation using the worktree tools to avoid affecting other work.`, bugID, description, affectedFiles, bugID)

	default:
		prompt = "Unknown prompt: " + name
	}

	return GetPromptResult{
		Description: "Worktree management prompt",
		Messages: []Message{
			{
				Role: "user",
				Text: Text{
					Type: "text",
					Text: prompt,
				},
			},
		},
	}
}

func (s *Server) getResources() []Resource {
	return []Resource{
		{
			URI:         "worktree://list",
			Name:        "Worktree List",
			Description: "Current list of all worktrees and their status",
			MimeType:    "text/plain",
		},
		{
			URI:         "worktree://branches",
			Name:        "Available Branches",
			Description: "List of all available branches in the repository",
			MimeType:    "text/plain",
		},
		{
			URI:         "worktree://status",
			Name:        "Repository Status",
			Description: "Current git status and worktree states",
			MimeType:    "text/plain",
		},
	}
}

func (s *Server) readResource(uri string) ReadResourceResult {
	var content ResourceContent

	switch uri {
	case "worktree://list":
		cmd := exec.Command("yagwt", "ls")
		cmd.Dir = s.repoRoot
		output, err := cmd.Output()
		if err != nil {
			output = []byte("Error listing worktrees: " + err.Error())
		}
		content = ResourceContent{
			URI:      uri,
			MimeType: "text/plain",
			Text:     string(output),
		}

	case "worktree://branches":
		cmd := exec.Command("git", "branch", "-a", "--format=%(refname:short)")
		cmd.Dir = s.repoRoot
		output, err := cmd.Output()
		if err != nil {
			output = []byte("Error listing branches: " + err.Error())
		}
		content = ResourceContent{
			URI:      uri,
			MimeType: "text/plain",
			Text:     string(output),
		}

	case "worktree://status":
		// Get git status of main repo
		cmd := exec.Command("git", "status", "--porcelain")
		cmd.Dir = s.repoRoot
		output, err := cmd.Output()
		status := string(output)
		if err != nil {
			status = "Error getting status: " + err.Error()
		}

		// Get current branch
		cmd2 := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
		cmd2.Dir = s.repoRoot
		output2, err2 := cmd2.Output()
		currentBranch := "unknown"
		if err2 == nil {
			currentBranch = strings.TrimSpace(string(output2))
		}

		content = ResourceContent{
			URI:      uri,
			MimeType: "text/plain",
			Text:     fmt.Sprintf("Current branch: %s\n\nGit status:\n%s", currentBranch, status),
		}

	default:
		content = ResourceContent{
			URI:      uri,
			MimeType: "text/plain",
			Text:     "Unknown resource: " + uri,
		}
	}

	return ReadResourceResult{
		Contents: []ResourceContent{content},
	}
}

func (s *Server) callTool(name string, args map[string]any) CallToolResult {
	switch name {
	case "worktree_create":
		return s.worktreeCreate(args)
	case "worktree_list":
		return s.worktreeList()
	case "worktree_remove":
		return s.worktreeRemove(args)
	case "worktree_path":
		return s.worktreePath(args)
	default:
		return CallToolResult{
			Content: []Content{{Type: "text", Text: "Unknown tool: " + name}},
			IsError: true,
		}
	}
}

func (s *Server) worktreeCreate(args map[string]any) CallToolResult {
	branch, _ := args["branch"].(string)
	base, _ := args["base"].(string)
	name, _ := args["name"].(string)

	if branch == "" {
		return errorResult("branch is required")
	}

	// Build yagwt command
	cmdArgs := []string{"new", branch, "--new-branch"}
	if base != "" {
		cmdArgs = append(cmdArgs, "-b", base)
	}
	if name != "" {
		cmdArgs = append(cmdArgs, "--name", name)
	}
	cmdArgs = append(cmdArgs, "--ephemeral")

	cmd := exec.Command("yagwt", cmdArgs...)
	cmd.Dir = s.repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to create worktree: %s\n%s", err, output))
	}

	// Extract path from output
	path := extractPath(string(output))
	if path == "" {
		path = string(output)
	}

	return CallToolResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Created worktree at: %s", path),
		}},
	}
}

func (s *Server) worktreeList() CallToolResult {
	cmd := exec.Command("yagwt", "ls", "--json")
	cmd.Dir = s.repoRoot
	output, err := cmd.Output()
	if err != nil {
		// Fall back to non-json
		cmd = exec.Command("yagwt", "ls")
		cmd.Dir = s.repoRoot
		output, _ = cmd.Output()
	}

	return CallToolResult{
		Content: []Content{{Type: "text", Text: string(output)}},
	}
}

func (s *Server) worktreeRemove(args map[string]any) CallToolResult {
	name, _ := args["name"].(string)
	deleteBranch, _ := args["delete_branch"].(string)

	if name == "" {
		return errorResult("name is required")
	}

	cmdArgs := []string{"rm", name}
	if deleteBranch == "true" {
		cmdArgs = append(cmdArgs, "--delete-branch")
	}

	cmd := exec.Command("yagwt", cmdArgs...)
	cmd.Dir = s.repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to remove worktree: %s\n%s", err, output))
	}

	return CallToolResult{
		Content: []Content{{Type: "text", Text: "Worktree removed: " + name}},
	}
}

func (s *Server) worktreePath(args map[string]any) CallToolResult {
	name, _ := args["name"].(string)

	if name == "" {
		return errorResult("name is required")
	}

	cmd := exec.Command("yagwt", "path", name)
	cmd.Dir = s.repoRoot
	output, err := cmd.Output()
	if err != nil {
		return errorResult(fmt.Sprintf("Worktree not found: %s", name))
	}

	return CallToolResult{
		Content: []Content{{Type: "text", Text: strings.TrimSpace(string(output))}},
	}
}

func extractPath(output string) string {
	// Look for "Path:" line in output
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Path:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Path:"))
		}
	}
	// Also try "Created worktree:" pattern
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "Created worktree:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

func errorResult(msg string) CallToolResult {
	return CallToolResult{
		Content: []Content{{Type: "text", Text: msg}},
		IsError: true,
	}
}

func (s *Server) sendResult(id any, result any) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.send(resp)
}

func (s *Server) sendError(id any, code int, message string) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &Error{Code: code, Message: message},
	}
	s.send(resp)
}

func (s *Server) send(v any) {
	data, _ := json.Marshal(v)
	fmt.Println(string(data))
}

func main() {
	server := NewServer()
	server.Run()
}
