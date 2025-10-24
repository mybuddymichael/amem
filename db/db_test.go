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
	defer func() { _ = db.Close() }()

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
	if err := db1.Close(); err != nil {
		t.Fatalf("Failed to close db1: %v", err)
	}

	db2, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Second init failed: %v", err)
	}
	defer func() { _ = db2.Close() }()

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

func TestAddEntity(t *testing.T) {
	dbPath := t.TempDir() + "/test_add_entity.db"
	key := "testkey123456789012"

	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Add new entity
	id1, err := db.AddEntity("Alice")
	if err != nil {
		t.Fatalf("Failed to add entity: %v", err)
	}
	if id1 == 0 {
		t.Error("Expected non-zero ID for new entity")
	}

	// Verify entity exists
	var text string
	err = db.conn.QueryRow("SELECT text FROM entities WHERE id = ?", id1).Scan(&text)
	if err != nil {
		t.Fatalf("Failed to query entity: %v", err)
	}
	if text != "Alice" {
		t.Errorf("Expected 'Alice', got '%s'", text)
	}

	// Add duplicate entity - should return existing ID
	id2, err := db.AddEntity("Alice")
	if err != nil {
		t.Fatalf("Failed to add duplicate entity: %v", err)
	}
	if id2 != id1 {
		t.Errorf("Expected same ID for duplicate entity: got %d, want %d", id2, id1)
	}

	// Verify only one entity exists
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM entities").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count entities: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 entity, got %d", count)
	}
}

func TestAddObservation(t *testing.T) {
	dbPath := t.TempDir() + "/test_add_observation.db"
	key := "testkey123456789012"

	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Add observation (entity doesn't exist yet)
	obsID, err := db.AddObservation("Bob", "Likes coffee")
	if err != nil {
		t.Fatalf("Failed to add observation: %v", err)
	}
	if obsID == 0 {
		t.Error("Expected non-zero ID for observation")
	}

	// Verify entity was auto-created
	var entityID int64
	var entityText string
	err = db.conn.QueryRow("SELECT id, text FROM entities WHERE text = 'Bob'").Scan(&entityID, &entityText)
	if err != nil {
		t.Fatalf("Entity was not auto-created: %v", err)
	}

	// Verify observation exists and links to entity
	var observationText string
	var linkedEntityID int64
	err = db.conn.QueryRow("SELECT text, entity_id FROM observations WHERE id = ?", obsID).Scan(&observationText, &linkedEntityID)
	if err != nil {
		t.Fatalf("Failed to query observation: %v", err)
	}
	if observationText != "Likes coffee" {
		t.Errorf("Expected 'Likes coffee', got '%s'", observationText)
	}
	if linkedEntityID != entityID {
		t.Errorf("Observation linked to wrong entity: got %d, want %d", linkedEntityID, entityID)
	}

	// Add another observation for same entity
	obsID2, err := db.AddObservation("Bob", "Works remotely")
	if err != nil {
		t.Fatalf("Failed to add second observation: %v", err)
	}

	// Verify still only one entity
	var entityCount int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM entities").Scan(&entityCount)
	if err != nil {
		t.Fatalf("Failed to count entities: %v", err)
	}
	if entityCount != 1 {
		t.Errorf("Expected 1 entity, got %d", entityCount)
	}

	// Verify two observations
	var obsCount int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM observations WHERE entity_id = ?", entityID).Scan(&obsCount)
	if err != nil {
		t.Fatalf("Failed to count observations: %v", err)
	}
	if obsCount != 2 {
		t.Errorf("Expected 2 observations, got %d", obsCount)
	}

	// Verify timestamp is set
	var timestamp string
	err = db.conn.QueryRow("SELECT timestamp FROM observations WHERE id = ?", obsID2).Scan(&timestamp)
	if err != nil {
		t.Fatalf("Failed to query timestamp: %v", err)
	}
	if timestamp == "" {
		t.Error("Expected timestamp to be set")
	}
}

func TestAddRelationship(t *testing.T) {
	dbPath := t.TempDir() + "/test_add_relationship.db"
	key := "testkey123456789012"

	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Add relationship (neither entity exists yet)
	relID, err := db.AddRelationship("Charlie", "Project X", "works_on")
	if err != nil {
		t.Fatalf("Failed to add relationship: %v", err)
	}
	if relID == 0 {
		t.Error("Expected non-zero ID for relationship")
	}

	// Verify both entities were auto-created
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM entities WHERE text IN ('Charlie', 'Project X')").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count entities: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 entities, got %d", count)
	}

	// Verify relationship exists and links correctly
	var fromID, toID int64
	var relType string
	err = db.conn.QueryRow("SELECT from_id, to_id, type FROM relationships WHERE id = ?", relID).Scan(&fromID, &toID, &relType)
	if err != nil {
		t.Fatalf("Failed to query relationship: %v", err)
	}
	if relType != "works_on" {
		t.Errorf("Expected 'works_on', got '%s'", relType)
	}

	// Verify fromID and toID point to correct entities
	var fromText, toText string
	err = db.conn.QueryRow("SELECT text FROM entities WHERE id = ?", fromID).Scan(&fromText)
	if err != nil {
		t.Fatalf("Failed to query from entity: %v", err)
	}
	if fromText != "Charlie" {
		t.Errorf("Expected 'Charlie', got '%s'", fromText)
	}

	err = db.conn.QueryRow("SELECT text FROM entities WHERE id = ?", toID).Scan(&toText)
	if err != nil {
		t.Fatalf("Failed to query to entity: %v", err)
	}
	if toText != "Project X" {
		t.Errorf("Expected 'Project X', got '%s'", toText)
	}

	// Add another relationship with existing entity
	relID2, err := db.AddRelationship("Charlie", "Diana", "manages")
	if err != nil {
		t.Fatalf("Failed to add second relationship: %v", err)
	}

	// Verify only 3 entities now (Charlie, Project X, Diana)
	err = db.conn.QueryRow("SELECT COUNT(*) FROM entities").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count entities: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected 3 entities, got %d", count)
	}

	// Verify timestamp is set
	var timestamp string
	err = db.conn.QueryRow("SELECT timestamp FROM relationships WHERE id = ?", relID2).Scan(&timestamp)
	if err != nil {
		t.Fatalf("Failed to query timestamp: %v", err)
	}
	if timestamp == "" {
		t.Error("Expected timestamp to be set")
	}
}

