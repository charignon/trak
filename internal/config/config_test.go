package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureConfigDir(t *testing.T) {
	// Create a temp directory to use as home
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	err := EnsureConfigDir()
	if err != nil {
		t.Fatalf("EnsureConfigDir() error = %v", err)
	}

	// Check that the directory was created
	expectedDir := filepath.Join(tmpHome, ".config", "trak")
	info, err := os.Stat(expectedDir)
	if err != nil {
		t.Fatalf("config directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("config path is not a directory")
	}

	// Calling again should not error
	err = EnsureConfigDir()
	if err != nil {
		t.Fatalf("EnsureConfigDir() second call error = %v", err)
	}
}

func TestGetWorktreeBaseDir(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	got := GetWorktreeBaseDir()
	expected := filepath.Join(tmpHome, "worktrees")
	if got != expected {
		t.Errorf("GetWorktreeBaseDir() = %v, want %v", got, expected)
	}
}

func TestLoadNotFound(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	_, err := Load()
	if err == nil {
		t.Fatalf("Load() expected error for missing config, got nil")
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	cfg := &Config{
		Repo: RepoConfig{
			Path:   "/path/to/repo",
			Remote: "git@github.com:user/repo.git",
		},
		Cache: CacheConfig{
			DefaultBranch:        "main",
			DefaultBranchUpdated: "2025-12-26T12:00:00Z",
		},
	}

	// Save
	err := Save(cfg)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tmpHome, ".config", "trak", "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	// Load
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify values
	if loaded.Repo.Path != cfg.Repo.Path {
		t.Errorf("Repo.Path = %v, want %v", loaded.Repo.Path, cfg.Repo.Path)
	}
	if loaded.Repo.Remote != cfg.Repo.Remote {
		t.Errorf("Repo.Remote = %v, want %v", loaded.Repo.Remote, cfg.Repo.Remote)
	}
	if loaded.Cache.DefaultBranch != cfg.Cache.DefaultBranch {
		t.Errorf("Cache.DefaultBranch = %v, want %v", loaded.Cache.DefaultBranch, cfg.Cache.DefaultBranch)
	}
	if loaded.Cache.DefaultBranchUpdated != cfg.Cache.DefaultBranchUpdated {
		t.Errorf("Cache.DefaultBranchUpdated = %v, want %v", loaded.Cache.DefaultBranchUpdated, cfg.Cache.DefaultBranchUpdated)
	}
}

func TestSaveOverwrite(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Save initial config
	cfg1 := &Config{
		Repo: RepoConfig{
			Path:   "/first/path",
			Remote: "git@github.com:user/first.git",
		},
	}
	if err := Save(cfg1); err != nil {
		t.Fatalf("Save() first config error = %v", err)
	}

	// Save new config (overwrite)
	cfg2 := &Config{
		Repo: RepoConfig{
			Path:   "/second/path",
			Remote: "git@github.com:user/second.git",
		},
	}
	if err := Save(cfg2); err != nil {
		t.Fatalf("Save() second config error = %v", err)
	}

	// Load and verify it's the second config
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Repo.Path != cfg2.Repo.Path {
		t.Errorf("Repo.Path = %v, want %v", loaded.Repo.Path, cfg2.Repo.Path)
	}
}

func TestGetDBPath(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	got := GetDBPath()
	expected := filepath.Join(tmpHome, ".config", "trak", "trak.db")
	if got != expected {
		t.Errorf("GetDBPath() = %v, want %v", got, expected)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Create config dir and write invalid YAML
	configDir := filepath.Join(tmpHome, ".config", "trak")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatalf("Load() expected error for invalid YAML, got nil")
	}
}

func TestEmptyConfig(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Save empty config
	cfg := &Config{}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load it back
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify empty values
	if loaded.Repo.Path != "" {
		t.Errorf("Repo.Path = %v, want empty", loaded.Repo.Path)
	}
	if loaded.Repo.Remote != "" {
		t.Errorf("Repo.Remote = %v, want empty", loaded.Repo.Remote)
	}
}
