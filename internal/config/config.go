package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the structure of .sprout.yml
type Config struct {
	Hooks HooksConfig `yaml:"hooks"`
}

// HooksConfig defines the hook configuration
type HooksConfig struct {
	OnCreate []string `yaml:"on_create"`
	OnOpen   []string `yaml:"on_open"`
}

// Load loads the .sprout.yml configuration with fallback support.
// It first checks currentPath for a worktree-specific config, then falls back
// to mainWorktreePath for a shared config (useful for gitignored configs).
// Returns an empty config if neither exists, or an error if parsing fails.
func Load(currentPath, mainWorktreePath string) (*Config, error) {
	// Try current path first (worktree-specific config)
	configPath := filepath.Join(currentPath, ".sprout.yml")
	data, err := os.ReadFile(configPath)

	if err != nil {
		if os.IsNotExist(err) {
			// Try main worktree path as fallback
			if mainWorktreePath != "" && mainWorktreePath != currentPath {
				configPath = filepath.Join(mainWorktreePath, ".sprout.yml")
				data, err = os.ReadFile(configPath)
				if err != nil {
					if os.IsNotExist(err) {
						// No config file in either location is fine, return empty config
						return &Config{}, nil
					}
					return nil, fmt.Errorf("failed to read config file from main worktree: %w", err)
				}
			} else {
				// No fallback available, return empty config
				return &Config{}, nil
			}
		} else {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate checks if the config is valid
func (c *Config) Validate() error {
	// Check that on_create commands are strings
	for i, cmd := range c.Hooks.OnCreate {
		if cmd == "" {
			return fmt.Errorf("on_create[%d] is empty", i)
		}
	}

	// Check that on_open commands are strings
	for i, cmd := range c.Hooks.OnOpen {
		if cmd == "" {
			return fmt.Errorf("on_open[%d] is empty", i)
		}
	}

	return nil
}

// HasHooks returns true if any hooks are defined
func (c *Config) HasHooks() bool {
	return len(c.Hooks.OnCreate) > 0 || len(c.Hooks.OnOpen) > 0
}

// HasCreateHooks returns true if on_create hooks are defined
func (c *Config) HasCreateHooks() bool {
	return len(c.Hooks.OnCreate) > 0
}

// HasOpenHooks returns true if on_open hooks are defined
func (c *Config) HasOpenHooks() bool {
	return len(c.Hooks.OnOpen) > 0
}