func TestAddRelationshipSelfReference(t *testing.T) {
	dbPath := t.TempDir() + "/test_self_reference.db"
	key := "testkey123456789012"

	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Add self-referencing relationship
	relID, err := db.AddRelationship("Eve", "Eve", "reports_to")
	if err != nil {
		t.Fatalf("Failed to add self-referencing relationship: %v", err)
	}

	// Verify only one entity was created
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM entities").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count entities: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 entity, got %d", count)
	}

	// Verify from_id and to_id are the same
	var fromID, toID int64
	err = db.conn.QueryRow("SELECT from_id, to_id FROM relationships WHERE id = ?", relID).Scan(&fromID, &toID)
	if err != nil {
		t.Fatalf("Failed to query relationship: %v", err)
	}
	if fromID != toID {
		t.Errorf("Expected from_id and to_id to be equal, got from=%d, to=%d", fromID, toID)
	}
}

func TestCascadeDelete(t *testing.T) {
	dbPath := t.TempDir() + "/test_cascade.db"
	key := "testkey123456789012"

	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Create entity with observations and relationships
	entityID, err := db.AddEntity("Frank")
	if err != nil {
		t.Fatalf("Failed to add entity: %v", err)
	}

	_, err = db.AddObservation("Frank", "First observation")
	if err != nil {
		t.Fatalf("Failed to add observation: %v", err)
	}

	_, err = db.AddObservation("Frank", "Second observation")
	if err != nil {
		t.Fatalf("Failed to add observation: %v", err)
	}

	_, err = db.AddRelationship("Frank", "Grace", "knows")
	if err != nil {
		t.Fatalf("Failed to add relationship: %v", err)
	}

	_, err = db.AddRelationship("Grace", "Frank", "knows")
	if err != nil {
		t.Fatalf("Failed to add relationship: %v", err)
	}

	// Delete the entity
	_, err = db.conn.Exec("DELETE FROM entities WHERE id = ?", entityID)
	if err != nil {
		t.Fatalf("Failed to delete entity: %v", err)
	}

	// Verify observations were cascaded
	var obsCount int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM observations WHERE entity_id = ?", entityID).Scan(&obsCount)
	if err != nil {
		t.Fatalf("Failed to count observations: %v", err)
	}
	if obsCount != 0 {
		t.Errorf("Expected 0 observations after cascade delete, got %d", obsCount)
	}

	// Verify relationships were cascaded
	var relCount int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM relationships WHERE from_id = ? OR to_id = ?", entityID, entityID).Scan(&relCount)
	if err != nil {
		t.Fatalf("Failed to count relationships: %v", err)
	}
	if relCount != 0 {
		t.Errorf("Expected 0 relationships after cascade delete, got %d", relCount)
	}

	// Verify Grace still exists
	var graceExists bool
	err = db.conn.QueryRow("SELECT EXISTS(SELECT 1 FROM entities WHERE text = 'Grace')").Scan(&graceExists)
	if err != nil {
		t.Fatalf("Failed to check Grace existence: %v", err)
	}
	if !graceExists {
		t.Error("Grace should still exist after Frank was deleted")
	}
}

