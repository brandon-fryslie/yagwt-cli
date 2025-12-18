package config

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/bmf/yagwt/internal/core"
	"github.com/pelletier/go-toml/v2"
)

// Config is the full configuration
type Config struct {
	Workspace WorkspaceConfig `toml:"workspace"`
	Cleanup   CleanupConfig   `toml:"cleanup"`
	Hooks     HooksConfig     `toml:"hooks"`
}

// WorkspaceConfig controls workspace creation
type WorkspaceConfig struct {
	RootStrategy string `toml:"rootStrategy"` // "sibling" or "inside"
	RootDir      string `toml:"rootDir"`      // ".workspaces" (if inside)
	NameTemplate string `toml:"nameTemplate"` // "{branch}" or custom
}

// CleanupConfig defines cleanup policies
type CleanupConfig struct {
	Policies map[string]CleanupPolicy `toml:"policies"`
}

// CleanupPolicy defines rules for cleanup
type CleanupPolicy struct {
	RemoveEphemeral bool          `toml:"removeEphemeral"`
	IdleThreshold   time.Duration `toml:"idleThreshold"`
	RespectPinned   bool          `toml:"respectPinned"`
	OnDirty         string        `toml:"onDirty"` // fail, stash, patch, wip-commit
}

// HooksConfig defines hook scripts
type HooksConfig struct {
	PostCreate string `toml:"postCreate"`
	PreRemove  string `toml:"preRemove"`
	PostRemove string `toml:"postRemove"`
	PostOpen   string `toml:"postOpen"`
}

// Load loads configuration from multiple sources with precedence
func Load(repoRoot string, configPath string) (*Config, error) {
	// Start with defaults
	config := DefaultConfig()

	// Config file search paths (in order of precedence)
	var configPaths []string

	// 1. Explicit config path (highest priority)
	if configPath != "" {
		configPaths = append(configPaths, configPath)
	}

	// 2. Repository-level config
	if repoRoot != "" {
		configPaths = append(configPaths, filepath.Join(repoRoot, ".yagwt", "config.toml"))
	}

	// 3. User-level config
	userConfigPath, err := getUserConfigPath()
	if err == nil {
		configPaths = append(configPaths, userConfigPath)
	}

	// Try each config path in order
	for _, path := range configPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		// Read and parse config
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, core.WrapError(core.ErrConfig, "failed to read config file", err).
				WithDetail("path", path)
		}

		// Parse TOML
		var fileConfig Config
		if err := toml.Unmarshal(data, &fileConfig); err != nil {
			return nil, core.WrapError(core.ErrConfig, "failed to parse config file", err).
				WithDetail("path", path).
				WithHint("Check TOML syntax", "")
		}

		// Merge with defaults (file config overrides defaults)
		config = mergeConfig(config, &fileConfig)

		// Only use the first found config file
		break
	}

	// Validate config
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Workspace: WorkspaceConfig{
			RootStrategy: "sibling",
			RootDir:      ".workspaces",
			NameTemplate: "{branch}",
		},
		Cleanup: CleanupConfig{
			Policies: map[string]CleanupPolicy{
				"default": {
					RemoveEphemeral: true,
					IdleThreshold:   30 * 24 * time.Hour, // 30 days
					RespectPinned:   true,
					OnDirty:         "fail",
				},
				"conservative": {
					RemoveEphemeral: true,
					IdleThreshold:   90 * 24 * time.Hour, // 90 days
					RespectPinned:   true,
					OnDirty:         "fail",
				},
				"aggressive": {
					RemoveEphemeral: true,
					IdleThreshold:   7 * 24 * time.Hour, // 7 days
					RespectPinned:   false,
					OnDirty:         "stash",
				},
			},
		},
		Hooks: HooksConfig{},
	}
}

// getUserConfigPath returns the user-level config path based on OS
func getUserConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Check OS-specific paths
	if runtime.GOOS == "darwin" {
		// macOS: ~/Library/Application Support/yagwt/config.toml
		return filepath.Join(homeDir, "Library", "Application Support", "yagwt", "config.toml"), nil
	}

	// Linux/Unix: $XDG_CONFIG_HOME/yagwt/config.toml or ~/.config/yagwt/config.toml
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "yagwt", "config.toml"), nil
	}

	return filepath.Join(homeDir, ".config", "yagwt", "config.toml"), nil
}

// mergeConfig merges file config with defaults
func mergeConfig(base *Config, override *Config) *Config {
	result := *base

	// Merge workspace config
	if override.Workspace.RootStrategy != "" {
		result.Workspace.RootStrategy = override.Workspace.RootStrategy
	}
	if override.Workspace.RootDir != "" {
		result.Workspace.RootDir = override.Workspace.RootDir
	}
	if override.Workspace.NameTemplate != "" {
		result.Workspace.NameTemplate = override.Workspace.NameTemplate
	}

	// Merge cleanup policies
	if override.Cleanup.Policies != nil {
		if result.Cleanup.Policies == nil {
			result.Cleanup.Policies = make(map[string]CleanupPolicy)
		}
		for name, policy := range override.Cleanup.Policies {
			result.Cleanup.Policies[name] = policy
		}
	}

	// Merge hooks
	if override.Hooks.PostCreate != "" {
		result.Hooks.PostCreate = override.Hooks.PostCreate
	}
	if override.Hooks.PreRemove != "" {
		result.Hooks.PreRemove = override.Hooks.PreRemove
	}
	if override.Hooks.PostRemove != "" {
		result.Hooks.PostRemove = override.Hooks.PostRemove
	}
	if override.Hooks.PostOpen != "" {
		result.Hooks.PostOpen = override.Hooks.PostOpen
	}

	return &result
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Validate root strategy
	if config.Workspace.RootStrategy != "sibling" && config.Workspace.RootStrategy != "inside" {
		return core.NewError(core.ErrConfig, "invalid rootStrategy").
			WithDetail("value", config.Workspace.RootStrategy).
			WithDetail("valid", "sibling, inside")
	}

	// Validate onDirty values in policies
	validOnDirty := map[string]bool{
		"fail":       true,
		"stash":      true,
		"patch":      true,
		"wip-commit": true,
	}

	for name, policy := range config.Cleanup.Policies {
		if policy.OnDirty != "" && !validOnDirty[policy.OnDirty] {
			return core.NewError(core.ErrConfig, "invalid onDirty value in cleanup policy").
				WithDetail("policy", name).
				WithDetail("value", policy.OnDirty).
				WithDetail("valid", "fail, stash, patch, wip-commit")
		}
	}

	return nil
}
