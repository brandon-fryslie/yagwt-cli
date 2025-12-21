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
	Tools *ToolsCapability `json:"tools,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
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
				Tools: &ToolsCapability{},
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
