package db

import (
	"testing"
)

func TestMigrations(t *testing.T) {
	dbPath := t.TempDir() + "/test_amem_migrations.db"
	key := "testkey123456789012"

	// Initialize database
	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Verify schema_migrations table exists
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		t.Fatalf("schema_migrations table doesn't exist: %v", err)
	}

	// Verify current version is 1
	version, err := getCurrentVersion(db.conn)
	if err != nil {
		t.Fatalf("Failed to get current version: %v", err)
	}
	if version != 1 {
		t.Errorf("Expected version 1, got %d", version)
	}

	// Verify tables exist
	tables := []string{"entities", "observations", "relationships"}
	for _, table := range tables {
		err = db.conn.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
		if err != nil {
			t.Errorf("Table %s doesn't exist: %v", table, err)
		}
	}

	// Verify indices exist
	var indexCount int
	err = db.conn.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='index' AND name IN (
			'idx_observations_entity',
			'idx_relationships_from',
			'idx_relationships_to',
			'idx_relationships_type'
		)
	`).Scan(&indexCount)
	if err != nil {
		t.Fatalf("Failed to query indices: %v", err)
	}
	if indexCount != 4 {
		t.Errorf("Expected 4 indices, got %d", indexCount)
	}
}

func TestMigrationsIdempotent(t *testing.T) {
	dbPath := t.TempDir() + "/test_amem_idempotent.db"
	key := "testkey123456789012"

	// Initialize database twice
	db1, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("First init failed: %v", err)
	}
	db1.Close()

	db2, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Second init failed: %v", err)
	}
	defer db2.Close()

	// Verify version is still 1
	version, err := getCurrentVersion(db2.conn)
	if err != nil {
		t.Fatalf("Failed to get current version: %v", err)
	}
	if version != 1 {
		t.Errorf("Expected version 1 after re-init, got %d", version)
	}

	// Verify migration was only applied once
	var migrationCount int
	err = db2.conn.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&migrationCount)
	if err != nil {
		t.Fatalf("Failed to count migrations: %v", err)
	}
	if migrationCount != 1 {
		t.Errorf("Expected 1 migration record, got %d", migrationCount)
	}
}