func TestDeleteEntity(t *testing.T) {
	dbPath := t.TempDir() + "/test_delete_entity.db"
	key := "testkey123456789012"

	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Add test entity
	id, err := db.AddEntity("TestEntity")
	if err != nil {
		t.Fatalf("Failed to add entity: %v", err)
	}

	// Delete by ID
	err = db.DeleteEntity(id)
	if err != nil {
		t.Fatalf("Failed to delete entity: %v", err)
	}

	// Verify entity was deleted
	entities, err := db.SearchEntities([]string{"TestEntity"})
	if err != nil {
		t.Fatalf("Failed to search entities: %v", err)
	}
	if len(entities) != 0 {
		t.Errorf("Expected entity to be deleted, but found %d entities", len(entities))
	}

	// Try to delete non-existent entity
	err = db.DeleteEntity(99999)
	if err == nil {
		t.Error("Expected error when deleting non-existent entity")
	}
}

func TestDeleteEntityByText(t *testing.T) {
	dbPath := t.TempDir() + "/test_delete_entity_text.db"
	key := "testkey123456789012"

	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Add test entity
	_, err = db.AddEntity("TestEntity")
	if err != nil {
		t.Fatalf("Failed to add entity: %v", err)
	}

	// Delete by text
	err = db.DeleteEntityByText("TestEntity")
	if err != nil {
		t.Fatalf("Failed to delete entity: %v", err)
	}

	// Verify entity was deleted
	entities, err := db.SearchEntities([]string{"TestEntity"})
	if err != nil {
		t.Fatalf("Failed to search entities: %v", err)
	}
	if len(entities) != 0 {
		t.Errorf("Expected entity to be deleted, but found %d entities", len(entities))
	}

	// Try to delete non-existent entity
	err = db.DeleteEntityByText("NonExistent")
	if err == nil {
		t.Error("Expected error when deleting non-existent entity")
	}
}

func TestDeleteObservation(t *testing.T) {
	dbPath := t.TempDir() + "/test_delete_observation.db"
	key := "testkey123456789012"

	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Add test observation
	obsID, err := db.AddObservation("TestEntity", "Test observation")
	if err != nil {
		t.Fatalf("Failed to add observation: %v", err)
	}

	// Delete observation
	err = db.DeleteObservation(obsID)
	if err != nil {
		t.Fatalf("Failed to delete observation: %v", err)
	}

	// Verify observation was deleted
	observations, err := db.SearchObservations("", []string{"Test observation"})
	if err != nil {
		t.Fatalf("Failed to search observations: %v", err)
	}
	if len(observations) != 0 {
		t.Errorf("Expected observation to be deleted, but found %d observations", len(observations))
	}

	// Try to delete non-existent observation
	err = db.DeleteObservation(99999)
	if err == nil {
		t.Error("Expected error when deleting non-existent observation")
	}
}

