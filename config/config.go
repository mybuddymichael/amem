package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents configuration at either ~/.config/amem/config.json or .amem/config.json
type Config struct {
	DBPath string `json:"db_path"`
}

// Read reads a config file from the given path.
// Returns os.ErrNotExist if the file doesn't exist.
func Read(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config JSON: %w", err)
	}

	if cfg.DBPath == "" {
		return nil, fmt.Errorf("config missing required field: db_path")
	}

	return &cfg, nil
}

// Write writes a config file to the given path.
// Creates parent directories if needed.
func Write(path string, cfg *Config) error {
	if cfg.DBPath == "" {
		return fmt.Errorf("config missing required field: db_path")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Atomic write: write to temp file then rename
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath) // cleanup on failure
		return fmt.Errorf("failed to rename config file: %w", err)
	}

	return nil
}

// GlobalPath returns the path to the global config file.
// On Unix: ~/.config/amem/config.json
// On macOS: ~/Library/Application Support/amem/config.json
// On Windows: %AppData%/amem/config.json
func GlobalPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}

	return filepath.Join(configDir, "amem", "config.json"), nil
}

// LocalPath returns the path to a local config file in the given directory.
// Does NOT search up the directory tree - just constructs the path.
func LocalPath(dir string) string {
	return filepath.Join(dir, ".amem", "config.json")
}
