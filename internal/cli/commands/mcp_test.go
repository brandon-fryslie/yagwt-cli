package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGetYagwtMCPConfig(t *testing.T) {
	config := getYagwtMCPConfig()

	if config.Command != "yagwt" {
		t.Errorf("Expected command to be 'yagwt', got %s", config.Command)
	}

	if len(config.Args) != 1 || config.Args[0] != "mcp-server" {
		t.Errorf("Expected args to be ['mcp-server'], got %v", config.Args)
	}
}

func TestReadMCPConfig_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, ".mcp.json")

	config, err := readMCPConfig(filename)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if config == nil {
		t.Fatal("Expected config to not be nil")
	}

	if config.McpServers == nil {
		t.Fatal("Expected McpServers to be initialized")
	}

	if len(config.McpServers) != 0 {
		t.Errorf("Expected empty McpServers map, got %d entries", len(config.McpServers))
	}
}

func TestReadMCPConfig_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, ".mcp.json")

	// Create empty file
	if err := os.WriteFile(filename, []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	config, err := readMCPConfig(filename)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if config.McpServers == nil {
		t.Fatal("Expected McpServers to be initialized")
	}

	if len(config.McpServers) != 0 {
		t.Errorf("Expected empty McpServers map, got %d entries", len(config.McpServers))
	}
}

func TestReadMCPConfig_WhitespaceFile(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, ".mcp.json")

	// Create file with only whitespace
	if err := os.WriteFile(filename, []byte("   \n\t  \n"), 0644); err != nil {
		t.Fatalf("Failed to create whitespace file: %v", err)
	}

	config, err := readMCPConfig(filename)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if config.McpServers == nil {
		t.Fatal("Expected McpServers to be initialized")
	}

	if len(config.McpServers) != 0 {
		t.Errorf("Expected empty McpServers map, got %d entries", len(config.McpServers))
	}
}

func TestReadMCPConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, ".mcp.json")

	// Create file with invalid JSON
	if err := os.WriteFile(filename, []byte(`{"invalid": json}`), 0644); err != nil {
		t.Fatalf("Failed to create invalid JSON file: %v", err)
	}

	_, err := readMCPConfig(filename)

	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}

	expectedMsg := "invalid JSON in " + filename
	if err.Error()[:len(expectedMsg)] != expectedMsg {
		t.Errorf("Expected error message to start with '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestReadMCPConfig_ValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, ".mcp.json")

	// Create file with valid JSON
	testData := `{"mcpServers": {"test": {"command": "test", "args": []}}}`
	if err := os.WriteFile(filename, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to create test JSON file: %v", err)
	}

	config, err := readMCPConfig(filename)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(config.McpServers) != 1 {
		t.Fatalf("Expected 1 server in config, got %d", len(config.McpServers))
	}

	server, exists := config.McpServers["test"]
	if !exists {
		t.Fatal("Expected 'test' server to exist")
	}

	if server.Command != "test" {
		t.Errorf("Expected command 'test', got '%s'", server.Command)
	}
}

func TestReadMCPConfig_JSONWithoutMcpServers(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, ".mcp.json")

	// Create file with JSON but no mcpServers
	testData := `{"otherField": "value"}`
	if err := os.WriteFile(filename, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to create test JSON file: %v", err)
	}

	config, err := readMCPConfig(filename)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if config.McpServers == nil {
		t.Fatal("Expected McpServers to be initialized")
	}

	if len(config.McpServers) != 0 {
		t.Errorf("Expected empty McpServers map, got %d entries", len(config.McpServers))
	}
}

func TestWriteMCPConfig_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, ".mcp.json")

	config := &MCPConfig{
		McpServers: map[string]MCPServerConfig{
			"yagwt": {
				Command: "yagwt",
				Args:    []string{"mcp-server"},
			},
		},
	}

	err := writeMCPConfig(filename, config)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify file contents
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	var writtenConfig MCPConfig
	if err := json.Unmarshal(data, &writtenConfig); err != nil {
		t.Fatalf("Failed to parse written JSON: %v", err)
	}

	if len(writtenConfig.McpServers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(writtenConfig.McpServers))
	}

	yagwt, exists := writtenConfig.McpServers["yagwt"]
	if !exists {
		t.Fatal("Expected 'yagwt' server to exist")
	}

	if yagwt.Command != "yagwt" {
		t.Errorf("Expected command 'yagwt', got '%s'", yagwt.Command)
	}
}

func TestWriteMCPConfig_EmptyMcpServers(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, ".mcp.json")

	config := &MCPConfig{
		McpServers: map[string]MCPServerConfig{},
	}

	err := writeMCPConfig(filename, config)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify file contents
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	expected := `{"mcpServers": {}}`
	if string(data) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(data))
	}
}

func TestAddYagwtMCPConfig_NotExists(t *testing.T) {
	config := &MCPConfig{
		McpServers: map[string]MCPServerConfig{
			"other": {
				Command: "other",
				Args:    []string{},
			},
		},
	}

	changed := addYagwtMCPConfig(config)

	if !changed {
		t.Error("Expected config to change")
	}

	if len(config.McpServers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(config.McpServers))
	}

	yagwt, exists := config.McpServers["yagwt"]
	if !exists {
		t.Fatal("Expected 'yagwt' server to be added")
	}

	if yagwt.Command != "yagwt" {
		t.Errorf("Expected command 'yagwt', got '%s'", yagwt.Command)
	}

	// Verify other server is preserved
	if _, exists := config.McpServers["other"]; !exists {
		t.Fatal("Expected 'other' server to be preserved")
	}
}

func TestAddYagwtMCPConfig_AlreadyExists(t *testing.T) {
	config := &MCPConfig{
		McpServers: map[string]MCPServerConfig{
			"yagwt": {
				Command: "existing",
				Args:    []string{},
			},
		},
	}

	changed := addYagwtMCPConfig(config)

	if changed {
		t.Error("Expected config to not change")
	}

	if len(config.McpServers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(config.McpServers))
	}

	// Verify existing config wasn't modified
	yagwt := config.McpServers["yagwt"]
	if yagwt.Command != "existing" {
		t.Errorf("Expected command 'existing', got '%s'", yagwt.Command)
	}
}

func TestRemoveYagwtMCPConfig_Exists(t *testing.T) {
	config := &MCPConfig{
		McpServers: map[string]MCPServerConfig{
			"yagwt": {
				Command: "yagwt",
				Args:    []string{},
			},
			"other": {
				Command: "other",
				Args:    []string{},
			},
		},
	}

	changed := removeYagwtMCPConfig(config)

	if !changed {
		t.Error("Expected config to change")
	}

	if len(config.McpServers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(config.McpServers))
	}

	if _, exists := config.McpServers["yagwt"]; exists {
		t.Error("Expected 'yagwt' server to be removed")
	}

	// Verify other server is preserved
	if _, exists := config.McpServers["other"]; !exists {
		t.Fatal("Expected 'other' server to be preserved")
	}
}

func TestRemoveYagwtMCPConfig_NotExists(t *testing.T) {
	config := &MCPConfig{
		McpServers: map[string]MCPServerConfig{
			"other": {
				Command: "other",
				Args:    []string{},
			},
		},
	}

	changed := removeYagwtMCPConfig(config)

	if changed {
		t.Error("Expected config to not change")
	}

	if len(config.McpServers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(config.McpServers))
	}
}
