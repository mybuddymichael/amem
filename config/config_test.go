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
