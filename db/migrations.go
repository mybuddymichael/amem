package db

import (
	"database/sql"
	"fmt"
	"sort"
)

// Migration represents a database schema migration
type Migration struct {
	Version int
	Up      string
	Down    string
}

// migrations contains all schema migrations in order
var migrations = []Migration{
	{
		Version: 1,
		Up: `
CREATE TABLE entities (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	text TEXT NOT NULL UNIQUE
);

CREATE TABLE observations (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	entity_id INTEGER NOT NULL,
	text TEXT NOT NULL,
	timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (entity_id) REFERENCES entities(id) ON DELETE CASCADE
);

CREATE TABLE relationships (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	from_id INTEGER NOT NULL,
	to_id INTEGER NOT NULL,
	type TEXT NOT NULL,
	timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (from_id) REFERENCES entities(id) ON DELETE CASCADE,
	FOREIGN KEY (to_id) REFERENCES entities(id) ON DELETE CASCADE
);

CREATE INDEX idx_observations_entity ON observations(entity_id);
CREATE INDEX idx_relationships_from ON relationships(from_id);
CREATE INDEX idx_relationships_to ON relationships(to_id);
CREATE INDEX idx_relationships_type ON relationships(type);
`,
		Down: `
DROP INDEX IF EXISTS idx_relationships_type;
DROP INDEX IF EXISTS idx_relationships_to;
DROP INDEX IF EXISTS idx_relationships_from;
DROP INDEX IF EXISTS idx_observations_entity;
DROP TABLE IF EXISTS relationships;
DROP TABLE IF EXISTS observations;
DROP TABLE IF EXISTS entities;
`,
	},
}

const schemaVersionsTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version INTEGER PRIMARY KEY,
	applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

// getCurrentVersion returns the current schema version
func getCurrentVersion(conn *sql.DB) (int, error) {
	var version int
	err := conn.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}

// migrate applies all pending migrations
func migrate(conn *sql.DB) error {
	// Create schema_migrations table
	if _, err := conn.Exec(schemaVersionsTable); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// Get current version
	currentVersion, err := getCurrentVersion(conn)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	// Apply pending migrations
	for _, m := range migrations {
		if m.Version <= currentVersion {
			continue
		}

		tx, err := conn.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %d: %w", m.Version, err)
		}

		// Execute migration
		if _, err := tx.Exec(m.Up); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to apply migration %d: %w", m.Version, err)
		}

		// Record migration
		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", m.Version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", m.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", m.Version, err)
		}
	}

	return nil
}
