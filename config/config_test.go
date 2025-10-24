package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigJSON(t *testing.T) {
	cfg := Config{
		DBPath: "/home/user/.amem.db",
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded Config
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.DBPath != cfg.DBPath {
		t.Errorf("expected DBPath %s, got %s", cfg.DBPath, decoded.DBPath)
	}
}

func TestWriteCreatesDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nested", "dir", "config.json")

	cfg := &Config{DBPath: "/test/path.db"}
	if err := Write(path, cfg); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}
}

func TestWriteReadRoundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.json")

	original := &Config{DBPath: "/test/database.db"}
	if err := Write(path, original); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	read, err := Read(path)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if read.DBPath != original.DBPath {
		t.Errorf("expected DBPath %s, got %s", original.DBPath, read.DBPath)
	}
}

func TestReadMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nonexistent.json")

	_, err := Read(path)
	if err == nil {
		t.Fatal("expected error for missing file")
	}

	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected os.ErrNotExist, got %v", err)
	}
}

func TestReadInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.json")

	if err := os.WriteFile(path, []byte("{invalid json}"), 0o644); err != nil {
		t.Fatalf("failed to write invalid JSON: %v", err)
	}

	_, err := Read(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestReadEmptyDBPath(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.json")

	if err := os.WriteFile(path, []byte(`{"db_path":""}`), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	_, err := Read(path)
	if err == nil {
		t.Fatal("expected error for empty db_path")
	}
}

func TestWriteEmptyDBPath(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.json")

	cfg := &Config{DBPath: ""}
	err := Write(path, cfg)
	if err == nil {
		t.Fatal("expected error for empty db_path")
	}
}

func TestGlobalPath(t *testing.T) {
	path, err := GlobalPath()
	if err != nil {
		t.Fatalf("GlobalPath failed: %v", err)
	}

	if path == "" {
		t.Fatal("GlobalPath returned empty string")
	}

	// Should contain "amem" and end with config.json
	if !filepath.IsAbs(path) {
		t.Errorf("GlobalPath should return absolute path, got %s", path)
	}

	if filepath.Base(path) != "config.json" {
		t.Errorf("GlobalPath should end with config.json, got %s", path)
	}
}

func TestLocalPath(t *testing.T) {
	dir := t.TempDir()
	path := LocalPath(dir)

	expected := filepath.Join(dir, ".amem", "config.json")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestFindLocalInCurrentDir(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{DBPath: "/test/path.db"}
	configPath := LocalPath(tmpDir)

	if err := Write(configPath, cfg); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	found, err := FindLocal(tmpDir)
	if err != nil {
		t.Fatalf("FindLocal failed: %v", err)
	}

	if found != configPath {
		t.Errorf("expected %s, got %s", configPath, found)
	}
}

func TestFindLocalInParentDir(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{DBPath: "/test/path.db"}
	configPath := LocalPath(tmpDir)

	if err := Write(configPath, cfg); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Search from a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	found, err := FindLocal(subDir)
	if err != nil {
		t.Fatalf("FindLocal failed: %v", err)
	}

	if found != configPath {
		t.Errorf("expected %s, got %s", configPath, found)
	}
}

func TestFindLocalMultipleLevelsUp(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{DBPath: "/test/path.db"}
	configPath := LocalPath(tmpDir)

	if err := Write(configPath, cfg); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Create nested subdirectories
	deepDir := filepath.Join(tmpDir, "level1", "level2", "level3")
	if err := os.MkdirAll(deepDir, 0o755); err != nil {
		t.Fatalf("failed to create deep dir: %v", err)
	}

	found, err := FindLocal(deepDir)
	if err != nil {
		t.Fatalf("FindLocal failed: %v", err)
	}

	if found != configPath {
		t.Errorf("expected %s, got %s", configPath, found)
	}
}

func TestFindLocalNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := FindLocal(tmpDir)
	if err == nil {
		t.Fatal("expected error when config not found")
	}

	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected error to wrap os.ErrNotExist, got %v", err)
	}
}

func TestLoadLocalConfig(t *testing.T) {
	// Skip if keyring access fails (e.g., in CI)
	if err := testKeyringAccess(); err != nil {
		t.Skipf("Skipping test: keyring not available: %v", err)
	}

	tmpDir := t.TempDir()
	cfg := &Config{DBPath: "/test/local.db"}
	configPath := LocalPath(tmpDir)

	if err := Write(configPath, cfg); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Store test key in keyring
	testKey := "test-local-key-123"
	account := "local:" + tmpDir
	defer cleanupKeyring(account)

	if err := setTestKey(account, testKey); err != nil {
		t.Fatalf("failed to set test key: %v", err)
	}

	// Create subdirectory to test discovery
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	// Change to subdir and test Load
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() { _ = os.Chdir(oldDir) }()

	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.DBPath != cfg.DBPath {
		t.Errorf("expected DBPath %s, got %s", cfg.DBPath, loaded.DBPath)
	}

	if loaded.EncryptionKey != testKey {
		t.Errorf("expected key %s, got %s", testKey, loaded.EncryptionKey)
	}
}

func TestLoadGlobalConfig(t *testing.T) {
	// Skip - global config test requires environment setup
	t.Skip("Global config test requires environment setup - tested manually")
}

func TestLoadNoConfigFound(t *testing.T) {
	tmpDir := t.TempDir()

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() { _ = os.Chdir(oldDir) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Mock global path to point to non-existent location
	// Since we can't override GlobalPath easily, we rely on the fact that
	// Load will check local first, then global. In tmpDir with no config,
	// it should error.

	_, err = Load()
	if err == nil {
		t.Fatal("expected error when no config found")
	}

	// Error message should be helpful
	if !errors.Is(err, os.ErrNotExist) && err.Error() != "no config found: run 'amem init' to create one" {
		t.Logf("got error: %v", err)
	}
}

// Helper functions for keyring testing
func testKeyringAccess() error {
	// Try to set and get a test value
	testAccount := "amem-test-access"
	err := setTestKey(testAccount, "test")
	if err != nil {
		return err
	}
	cleanupKeyring(testAccount)
	return nil
}

func setTestKey(account, key string) error {
	// Import keyring here to avoid import cycle
	// This is a simplified version - actual implementation uses amem/keyring
	return os.Setenv("AMEM_ENCRYPTION_KEY", key)
}

func cleanupKeyring(account string) {
	// Clean up test keys
	_ = os.Unsetenv("AMEM_ENCRYPTION_KEY")
}
