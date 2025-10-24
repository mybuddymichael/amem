package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"amem/keyring"
)

// Config represents configuration at either ~/.config/amem/config.json or .amem/config.json
type Config struct {
	DBPath string `json:"db_path"`
}

// LoadedConfig contains both the database path and encryption key ready for use.
type LoadedConfig struct {
	DBPath        string
	EncryptionKey string
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

// FindLocal walks up the directory tree from startDir looking for .amem/config.json.
// Returns the path to the config file if found, or an error if not found.
func FindLocal(startDir string) (string, error) {
	current := startDir

	for {
		configPath := LocalPath(current)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		parent := filepath.Dir(current)
		// Reached filesystem root
		if parent == current {
			return "", fmt.Errorf("no local config found: %w", os.ErrNotExist)
		}
		current = parent
	}
}

// Load discovers and loads config with encryption key.
// Searches for local config first (walking up from cwd), then falls back to global config.
// Returns helpful error if no config exists.
func Load() (*LoadedConfig, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Try local config first
	localPath, err := FindLocal(cwd)
	if err == nil {
		cfg, err := Read(localPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read local config at %s: %w", localPath, err)
		}

		// Get directory containing .amem (parent of config file's parent)
		configDir := filepath.Dir(localPath)  // .amem directory
		projectDir := filepath.Dir(configDir) // project directory
		account := "local:" + projectDir

		key, err := keyring.Get(account)
		if err != nil {
			return nil, fmt.Errorf("failed to load encryption key for local config at %s: %w", projectDir, err)
		}

		return &LoadedConfig{
			DBPath:        cfg.DBPath,
			EncryptionKey: key,
		}, nil
	}

	// If local not found, try global
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("error searching for local config: %w", err)
	}

	globalPath, err := GlobalPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get global config path: %w", err)
	}

	cfg, err := Read(globalPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("no config found: run 'amem init' to create one")
		}
		return nil, fmt.Errorf("failed to read global config at %s: %w", globalPath, err)
	}

	key, err := keyring.Get("global")
	if err != nil {
		return nil, fmt.Errorf("failed to load encryption key for global config: %w", err)
	}

	return &LoadedConfig{
		DBPath:        cfg.DBPath,
		EncryptionKey: key,
	}, nil
}
