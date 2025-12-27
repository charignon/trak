// Package config provides configuration loading and saving for trak.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the trak configuration.
type Config struct {
	Repo  RepoConfig  `yaml:"repo"`
	Cache CacheConfig `yaml:"cache,omitempty"`
}

// RepoConfig contains repository-related configuration.
type RepoConfig struct {
	Path   string `yaml:"path"`
	Remote string `yaml:"remote"`
}

// CacheConfig contains cached values that are updated periodically.
type CacheConfig struct {
	DefaultBranch        string `yaml:"default_branch,omitempty"`
	DefaultBranchUpdated string `yaml:"default_branch_updated,omitempty"`
}

// configDir returns the path to the trak config directory.
func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home is not available
		return ".config/trak"
	}
	return filepath.Join(home, ".config", "trak")
}

// configPath returns the path to the trak config file.
func configPath() string {
	return filepath.Join(configDir(), "config.yaml")
}

// EnsureConfigDir creates the config directory if it doesn't exist.
func EnsureConfigDir() error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	return nil
}

// GetWorktreeBaseDir returns the base directory for worktrees.
func GetWorktreeBaseDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home is not available
		return "worktrees"
	}
	return filepath.Join(home, "worktrees")
}

// Load reads the configuration from ~/.config/trak/config.yaml.
func Load() (*Config, error) {
	path := configPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// Save writes the configuration to ~/.config/trak/config.yaml.
func Save(cfg *Config) error {
	if err := EnsureConfigDir(); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	path := configPath()
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetDBPath returns the path to the trak database file.
func GetDBPath() string {
	return filepath.Join(configDir(), "trak.db")
}