func TestDeleteRelationship(t *testing.T) {
	dbPath := t.TempDir() + "/test_delete_relationship.db"
	key := "testkey123456789012"

	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Add test relationship
	relID, err := db.AddRelationship("Entity1", "Entity2", "knows")
	if err != nil {
		t.Fatalf("Failed to add relationship: %v", err)
	}

	// Delete relationship
	err = db.DeleteRelationship(relID)
	if err != nil {
		t.Fatalf("Failed to delete relationship: %v", err)
	}

	// Verify relationship was deleted
	relationships, err := db.SearchRelationships("", "", "", []string{"knows"})
	if err != nil {
		t.Fatalf("Failed to search relationships: %v", err)
	}
	if len(relationships) != 0 {
		t.Errorf("Expected relationship to be deleted, but found %d relationships", len(relationships))
	}

	// Try to delete non-existent relationship
	err = db.DeleteRelationship(99999)
	if err == nil {
		t.Error("Expected error when deleting non-existent relationship")
	}
}

func TestUpdateEntity(t *testing.T) {
	dbPath := t.TempDir() + "/test_update_entity.db"
	key := "testkey123456789012"

	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Add test entity
	_, err = db.AddEntity("OldName")
	if err != nil {
		t.Fatalf("Failed to add entity: %v", err)
	}

	// Update entity
	err = db.UpdateEntity("OldName", "NewName")
	if err != nil {
		t.Fatalf("Failed to update entity: %v", err)
	}

	// Verify entity was updated
	entities, err := db.SearchEntities([]string{"NewName"})
	if err != nil {
		t.Fatalf("Failed to search entities: %v", err)
	}
	if len(entities) != 1 {
		t.Errorf("Expected 1 entity with new name, got %d", len(entities))
	}
	if len(entities) > 0 && entities[0].Text != "NewName" {
		t.Errorf("Expected entity text to be 'NewName', got '%s'", entities[0].Text)
	}

	// Verify old name no longer exists
	entities, err = db.SearchEntities([]string{"OldName"})
	if err != nil {
		t.Fatalf("Failed to search entities: %v", err)
	}
	if len(entities) != 0 {
		t.Errorf("Expected old entity name to be gone, but found %d entities", len(entities))
	}

	// Try to update non-existent entity
	err = db.UpdateEntity("NonExistent", "NewName")
	if err == nil {
		t.Error("Expected error when updating non-existent entity")
	}
}

func TestUpdateObservation(t *testing.T) {
	dbPath := t.TempDir() + "/test_update_observation.db"
	key := "testkey123456789012"

	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Add test observation
	obsID, err := db.AddObservation("TestEntity", "Old observation text")
	if err != nil {
		t.Fatalf("Failed to add observation: %v", err)
	}

	// Update observation
	err = db.UpdateObservation(obsID, "New observation text")
	if err != nil {
		t.Fatalf("Failed to update observation: %v", err)
	}

	// Verify observation was updated
	observations, err := db.SearchObservations("", []string{"New observation text"})
	if err != nil {
		t.Fatalf("Failed to search observations: %v", err)
	}
	if len(observations) != 1 {
		t.Errorf("Expected 1 observation with new text, got %d", len(observations))
	}
	if len(observations) > 0 && observations[0].Text != "New observation text" {
		t.Errorf("Expected observation text to be 'New observation text', got '%s'", observations[0].Text)
	}

	// Verify old text no longer exists
	observations, err = db.SearchObservations("", []string{"Old observation text"})
	if err != nil {
		t.Fatalf("Failed to search observations: %v", err)
	}
	if len(observations) != 0 {
		t.Errorf("Expected old observation text to be gone, but found %d observations", len(observations))
	}

	// Try to update non-existent observation
	err = db.UpdateObservation(99999, "New text")
	if err == nil {
		t.Error("Expected error when updating non-existent observation")
	}
}
