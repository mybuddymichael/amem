package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"amem/config"
	"amem/db"
)

// testEnv holds test environment paths and state
type testEnv struct {
	t          *testing.T
	homeDir    string
	workDir    string
	dbPath     string
	configPath string
	key        string
}

// setupTestEnv creates a test environment with temporary directories
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	homeDir := t.TempDir()
	workDir := t.TempDir()
	key := "testkey123456789012"

	env := &testEnv{
		t:       t,
		homeDir: homeDir,
		workDir: workDir,
		key:     key,
	}

	// Set environment variables to use temp home and override keyring
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
	t.Setenv("AMEM_ENCRYPTION_KEY", key)

	return env
}

// setupTestDB initializes a database and config without using the init command
// This avoids keyring operations that cause macOS popups during tests
func (e *testEnv) setupTestDB(globalConfig bool) error {
	e.t.Helper()

	// Create database path
	if globalConfig {
		e.dbPath = filepath.Join(e.homeDir, "amem.db")
		var err error
		e.configPath, err = config.GlobalPath()
		if err != nil {
			return err
		}
	} else {
		e.dbPath = filepath.Join(e.workDir, "amem.db")
		e.configPath = config.LocalPath(e.workDir)
	}

	// Initialize database directly
	database, err := db.Init(e.dbPath, e.key)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	if err := database.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	// Write config file
	cfg := &config.Config{
		DBPath: e.dbPath,
	}
	if err := config.Write(e.configPath, cfg); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// runCLI runs the CLI command with given args and returns stdout, stderr, and error
func (e *testEnv) runCLI(args ...string) (stdout, stderr string, err error) {
	e.t.Helper()

	// Change to work directory
	oldWd, err := os.Getwd()
	if err != nil {
		e.t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	if err := os.Chdir(e.workDir); err != nil {
		e.t.Fatalf("Failed to change to work directory: %v", err)
	}

	// Capture stdout and stderr
	var outBuf, errBuf bytes.Buffer
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	// Create pipes for stdout and stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	// Read output in goroutines
	outDone := make(chan struct{})
	errDone := make(chan struct{})
	go func() {
		_, _ = outBuf.ReadFrom(rOut)
		close(outDone)
	}()
	go func() {
		_, _ = errBuf.ReadFrom(rErr)
		close(errDone)
	}()

	// Run command
	cmd := buildCommand()
	cmdErr := cmd.Run(context.Background(), append([]string{"amem"}, args...))

	// If there's an error, format it like main() does
	if cmdErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", cmdErr)
	}

	// Close write ends to signal EOF
	_ = wOut.Close()
	_ = wErr.Close()

	// Wait for readers to finish
	<-outDone
	<-errDone

	return outBuf.String(), errBuf.String(), cmdErr
}

// TestInit tests the init command
// Note: These tests are skipped to avoid macOS keychain popups
// The init command tries to store keys in the OS keychain which requires user interaction
// To run these tests, remove the Skip call below
func TestInit(t *testing.T) {
	t.Skip("Skipping init tests that require keychain access")

	t.Run("init fails when database already exists", func(t *testing.T) {
		env := setupTestEnv(t)
		dbPath := filepath.Join(env.homeDir, "test.db")

		// First init should succeed
		_, _, err := env.runCLI("init", "--db-path", dbPath, "--encryption-key", env.key)
		if err != nil {
			t.Fatalf("First init command failed: %v", err)
		}

		// Second init should fail
		_, stderr, err := env.runCLI("init", "--db-path", dbPath, "--encryption-key", env.key)
		if err == nil {
			t.Error("Expected init to fail when database exists")
		}
		if !strings.Contains(stderr, "database already exists") {
			t.Errorf("Expected error about existing database, got: %s", stderr)
		}
	})

	t.Run("init fails with both global and local flags", func(t *testing.T) {
		env := setupTestEnv(t)
		dbPath := filepath.Join(env.homeDir, "test.db")

		_, stderr, err := env.runCLI("init", "--db-path", dbPath, "--encryption-key", env.key, "--global", "--local")
		if err == nil {
			t.Error("Expected init to fail with both --global and --local")
		}
		if !strings.Contains(stderr, "cannot specify both") {
			t.Errorf("Expected error about conflicting flags, got: %s", stderr)
		}
	})
}

// TestCheck tests the check command
func TestCheck(t *testing.T) {
	env := setupTestEnv(t)

	// Initialize database without using init command (avoids keyring)
	if err := env.setupTestDB(true); err != nil {
		t.Fatalf("setupTestDB failed: %v", err)
	}

	// Run check command
	stdout, _, err := env.runCLI("check")
	if err != nil {
		t.Fatalf("check command failed: %v", err)
	}

	// Verify output contains expected checks
	expectedChecks := []string{
		"✓ Config loaded",
		"✓ Database path",
		"✓ Database file exists",
		"✓ Encryption key valid",
		"Database contents:",
		"Entities: 0",
		"Observations: 0",
		"Relationships: 0",
	}

	for _, check := range expectedChecks {
		if !strings.Contains(stdout, check) {
			t.Errorf("Expected check output to contain '%s', got: %s", check, stdout)
		}
	}
}

// TestAddAndSearch tests add and search commands
func TestAddAndSearch(t *testing.T) {
	env := setupTestEnv(t)

	// Initialize database
	if err := env.setupTestDB(true); err != nil {
		t.Fatalf("setupTestDB failed: %v", err)
	}

	t.Run("add and search entities", func(t *testing.T) {
		// Add entities
		stdout, _, err := env.runCLI("add", "entity", "Alice", "Bob", "Charlie")
		if err != nil {
			t.Fatalf("add entity command failed: %v", err)
		}
		if !strings.Contains(stdout, "Added entity: Alice") {
			t.Errorf("Expected success message for Alice, got: %s", stdout)
		}

		// Search for entities
		stdout, _, err = env.runCLI("search", "entities", "Alice")
		if err != nil {
			t.Fatalf("search entities command failed: %v", err)
		}
		if !strings.Contains(stdout, "Alice") {
			t.Errorf("Expected Alice in search results, got: %s", stdout)
		}
		if strings.Contains(stdout, "Charlie") {
			t.Errorf("Did not expect Charlie in search results for Alice, got: %s", stdout)
		}
	})

	t.Run("add and search observations", func(t *testing.T) {
		// Add observation
		stdout, _, err := env.runCLI("add", "observation", "--entity", "Alice", "--text", "Works on Go projects")
		if err != nil {
			t.Fatalf("add observation command failed: %v", err)
		}
		if !strings.Contains(stdout, "Added observation about 'Alice'") {
			t.Errorf("Expected success message, got: %s", stdout)
		}

		// Search observations
		stdout, _, err = env.runCLI("search", "observations", "--about", "Alice")
		if err != nil {
			t.Fatalf("search observations command failed: %v", err)
		}
		if !strings.Contains(stdout, "Works on Go projects") {
			t.Errorf("Expected observation in search results, got: %s", stdout)
		}
	})

	t.Run("add and search relationships", func(t *testing.T) {
		// Add relationship
		stdout, _, err := env.runCLI("add", "relationship", "--from", "Alice", "--to", "Bob", "--type", "knows")
		if err != nil {
			t.Fatalf("add relationship command failed: %v", err)
		}
		if !strings.Contains(stdout, "Added relationship") {
			t.Errorf("Expected success message, got: %s", stdout)
		}

		// Search relationships
		stdout, _, err = env.runCLI("search", "relationships", "--from", "Alice")
		if err != nil {
			t.Fatalf("search relationships command failed: %v", err)
		}
		if !strings.Contains(stdout, "Alice") || !strings.Contains(stdout, "knows") || !strings.Contains(stdout, "Bob") {
			t.Errorf("Expected relationship in search results, got: %s", stdout)
		}
	})

	t.Run("search all with keywords", func(t *testing.T) {
		stdout, _, err := env.runCLI("search", "Alice")
		if err != nil {
			t.Fatalf("search all command failed: %v", err)
		}

		// Should find entity, observation, and relationship
		if !strings.Contains(stdout, "Entities") {
			t.Errorf("Expected Entities section, got: %s", stdout)
		}
		if !strings.Contains(stdout, "Observations") {
			t.Errorf("Expected Observations section, got: %s", stdout)
		}
		if !strings.Contains(stdout, "Relationships") {
			t.Errorf("Expected Relationships section, got: %s", stdout)
		}
	})

	t.Run("search with --with-ids flag", func(t *testing.T) {
		stdout, _, err := env.runCLI("search", "entities", "--with-ids", "Alice")
		if err != nil {
			t.Fatalf("search with --with-ids failed: %v", err)
		}

		// Should contain ID in brackets
		if !strings.Contains(stdout, "[") || !strings.Contains(stdout, "]") {
			t.Errorf("Expected IDs in brackets, got: %s", stdout)
		}
	})

	t.Run("search entities with --any flag (OR logic)", func(t *testing.T) {
		// Add more entities for testing
		_, _, _ = env.runCLI("add", "entity", "Python", "Golang")

		// Search with --any (should find entities matching ANY keyword)
		stdout, _, err := env.runCLI("search", "entities", "--any", "Alice", "Python")
		if err != nil {
			t.Fatalf("search with --any failed: %v", err)
		}
		if !strings.Contains(stdout, "Alice") || !strings.Contains(stdout, "Python") {
			t.Errorf("Expected both Alice and Python with --any, got: %s", stdout)
		}
	})

	t.Run("search entities with --all flag (AND logic)", func(t *testing.T) {
		// Add entity with multiple keywords
		_, _, _ = env.runCLI("add", "entity", "Alice Smith")

		// Search with --all (should only find entities matching ALL keywords)
		stdout, _, err := env.runCLI("search", "entities", "--all", "Alice", "Smith")
		if err != nil {
			t.Fatalf("search with --all failed: %v", err)
		}
		if !strings.Contains(stdout, "Alice Smith") {
			t.Errorf("Expected 'Alice Smith' with --all, got: %s", stdout)
		}

		// Should not find just "Alice" when both keywords required
		if strings.Contains(stdout, "Alice\n") && !strings.Contains(stdout, "Smith") {
			t.Errorf("Should not find 'Alice' alone with --all, got: %s", stdout)
		}
	})

	t.Run("search observations with --any flag", func(t *testing.T) {
		// Add more observations
		_, _, _ = env.runCLI("add", "observation", "--entity", "Alice", "--text", "Likes Python programming")

		stdout, _, err := env.runCLI("search", "observations", "--any", "Go", "Python")
		if err != nil {
			t.Fatalf("search observations with --any failed: %v", err)
		}
		if !strings.Contains(stdout, "Go projects") || !strings.Contains(stdout, "Python programming") {
			t.Errorf("Expected observations with Go or Python, got: %s", stdout)
		}
	})

	t.Run("search observations with --all flag", func(t *testing.T) {
		// Add observation with multiple keywords
		_, _, _ = env.runCLI("add", "observation", "--entity", "Alice", "--text", "Loves Go and Python both")

		stdout, _, err := env.runCLI("search", "observations", "--all", "Go", "Python")
		if err != nil {
			t.Fatalf("search observations with --all failed: %v", err)
		}
		if !strings.Contains(stdout, "Loves Go and Python both") {
			t.Errorf("Expected observation with both keywords, got: %s", stdout)
		}
	})

	t.Run("search relationships with --any flag", func(t *testing.T) {
		// Add more relationships
		_, _, _ = env.runCLI("add", "relationship", "--from", "Alice", "--to", "Python", "--type", "uses")

		stdout, _, err := env.runCLI("search", "relationships", "--any", "knows", "uses")
		if err != nil {
			t.Fatalf("search relationships with --any failed: %v", err)
		}
		if !strings.Contains(stdout, "knows") && !strings.Contains(stdout, "uses") {
			t.Errorf("Expected relationships with knows or uses, got: %s", stdout)
		}
	})

	t.Run("search relationships with --all flag", func(t *testing.T) {
		stdout, _, err := env.runCLI("search", "relationships", "--all", "Alice", "uses")
		if err != nil {
			t.Fatalf("search relationships with --all failed: %v", err)
		}
		if !strings.Contains(stdout, "Alice") || !strings.Contains(stdout, "uses") {
			t.Errorf("Expected relationships with both Alice and uses, got: %s", stdout)
		}
	})

	t.Run("search fails with both --any and --all", func(t *testing.T) {
		_, stderr, err := env.runCLI("search", "entities", "--any", "--all", "Alice")
		if err == nil {
			t.Error("Expected search to fail with both --any and --all")
		}
		if !strings.Contains(stderr, "cannot specify both") {
			t.Errorf("Expected error about conflicting flags, got: %s", stderr)
		}
	})

	t.Run("top-level search with --any flag", func(t *testing.T) {
		stdout, _, err := env.runCLI("search", "--any", "Alice", "Python")
		if err != nil {
			t.Fatalf("top-level search with --any failed: %v", err)
		}
		// Should find results across all categories
		if !strings.Contains(stdout, "Entities") || !strings.Contains(stdout, "Observations") {
			t.Errorf("Expected multiple categories in results, got: %s", stdout)
		}
	})

	t.Run("top-level search with --all flag", func(t *testing.T) {
		stdout, _, err := env.runCLI("search", "--all", "Alice", "Go")
		if err != nil {
			t.Fatalf("top-level search with --all failed: %v", err)
		}
		// Should only find items containing both keywords
		if !strings.Contains(stdout, "Alice") || !strings.Contains(stdout, "Go") {
			t.Errorf("Expected results with both keywords, got: %s", stdout)
		}
	})
}

// TestDelete tests delete commands
func TestDelete(t *testing.T) {
	env := setupTestEnv(t)

	// Initialize database
	if err := env.setupTestDB(true); err != nil {
		t.Fatalf("setupTestDB failed: %v", err)
	}

	t.Run("delete entity by name", func(t *testing.T) {
		// Add entity
		_, _, err := env.runCLI("add", "entity", "TestEntity")
		if err != nil {
			t.Fatalf("add entity command failed: %v", err)
		}

		// Delete entity
		stdout, _, err := env.runCLI("delete", "entity", "TestEntity")
		if err != nil {
			t.Fatalf("delete entity command failed: %v", err)
		}
		if !strings.Contains(stdout, "Deleted entity: TestEntity") {
			t.Errorf("Expected success message, got: %s", stdout)
		}

		// Verify entity is gone
		stdout, _, _ = env.runCLI("search", "entities", "TestEntity")
		if strings.Contains(stdout, "TestEntity") {
			t.Errorf("Entity should be deleted, but found in search: %s", stdout)
		}
	})

	t.Run("delete entity by ID", func(t *testing.T) {
		// Add entity and get its ID
		_, _, err := env.runCLI("add", "entity", "ToDelete")
		if err != nil {
			t.Fatalf("add entity command failed: %v", err)
		}

		// Get ID from search
		stdout, _, _ := env.runCLI("search", "entities", "--with-ids", "ToDelete")
		if !strings.Contains(stdout, "[") {
			t.Fatalf("Could not find ID in search output: %s", stdout)
		}

		// Extract ID by finding [number]
		idStart := strings.Index(stdout, "[")
		idEnd := strings.Index(stdout, "]")
		if idStart == -1 || idEnd == -1 {
			t.Fatalf("Could not find ID brackets in output: %s", stdout)
		}
		var id int
		if _, err := fmt.Sscanf(stdout[idStart:idEnd+1], "[%d]", &id); err != nil {
			t.Fatalf("Could not parse ID from output: %s", stdout)
		}

		// Delete by ID
		deleteStdout, _, err := env.runCLI("delete", "entity", "--ids", fmt.Sprintf("%d", id))
		if err != nil {
			t.Fatalf("delete entity by ID failed: %v", err)
		}
		if !strings.Contains(deleteStdout, fmt.Sprintf("Deleted entity ID %d", id)) {
			t.Errorf("Expected success message, got: %s", deleteStdout)
		}
	})

	t.Run("delete observation by ID", func(t *testing.T) {
		// Add entity and observation
		_, _, _ = env.runCLI("add", "entity", "EntityForObs")
		_, _, _ = env.runCLI("add", "observation", "--entity", "EntityForObs", "--text", "Test observation")

		// Get observation ID
		stdout, _, _ := env.runCLI("search", "observations", "--with-ids", "--about", "EntityForObs")

		// Extract ID by finding [number]
		idStart := strings.Index(stdout, "[")
		idEnd := strings.Index(stdout, "]")
		if idStart == -1 || idEnd == -1 {
			t.Fatalf("Could not find ID brackets in output: %s", stdout)
		}
		var id int
		if _, err := fmt.Sscanf(stdout[idStart:idEnd+1], "[%d]", &id); err != nil {
			t.Fatalf("Could not parse observation ID: %s", stdout)
		}

		// Delete observation
		deleteStdout, _, err := env.runCLI("delete", "observation", "--ids", fmt.Sprintf("%d", id))
		if err != nil {
			t.Fatalf("delete observation failed: %v", err)
		}
		if !strings.Contains(deleteStdout, fmt.Sprintf("Deleted observation ID %d", id)) {
			t.Errorf("Expected success message, got: %s", deleteStdout)
		}
	})

	t.Run("delete relationship by ID", func(t *testing.T) {
		// Add entities and relationship
		_, _, _ = env.runCLI("add", "entity", "Person1", "Person2")
		_, _, _ = env.runCLI("add", "relationship", "--from", "Person1", "--to", "Person2", "--type", "knows")

		// Get relationship ID
		stdout, _, _ := env.runCLI("search", "relationships", "--with-ids", "--from", "Person1")

		// Extract ID by finding [number]
		idStart := strings.Index(stdout, "[")
		idEnd := strings.Index(stdout, "]")
		if idStart == -1 || idEnd == -1 {
			t.Fatalf("Could not find ID brackets in output: %s", stdout)
		}
		var id int
		if _, err := fmt.Sscanf(stdout[idStart:idEnd+1], "[%d]", &id); err != nil {
			t.Fatalf("Could not parse relationship ID: %s", stdout)
		}

		// Delete relationship
		deleteStdout, _, err := env.runCLI("delete", "relationship", "--ids", fmt.Sprintf("%d", id))
		if err != nil {
			t.Fatalf("delete relationship failed: %v", err)
		}
		if !strings.Contains(deleteStdout, fmt.Sprintf("Deleted relationship ID %d", id)) {
			t.Errorf("Expected success message, got: %s", deleteStdout)
		}
	})

	t.Run("delete entity fails with both name and IDs", func(t *testing.T) {
		_, stderr, err := env.runCLI("delete", "entity", "SomeName", "--ids", "1")
		if err == nil {
			t.Error("Expected delete to fail with both name and IDs")
		}
		if !strings.Contains(stderr, "cannot specify both") {
			t.Errorf("Expected error about conflicting arguments, got: %s", stderr)
		}
	})
}

// TestEdit tests edit commands
func TestEdit(t *testing.T) {
	env := setupTestEnv(t)

	// Initialize database
	if err := env.setupTestDB(true); err != nil {
		t.Fatalf("setupTestDB failed: %v", err)
	}

	t.Run("edit entity name", func(t *testing.T) {
		// Add entity
		_, _, err := env.runCLI("add", "entity", "OldName")
		if err != nil {
			t.Fatalf("add entity command failed: %v", err)
		}

		// Edit entity
		stdout, _, err := env.runCLI("edit", "entity", "OldName", "--new-name", "NewName")
		if err != nil {
			t.Fatalf("edit entity command failed: %v", err)
		}
		if !strings.Contains(stdout, "Updated entity 'OldName' to 'NewName'") {
			t.Errorf("Expected success message, got: %s", stdout)
		}

		// Verify old name is gone and new name exists
		searchOut, _, _ := env.runCLI("search", "entities", "NewName")
		if !strings.Contains(searchOut, "NewName") {
			t.Errorf("Expected to find NewName, got: %s", searchOut)
		}

		searchOut, _, _ = env.runCLI("search", "entities", "OldName")
		if strings.Contains(searchOut, "OldName") {
			t.Errorf("Did not expect to find OldName, got: %s", searchOut)
		}
	})

	t.Run("edit observation text", func(t *testing.T) {
		// Add entity and observation
		_, _, _ = env.runCLI("add", "entity", "TestPerson")
		_, _, _ = env.runCLI("add", "observation", "--entity", "TestPerson", "--text", "Old text")

		// Get observation ID
		stdout, _, _ := env.runCLI("search", "observations", "--with-ids", "--about", "TestPerson")

		// Extract ID by finding [number]
		idStart := strings.Index(stdout, "[")
		idEnd := strings.Index(stdout, "]")
		if idStart == -1 || idEnd == -1 {
			t.Fatalf("Could not find ID brackets in output: %s", stdout)
		}
		var id int
		if _, err := fmt.Sscanf(stdout[idStart:idEnd+1], "[%d]", &id); err != nil {
			t.Fatalf("Could not parse observation ID: %s", stdout)
		}

		// Edit observation
		editOut, _, err := env.runCLI("edit", "observation", "--id", fmt.Sprintf("%d", id), "--new-text", "New text")
		if err != nil {
			t.Fatalf("edit observation command failed: %v", err)
		}
		if !strings.Contains(editOut, fmt.Sprintf("Updated observation ID %d", id)) {
			t.Errorf("Expected success message, got: %s", editOut)
		}

		// Verify text was updated
		searchOut, _, _ := env.runCLI("search", "observations", "--about", "TestPerson")
		if !strings.Contains(searchOut, "New text") {
			t.Errorf("Expected to find 'New text', got: %s", searchOut)
		}
		if strings.Contains(searchOut, "Old text") {
			t.Errorf("Did not expect to find 'Old text', got: %s", searchOut)
		}
	})

	t.Run("edit observation entity", func(t *testing.T) {
		// Add two entities and an observation for the first
		_, _, _ = env.runCLI("add", "entity", "Entity1", "Entity2")
		_, _, _ = env.runCLI("add", "observation", "--entity", "Entity1", "--text", "Observation text")

		// Get observation ID
		stdout, _, _ := env.runCLI("search", "observations", "--with-ids", "--about", "Entity1")
		idStart := strings.Index(stdout, "[")
		idEnd := strings.Index(stdout, "]")
		if idStart == -1 || idEnd == -1 {
			t.Fatalf("Could not find ID brackets in output: %s", stdout)
		}
		var obsID int
		if _, err := fmt.Sscanf(stdout[idStart:idEnd+1], "[%d]", &obsID); err != nil {
			t.Fatalf("Could not parse observation ID: %s", stdout)
		}

		// Get Entity2 ID
		stdout, _, _ = env.runCLI("search", "entities", "--with-ids", "Entity2")
		idStart = strings.Index(stdout, "[")
		idEnd = strings.Index(stdout, "]")
		if idStart == -1 || idEnd == -1 {
			t.Fatalf("Could not find ID brackets in output: %s", stdout)
		}
		var entity2ID int
		if _, err := fmt.Sscanf(stdout[idStart:idEnd+1], "[%d]", &entity2ID); err != nil {
			t.Fatalf("Could not parse entity ID: %s", stdout)
		}

		// Change observation to point to Entity2
		editOut, _, err := env.runCLI("edit", "observation", "--id", fmt.Sprintf("%d", obsID), "--new-entity-id", fmt.Sprintf("%d", entity2ID))
		if err != nil {
			t.Fatalf("edit observation command failed: %v", err)
		}
		if !strings.Contains(editOut, fmt.Sprintf("Updated observation ID %d", obsID)) {
			t.Errorf("Expected success message, got: %s", editOut)
		}

		// Verify observation now belongs to Entity2
		searchOut, _, _ := env.runCLI("search", "observations", "--about", "Entity2")
		if !strings.Contains(searchOut, "Observation text") {
			t.Errorf("Expected to find observation under Entity2, got: %s", searchOut)
		}

		// Verify observation no longer belongs to Entity1
		searchOut, _, _ = env.runCLI("search", "observations", "--about", "Entity1")
		if strings.Contains(searchOut, "Observation text") {
			t.Errorf("Did not expect to find observation under Entity1, got: %s", searchOut)
		}
	})
}

// TestFullWorkflow tests a complete end-to-end workflow
func TestFullWorkflow(t *testing.T) {
	env := setupTestEnv(t)

	// 1. Initialize database
	if err := env.setupTestDB(true); err != nil {
		t.Fatalf("setupTestDB failed: %v", err)
	}

	// 2. Check database status
	stdout, _, err := env.runCLI("check")
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !strings.Contains(stdout, "Entities: 0") {
		t.Errorf("Expected empty database")
	}

	// 3. Add some entities
	_, _, err = env.runCLI("add", "entity", "Alice", "Bob", "GitHub")
	if err != nil {
		t.Fatalf("add entities failed: %v", err)
	}

	// 4. Add observations
	_, _, err = env.runCLI("add", "observation", "--entity", "Alice", "--text", "Working on amem project")
	if err != nil {
		t.Fatalf("add observation failed: %v", err)
	}

	// 5. Add relationships
	_, _, err = env.runCLI("add", "relationship", "--from", "Alice", "--to", "GitHub", "--type", "uses")
	if err != nil {
		t.Fatalf("add relationship failed: %v", err)
	}

	// 6. Search and verify data
	stdout, _, err = env.runCLI("search", "Alice")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if !strings.Contains(stdout, "Alice") {
		t.Errorf("Expected Alice in search results")
	}
	if !strings.Contains(stdout, "amem project") {
		t.Errorf("Expected observation in search results")
	}
	if !strings.Contains(stdout, "uses") {
		t.Errorf("Expected relationship in search results")
	}

	// 7. Check updated counts
	stdout, _, err = env.runCLI("check")
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !strings.Contains(stdout, "Entities: 3") {
		t.Errorf("Expected 3 entities, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Observations: 1") {
		t.Errorf("Expected 1 observation, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Relationships: 1") {
		t.Errorf("Expected 1 relationship, got: %s", stdout)
	}

	// 8. Edit an entity
	_, _, err = env.runCLI("edit", "entity", "Bob", "--new-name", "Robert")
	if err != nil {
		t.Fatalf("edit entity failed: %v", err)
	}

	// 9. Verify edit
	stdout, _, _ = env.runCLI("search", "entities", "Robert")
	if !strings.Contains(stdout, "Robert") {
		t.Errorf("Expected to find Robert after edit")
	}

	// 10. Delete an entity
	_, _, err = env.runCLI("delete", "entity", "Robert")
	if err != nil {
		t.Fatalf("delete entity failed: %v", err)
	}

	// 11. Verify deletion
	stdout, _, err = env.runCLI("check")
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !strings.Contains(stdout, "Entities: 2") {
		t.Errorf("Expected 2 entities after deletion, got: %s", stdout)
	}
}

// TestErrorCases tests various error scenarios
func TestErrorCases(t *testing.T) {
	t.Run("commands fail without config", func(t *testing.T) {
		env := setupTestEnv(t)

		_, stderr, err := env.runCLI("check")
		if err == nil {
			t.Error("Expected check to fail without config")
		}
		if !strings.Contains(stderr, "config") {
			t.Errorf("Expected error about missing config, got: %s", stderr)
		}
	})

	t.Run("add entity requires at least one name", func(t *testing.T) {
		env := setupTestEnv(t)
		_ = env.setupTestDB(true)

		_, stderr, err := env.runCLI("add", "entity")
		if err == nil {
			t.Error("Expected add entity to fail without names")
		}
		if !strings.Contains(stderr, "required") {
			t.Errorf("Expected error about required argument, got: %s", stderr)
		}
	})

	t.Run("add observation requires entity and text flags", func(t *testing.T) {
		env := setupTestEnv(t)
		_ = env.setupTestDB(true)

		_, stderr, err := env.runCLI("add", "observation", "--entity", "TestEntity")
		if err == nil {
			t.Error("Expected add observation to fail without --text")
		}
		if !strings.Contains(stderr, "required") && !strings.Contains(stderr, "text") {
			t.Errorf("Expected error about missing --text, got: %s", stderr)
		}
	})

	t.Run("edit entity requires new-name flag", func(t *testing.T) {
		env := setupTestEnv(t)
		_ = env.setupTestDB(true)
		_, _, _ = env.runCLI("add", "entity", "TestEntity")

		_, stderr, err := env.runCLI("edit", "entity", "TestEntity")
		if err == nil {
			t.Error("Expected edit entity to fail without --new-name")
		}
		if !strings.Contains(stderr, "new-name") {
			t.Errorf("Expected error about new-name flag, got: %s", stderr)
		}
	})

	t.Run("delete observation requires ids flag", func(t *testing.T) {
		env := setupTestEnv(t)
		_ = env.setupTestDB(true)

		_, stderr, err := env.runCLI("delete", "observation")
		if err == nil {
			t.Error("Expected delete observation to fail without --ids")
		}
		if !strings.Contains(stderr, "ids") {
			t.Errorf("Expected error about ids flag, got: %s", stderr)
		}
	})
}
