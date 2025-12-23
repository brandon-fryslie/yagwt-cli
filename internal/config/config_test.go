package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bmf/yagwt/internal/errors"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Workspace.RootStrategy != "sibling" {
		t.Errorf("Expected rootStrategy 'sibling', got %q", config.Workspace.RootStrategy)
	}

	if config.Workspace.RootDir != ".workspaces" {
		t.Errorf("Expected rootDir '.workspaces', got %q", config.Workspace.RootDir)
	}

	if config.Workspace.NameTemplate != "{branch}" {
		t.Errorf("Expected nameTemplate '{branch}', got %q", config.Workspace.NameTemplate)
	}

	// Check default policies exist
	if _, ok := config.Cleanup.Policies["default"]; !ok {
		t.Error("Expected 'default' cleanup policy")
	}

	if _, ok := config.Cleanup.Policies["conservative"]; !ok {
		t.Error("Expected 'conservative' cleanup policy")
	}

	if _, ok := config.Cleanup.Policies["aggressive"]; !ok {
		t.Error("Expected 'aggressive' cleanup policy")
	}
}

func TestLoadDefault(t *testing.T) {
	// Load with no config files
	config, err := Load("/nonexistent/path", "")
	if err != nil {
		t.Fatalf("Failed to load default config: %v", err)
	}

	if config.Workspace.RootStrategy != "sibling" {
		t.Errorf("Expected default rootStrategy 'sibling', got %q", config.Workspace.RootStrategy)
	}
}

func TestLoadFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	configDir := filepath.Join(repoDir, ".yagwt")

	// Create config directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Write config file
	configPath := filepath.Join(configDir, "config.toml")
	configContent := `
[workspace]
rootStrategy = "inside"
rootDir = "my-workspaces"
nameTemplate = "{branch}-workspace"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load config
	config, err := Load(repoDir, "")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify overrides
	if config.Workspace.RootStrategy != "inside" {
		t.Errorf("Expected rootStrategy 'inside', got %q", config.Workspace.RootStrategy)
	}

	if config.Workspace.RootDir != "my-workspaces" {
		t.Errorf("Expected rootDir 'my-workspaces', got %q", config.Workspace.RootDir)
	}

	if config.Workspace.NameTemplate != "{branch}-workspace" {
		t.Errorf("Expected nameTemplate '{branch}-workspace', got %q", config.Workspace.NameTemplate)
	}
}

func TestLoadExplicitPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "custom-config.toml")

	// Write config file
	configContent := `
[workspace]
rootStrategy = "inside"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load with explicit path
	config, err := Load("", configPath)
	if err != nil {
		t.Fatalf("Failed to load config with explicit path: %v", err)
	}

	if config.Workspace.RootStrategy != "inside" {
		t.Errorf("Expected rootStrategy 'inside', got %q", config.Workspace.RootStrategy)
	}
}

func TestValidateRootStrategy(t *testing.T) {
	tests := []struct {
		name     string
		strategy string
		wantErr  bool
	}{
		{"valid sibling", "sibling", false},
		{"valid inside", "inside", false},
		{"invalid", "invalid", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.Workspace.RootStrategy = tt.strategy

			err := validateConfig(config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				coreErr, ok := err.(*errors.Error)
				if !ok {
					t.Fatalf("Expected *errors.Error, got %T", err)
				}

				if coreErr.Code != errors.ErrConfig {
					t.Errorf("Expected error code %s, got %s", errors.ErrConfig, coreErr.Code)
				}
			}
		})
	}
}

func TestValidateOnDirty(t *testing.T) {
	tests := []struct {
		name    string
		onDirty string
		wantErr bool
	}{
		{"valid fail", "fail", false},
		{"valid stash", "stash", false},
		{"valid patch", "patch", false},
		{"valid wip-commit", "wip-commit", false},
		{"invalid", "invalid-value", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.Cleanup.Policies["test"] = CleanupPolicy{
				OnDirty: tt.onDirty,
			}

			err := validateConfig(config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				coreErr, ok := err.(*errors.Error)
				if !ok {
					t.Fatalf("Expected *errors.Error, got %T", err)
				}

				if coreErr.Code != errors.ErrConfig {
					t.Errorf("Expected error code %s, got %s", errors.ErrConfig, coreErr.Code)
				}
			}
		})
	}
}

func TestInvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Write invalid TOML
	if err := os.WriteFile(configPath, []byte("{invalid toml"), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load should fail
	_, err := Load("", configPath)
	if err == nil {
		t.Fatal("Expected error for invalid TOML")
	}

	coreErr, ok := err.(*errors.Error)
	if !ok {
		t.Fatalf("Expected *errors.Error, got %T", err)
	}

	if coreErr.Code != errors.ErrConfig {
		t.Errorf("Expected error code %s, got %s", errors.ErrConfig, coreErr.Code)
	}
}

