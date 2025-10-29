package view

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"amem/db"
)

// captureOutput runs a function and returns its stdout output
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestFormatEntitiesEmpty(t *testing.T) {
	output := captureOutput(func() {
		FormatEntities([]db.Entity{}, false)
	})

	expected := "No entities found\n"
	if output != expected {
		t.Errorf("Expected '%s', got '%s'", expected, output)
	}
}

func TestFormatEntitiesSingle(t *testing.T) {
	entities := []db.Entity{
		{ID: 1, Text: "Alice"},
	}

	// Without IDs
	output := captureOutput(func() {
		FormatEntities(entities, false)
	})

	if !strings.Contains(output, "Found 1 entities:") {
		t.Errorf("Expected header 'Found 1 entities:', got '%s'", output)
	}
	if !strings.Contains(output, "Alice") {
		t.Errorf("Expected 'Alice' in output, got '%s'", output)
	}
	if strings.Contains(output, "[1]") {
		t.Errorf("Should not contain ID '[1]', got '%s'", output)
	}

	// With IDs
	output = captureOutput(func() {
		FormatEntities(entities, true)
	})

	if !strings.Contains(output, "[1] Alice") {
		t.Errorf("Expected '[1] Alice' in output, got '%s'", output)
	}
}

func TestFormatEntitiesMultiple(t *testing.T) {
	entities := []db.Entity{
		{ID: 1, Text: "Alice"},
		{ID: 2, Text: "Bob"},
		{ID: 3, Text: "Charlie"},
	}

	output := captureOutput(func() {
		FormatEntities(entities, false)
	})

	if !strings.Contains(output, "Found 3 entities:") {
		t.Errorf("Expected header 'Found 3 entities:', got '%s'", output)
	}
	if !strings.Contains(output, "Alice") || !strings.Contains(output, "Bob") || !strings.Contains(output, "Charlie") {
		t.Errorf("Expected all entity names in output, got '%s'", output)
	}
}

func TestFormatObservationsEmpty(t *testing.T) {
	output := captureOutput(func() {
		FormatObservations([]db.Observation{}, false)
	})

	expected := "No observations found\n"
	if output != expected {
		t.Errorf("Expected '%s', got '%s'", expected, output)
	}
}

func TestFormatObservationsSingle(t *testing.T) {
	observations := []db.Observation{
		{ID: 1, EntityText: "Alice", Text: "Likes coffee", Timestamp: "2024-01-15 10:30:00"},
	}

	// Without IDs
	output := captureOutput(func() {
		FormatObservations(observations, false)
	})

	if !strings.Contains(output, "Found 1 observations:") {
		t.Errorf("Expected header 'Found 1 observations:', got '%s'", output)
	}
	if !strings.Contains(output, "Alice: Likes coffee") {
		t.Errorf("Expected 'Alice: Likes coffee' in output, got '%s'", output)
	}
	if strings.Contains(output, "[1]") {
		t.Errorf("Should not contain ID '[1]', got '%s'", output)
	}

	// With IDs
	output = captureOutput(func() {
		FormatObservations(observations, true)
	})

	if !strings.Contains(output, "[1] Alice: Likes coffee") {
		t.Errorf("Expected '[1] Alice: Likes coffee' in output, got '%s'", output)
	}
}

func TestFormatObservationsMultiple(t *testing.T) {
	observations := []db.Observation{
		{ID: 1, EntityText: "Alice", Text: "Likes coffee", Timestamp: "2024-01-15 10:30:00"},
		{ID: 2, EntityText: "Bob", Text: "Works remotely", Timestamp: "2024-01-15 11:00:00"},
	}

	output := captureOutput(func() {
		FormatObservations(observations, false)
	})

	if !strings.Contains(output, "Found 2 observations:") {
		t.Errorf("Expected header 'Found 2 observations:', got '%s'", output)
	}
	if !strings.Contains(output, "Alice: Likes coffee") || !strings.Contains(output, "Bob: Works remotely") {
		t.Errorf("Expected all observations in output, got '%s'", output)
	}
}

func TestFormatRelationshipsEmpty(t *testing.T) {
	output := captureOutput(func() {
		FormatRelationships([]db.Relationship{}, false)
	})

	expected := "No relationships found\n"
	if output != expected {
		t.Errorf("Expected '%s', got '%s'", expected, output)
	}
}

