package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	mcpRemove bool
)

// MCPConfig represents the root structure of .mcp.json
type MCPConfig struct {
	McpServers map[string]MCPServerConfig `json:"mcpServers"`
}

// MCPServerConfig represents an MCP server configuration
type MCPServerConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Env     []string `json:"env,omitempty"`
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Manage MCP configuration in .mcp.json",
	Long: `Manage the Model Context Protocol (MCP) configuration for yagwt.

By default, adds yagwt MCP server configuration to .mcp.json in the current directory.
Use --rm to remove the yagwt configuration.

The MCP configuration allows yagwt to act as an MCP server, providing
worktree management capabilities to MCP-compatible clients.

Examples:
  yagwt mcp              # Add yagwt MCP configuration
  yagwt mcp --rm         # Remove yagwt MCP configuration`,
	Run: runMCP,
}

func init() {
	mcpCmd.Flags().BoolVar(&mcpRemove, "rm", false, "Remove yagwt MCP configuration")
}

// getYagwtMCPConfig returns the yagwt MCP server configuration
func getYagwtMCPConfig() MCPServerConfig {
	// Default to assuming yagwt is in PATH
	yagwtPath := "yagwt"

	// Try to get the path to the current binary if running as yagwt
	if execPath, err := os.Executable(); err == nil {
		// Check if we're actually running as yagwt
		if filepath.Base(execPath) == "yagwt" {
			yagwtPath = execPath
		}
	}

	return MCPServerConfig{
		Command: yagwtPath,
		Args:    []string{"mcp-server"},
	}
}

// readMCPConfig reads and parses the .mcp.json file
func readMCPConfig(filename string) (*MCPConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return &MCPConfig{McpServers: make(map[string]MCPServerConfig)}, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", filename, err)
	}

	// Check if file is empty or whitespace only
	if len(bytes.TrimSpace(data)) == 0 {
		return &MCPConfig{McpServers: make(map[string]MCPServerConfig)}, nil
	}

	var config MCPConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", filename, err)
	}

	// Ensure McpServers map exists
	if config.McpServers == nil {
		config.McpServers = make(map[string]MCPServerConfig)
	}

	return &config, nil
}

// writeMCPConfig writes the MCP configuration to file with pretty printing
func writeMCPConfig(filename string, config *MCPConfig) error {
	// Remove empty mcpServers to avoid writing empty objects
	if len(config.McpServers) == 0 {
		// Write an empty mcpServers object
		data := []byte(`{"mcpServers": {}}`)

		// Ensure directory exists
		dir := filepath.Dir(filename)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Write with proper permissions
		if err := os.WriteFile(filename, data, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
		return nil
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal MCP config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write with proper permissions
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", filename, err)
	}

	return nil
}

// addYagwtMCPConfig adds yagwt to the MCP configuration
func addYagwtMCPConfig(config *MCPConfig) bool {
	const yagwtServerName = "yagwt"

	if _, exists := config.McpServers[yagwtServerName]; !exists {
		config.McpServers[yagwtServerName] = getYagwtMCPConfig()
		return true
	}
	return false
}

// removeYagwtMCPConfig removes yagwt from the MCP configuration
func removeYagwtMCPConfig(config *MCPConfig) bool {
	const yagwtServerName = "yagwt"

	if _, exists := config.McpServers[yagwtServerName]; exists {
		delete(config.McpServers, yagwtServerName)
		return true
	}
	return false
}

func runMCP(cmd *cobra.Command, args []string) {
	initFormatter()

	const mcpConfigFile = ".mcp.json"

	// Check if file exists and has content, validate JSON if present
	if fileInfo, err := os.Stat(mcpConfigFile); err == nil && fileInfo.Size() > 0 {
		data, err := os.ReadFile(mcpConfigFile)
		if err != nil {
			handleError(fmt.Errorf("failed to read %s: %w", mcpConfigFile, err))
			return
		}

		// If file has content, validate it's valid JSON
		if len(bytes.TrimSpace(data)) > 0 {
			var test interface{}
			if json.Unmarshal(data, &test) != nil {
				fmt.Fprintf(os.Stderr, "Error: %s exists but contains invalid JSON\n", mcpConfigFile)
				fmt.Fprintf(os.Stderr, "Please fix the JSON or remove the file manually\n")
				os.Exit(ExitFailure)
			}
		}
	}

	// Read current configuration
	config, err := readMCPConfig(mcpConfigFile)
	if err != nil {
		handleError(fmt.Errorf("failed to read MCP configuration: %w", err))
		return
	}

	var changed bool
	var action string

	if mcpRemove {
		changed = removeYagwtMCPConfig(config)
		action = "removed"
	} else {
		changed = addYagwtMCPConfig(config)
		action = "added"
	}

	if !changed {
		if mcpRemove {
			// Use regular printOutput for informational message
			printOutput("ℹ yagwt MCP configuration not found in .mcp.json\n")
		} else {
			// Use regular printOutput for informational message
			printOutput("ℹ yagwt MCP configuration already exists in .mcp.json\n")
		}
		return
	}

	// Write updated configuration
	if err := writeMCPConfig(mcpConfigFile, config); err != nil {
		handleError(fmt.Errorf("failed to write MCP configuration: %w", err))
		return
	}

	// Validate the written file
	if data, err := os.ReadFile(mcpConfigFile); err == nil {
		var test interface{}
		if json.Unmarshal(data, &test) != nil {
			fmt.Fprintf(os.Stderr, "Error: Wrote invalid JSON to %s\n", mcpConfigFile)
			os.Exit(ExitFailure)
		}
	}

	printOutput(formatter.FormatSuccess(fmt.Sprintf("yagwt MCP configuration %s to .mcp.json", action)))
}