func TestConfigPrecedence(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")

	// Create repo config
	repoConfigDir := filepath.Join(repoDir, ".yagwt")
	if err := os.MkdirAll(repoConfigDir, 0755); err != nil {
		t.Fatalf("Failed to create repo config dir: %v", err)
	}

	repoConfigPath := filepath.Join(repoConfigDir, "config.toml")
	repoConfig := `
[workspace]
rootStrategy = "inside"
`
	if err := os.WriteFile(repoConfigPath, []byte(repoConfig), 0644); err != nil {
		t.Fatalf("Failed to write repo config: %v", err)
	}

	// Create explicit config (higher precedence)
	explicitConfigPath := filepath.Join(tmpDir, "explicit.toml")
	explicitConfig := `
[workspace]
rootStrategy = "sibling"
`
	if err := os.WriteFile(explicitConfigPath, []byte(explicitConfig), 0644); err != nil {
		t.Fatalf("Failed to write explicit config: %v", err)
	}

	// Load with both configs - explicit should win
	config, err := Load(repoDir, explicitConfigPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.Workspace.RootStrategy != "sibling" {
		t.Errorf("Expected explicit config to take precedence, got %q", config.Workspace.RootStrategy)
	}
}

func TestMergeConfig(t *testing.T) {
	base := DefaultConfig()
	override := &Config{
		Workspace: WorkspaceConfig{
			RootStrategy: "inside",
			// RootDir and NameTemplate left empty - should keep defaults
		},
		Hooks: HooksConfig{
			PostCreate: "/usr/local/bin/post-create.sh",
		},
	}

	merged := mergeConfig(base, override)

	// Override values should be set
	if merged.Workspace.RootStrategy != "inside" {
		t.Errorf("Expected merged rootStrategy 'inside', got %q", merged.Workspace.RootStrategy)
	}

	// Default values should remain
	if merged.Workspace.RootDir != ".workspaces" {
		t.Errorf("Expected default rootDir '.workspaces', got %q", merged.Workspace.RootDir)
	}

	if merged.Workspace.NameTemplate != "{branch}" {
		t.Errorf("Expected default nameTemplate '{branch}', got %q", merged.Workspace.NameTemplate)
	}

	// Hook should be set
	if merged.Hooks.PostCreate != "/usr/local/bin/post-create.sh" {
		t.Errorf("Expected postCreate hook to be set, got %q", merged.Hooks.PostCreate)
	}

	// Default policies should still exist
	if _, ok := merged.Cleanup.Policies["default"]; !ok {
		t.Error("Expected default cleanup policy to remain after merge")
	}
}

func TestCleanupPolicyDuration(t *testing.T) {
	config := DefaultConfig()

	// Check default policy durations
	defaultPolicy := config.Cleanup.Policies["default"]
	expectedDuration := 30 * 24 * time.Hour

	if defaultPolicy.IdleThreshold != expectedDuration {
		t.Errorf("Expected idle threshold %v, got %v", expectedDuration, defaultPolicy.IdleThreshold)
	}

	conservativePolicy := config.Cleanup.Policies["conservative"]
	expectedDuration = 90 * 24 * time.Hour

	if conservativePolicy.IdleThreshold != expectedDuration {
		t.Errorf("Expected conservative idle threshold %v, got %v", expectedDuration, conservativePolicy.IdleThreshold)
	}

	aggressivePolicy := config.Cleanup.Policies["aggressive"]
	expectedDuration = 7 * 24 * time.Hour

	if aggressivePolicy.IdleThreshold != expectedDuration {
		t.Errorf("Expected aggressive idle threshold %v, got %v", expectedDuration, aggressivePolicy.IdleThreshold)
	}
}

func TestHooksConfig(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	configDir := filepath.Join(repoDir, ".yagwt")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.toml")
	configContent := `
[hooks]
postCreate = "/usr/local/bin/post-create.sh"
preRemove = "/usr/local/bin/pre-remove.sh"
postRemove = "/usr/local/bin/post-remove.sh"
postOpen = "/usr/local/bin/post-open.sh"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := Load(repoDir, "")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.Hooks.PostCreate != "/usr/local/bin/post-create.sh" {
		t.Errorf("Expected postCreate hook, got %q", config.Hooks.PostCreate)
	}

	if config.Hooks.PreRemove != "/usr/local/bin/pre-remove.sh" {
		t.Errorf("Expected preRemove hook, got %q", config.Hooks.PreRemove)
	}

	if config.Hooks.PostRemove != "/usr/local/bin/post-remove.sh" {
		t.Errorf("Expected postRemove hook, got %q", config.Hooks.PostRemove)
	}

	if config.Hooks.PostOpen != "/usr/local/bin/post-open.sh" {
		t.Errorf("Expected postOpen hook, got %q", config.Hooks.PostOpen)
	}
}
