// Package config manages the CLI's two config files.
//
// ~/.ops0/config.yaml      — user-wide credentials and defaults
// <repo>/.ops0/config.json — per-repo binding to an ops0 IaC project
//
// We deliberately use two different formats: YAML for the human-edited user
// config (comments, sections), JSON for the machine-managed repo config so
// it's safe to check into source control without surprises.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// User-wide config ────────────────────────────────────────────────────────────

// UserConfig is the on-disk shape of ~/.ops0/config.yaml.
type UserConfig struct {
	// APIBaseURL is the ops0 backend (e.g. https://brew.ops0.ai).
	// Override per-host so we can hit staging from a dev machine without
	// re-authenticating production.
	APIBaseURL string `yaml:"api_base_url"`

	// APIKey is the user's personal access token, scoped per the choices
	// made when generating it in the ops0 settings UI.
	APIKey string `yaml:"api_key"`

	// Telemetry controls whether anonymous check results (pass/fail counts,
	// templateIds — never source code) are reported back to ops0.
	Telemetry bool `yaml:"telemetry"`
}

// UserConfigPath returns ~/.ops0/config.yaml. Uses XDG_CONFIG_HOME if set so
// dotfiles purists can move it.
func UserConfigPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "ops0", "config.yaml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ops0", "config.yaml"), nil
}

// LoadUser reads the user config. Returns a zero-value UserConfig with
// defaults if the file doesn't exist — this is intentional so the CLI is
// usable before `ops0 login`.
func LoadUser() (*UserConfig, error) {
	path, err := UserConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &UserConfig{
			APIBaseURL: "https://brew.ops0.ai",
			Telemetry:  true,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	cfg := &UserConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if cfg.APIBaseURL == "" {
		cfg.APIBaseURL = "https://brew.ops0.ai"
	}
	return cfg, nil
}

// SaveUser writes the config, creating the parent directory with 0700 perms
// (since it contains a token).
func SaveUser(cfg *UserConfig) error {
	path, err := UserConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	// 0600: only the owner can read; the API key is in here.
	return os.WriteFile(path, data, 0o600)
}

// Per-repo config ────────────────────────────────────────────────────────────

// RepoConfig is the on-disk shape of <repo>/.ops0/config.json.
// Checked into the repo so every collaborator inherits the same binding.
type RepoConfig struct {
	// ProjectID is the ops0 IaC project this repo represents.
	// Empty = unbound; CLI falls back to org-wide policies only.
	ProjectID string `json:"projectId,omitempty"`

	// Paths constrains policy checks to specific subdirectories.
	// Empty = check entire repo. Useful for monorepos.
	Paths []string `json:"paths,omitempty"`

	// PolicyVersion pins the policy bundle version for reproducible checks.
	// Empty = always use latest. Pinning is recommended for CI.
	PolicyVersion string `json:"policyVersion,omitempty"`
}

// RepoConfigPath returns <cwd-or-given>/.ops0/config.json.
func RepoConfigPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".ops0", "config.json")
}

// LoadRepo reads the per-repo config. Returns (nil, nil) if no .ops0/ exists —
// distinguishable from an error so callers can decide whether to require it.
func LoadRepo(repoRoot string) (*RepoConfig, error) {
	path := RepoConfigPath(repoRoot)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	cfg := &RepoConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return cfg, nil
}

// FindRepo walks up from startPath (a directory or a file inside one) looking
// for the NEAREST `.ops0/config.json`. Returns (cfg, repoRoot, nil) when
// found, or (nil, "", nil) when no ancestor has one.
//
// This is the right resolver for monorepos where each subdirectory was
// initialized as its own ops0 project. Walking up from the file path means
// `ops0 policies check dir1/main.tf` and `ops0 policies check dir2/main.tf`
// resolve to different project IDs even when the CLI was launched from
// the shared repo root.
func FindRepo(startPath string) (*RepoConfig, string, error) {
	if startPath == "" {
		return nil, "", nil
	}
	dir, err := filepath.Abs(startPath)
	if err != nil {
		return nil, "", err
	}
	// If we were given a file path, start from its containing dir.
	if info, statErr := os.Stat(dir); statErr == nil && !info.IsDir() {
		dir = filepath.Dir(dir)
	}

	for {
		cfg, err := LoadRepo(dir)
		if err != nil {
			return nil, "", err
		}
		if cfg != nil {
			return cfg, dir, nil
		}
		parent := filepath.Dir(dir)
		// filepath.Dir("/") == "/" and filepath.Dir(".") == "." → stop.
		if parent == dir {
			return nil, "", nil
		}
		dir = parent
	}
}

// SaveRepo writes <repo>/.ops0/config.json. Uses 0644 — this file is meant
// to be readable and committed to git.
func SaveRepo(repoRoot string, cfg *RepoConfig) error {
	path := RepoConfigPath(repoRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	// trailing newline is friendlier for git diffs
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