func TestFormatRelationshipsSingle(t *testing.T) {
	relationships := []db.Relationship{
		{ID: 1, FromText: "Alice", ToText: "Bob", Type: "knows", Timestamp: "2024-01-15 10:30:00"},
	}

	// Without IDs
	output := captureOutput(func() {
		FormatRelationships(relationships, false)
	})

	if !strings.Contains(output, "Found 1 relationships:") {
		t.Errorf("Expected header 'Found 1 relationships:', got '%s'", output)
	}
	if !strings.Contains(output, "Alice -[knows]-> Bob") {
		t.Errorf("Expected 'Alice -[knows]-> Bob' in output, got '%s'", output)
	}
	if strings.Contains(output, "[1]") {
		t.Errorf("Should not contain ID '[1]', got '%s'", output)
	}

	// With IDs
	output = captureOutput(func() {
		FormatRelationships(relationships, true)
	})

	if !strings.Contains(output, "[1] Alice -[knows]-> Bob") {
		t.Errorf("Expected '[1] Alice -[knows]-> Bob' in output, got '%s'", output)
	}
}

func TestFormatRelationshipsMultiple(t *testing.T) {
	relationships := []db.Relationship{
		{ID: 1, FromText: "Alice", ToText: "Bob", Type: "knows", Timestamp: "2024-01-15 10:30:00"},
		{ID: 2, FromText: "Bob", ToText: "Charlie", Type: "manages", Timestamp: "2024-01-15 11:00:00"},
	}

	output := captureOutput(func() {
		FormatRelationships(relationships, false)
	})

	if !strings.Contains(output, "Found 2 relationships:") {
		t.Errorf("Expected header 'Found 2 relationships:', got '%s'", output)
	}
	if !strings.Contains(output, "Alice -[knows]-> Bob") || !strings.Contains(output, "Bob -[manages]-> Charlie") {
		t.Errorf("Expected all relationships in output, got '%s'", output)
	}
}

func TestFormatAllEmpty(t *testing.T) {
	output := captureOutput(func() {
		FormatAll([]db.Entity{}, []db.Observation{}, []db.Relationship{}, false)
	})

	expected := "No results found\n"
	if output != expected {
		t.Errorf("Expected '%s', got '%s'", expected, output)
	}
}

func TestFormatAllMixed(t *testing.T) {
	entities := []db.Entity{
		{ID: 1, Text: "Alice"},
	}
	observations := []db.Observation{
		{ID: 1, EntityText: "Alice", Text: "Likes coffee", Timestamp: "2024-01-15 10:30:00"},
	}

	output := captureOutput(func() {
		FormatAll(entities, observations, []db.Relationship{}, false)
	})

	if !strings.Contains(output, "Entities (1):") {
		t.Errorf("Expected 'Entities (1):' header, got '%s'", output)
	}
	if !strings.Contains(output, "Observations (1):") {
		t.Errorf("Expected 'Observations (1):' header, got '%s'", output)
	}
	if strings.Contains(output, "Relationships") {
		t.Errorf("Should not contain Relationships section, got '%s'", output)
	}
}

func TestFormatAllPopulated(t *testing.T) {
	entities := []db.Entity{
		{ID: 1, Text: "Alice"},
		{ID: 2, Text: "Bob"},
	}
	observations := []db.Observation{
		{ID: 1, EntityText: "Alice", Text: "Likes coffee", Timestamp: "2024-01-15 10:30:00"},
	}
	relationships := []db.Relationship{
		{ID: 1, FromText: "Alice", ToText: "Bob", Type: "knows", Timestamp: "2024-01-15 10:30:00"},
	}

	// Without IDs
	output := captureOutput(func() {
		FormatAll(entities, observations, relationships, false)
	})

	if !strings.Contains(output, "Entities (2):") {
		t.Errorf("Expected 'Entities (2):' header, got '%s'", output)
	}
	if !strings.Contains(output, "Observations (1):") {
		t.Errorf("Expected 'Observations (1):' header, got '%s'", output)
	}
	if !strings.Contains(output, "Relationships (1):") {
		t.Errorf("Expected 'Relationships (1):' header, got '%s'", output)
	}
	if !strings.Contains(output, "Alice") {
		t.Errorf("Expected 'Alice' in output, got '%s'", output)
	}
	if strings.Contains(output, "[1]") {
		t.Errorf("Should not contain IDs, got '%s'", output)
	}

	// With IDs
	output = captureOutput(func() {
		FormatAll(entities, observations, relationships, true)
	})

	if !strings.Contains(output, "[1]") {
		t.Errorf("Expected IDs in output, got '%s'", output)
	}
}
