package config

import (
	"encoding/json"
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
