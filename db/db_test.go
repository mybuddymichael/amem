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
	entities, err := db.SearchEntities([]string{"TestEntity"}, false)
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
	entities, err := db.SearchEntities([]string{"TestEntity"}, false)
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
	observations, err := db.SearchObservations("", []string{"Test observation"}, false)
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
	relationships, err := db.SearchRelationships("", "", "", []string{"knows"}, false)
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
	entities, err := db.SearchEntities([]string{"NewName"}, false)
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
	entities, err = db.SearchEntities([]string{"OldName"}, false)
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
	observations, err := db.SearchObservations("", []string{"New observation text"}, false)
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
	observations, err = db.SearchObservations("", []string{"Old observation text"}, false)
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

func TestUpdateObservationEntity(t *testing.T) {
	dbPath := t.TempDir() + "/test_update_observation_entity.db"
	key := "testkey123456789012"

	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Add two test entities
	entity1ID, err := db.AddEntity("Entity1")
	if err != nil {
		t.Fatalf("Failed to add entity1: %v", err)
	}
	entity2ID, err := db.AddEntity("Entity2")
	if err != nil {
		t.Fatalf("Failed to add entity2: %v", err)
	}

	// Add observation for entity1
	obsID, err := db.AddObservation("Entity1", "Test observation")
	if err != nil {
		t.Fatalf("Failed to add observation: %v", err)
	}

	// Update observation to point to entity2
	err = db.UpdateObservationEntity(obsID, entity2ID)
	if err != nil {
		t.Fatalf("Failed to update observation entity: %v", err)
	}

	// Verify observation now belongs to entity2
	observations, err := db.SearchObservations("Entity2", []string{}, false)
	if err != nil {
		t.Fatalf("Failed to search observations: %v", err)
	}
	if len(observations) != 1 {
		t.Errorf("Expected 1 observation for Entity2, got %d", len(observations))
	}
	if len(observations) > 0 && observations[0].ID != obsID {
		t.Errorf("Expected observation ID %d, got %d", obsID, observations[0].ID)
	}

	// Verify observation no longer belongs to entity1
	observations, err = db.SearchObservations("Entity1", []string{}, false)
	if err != nil {
		t.Fatalf("Failed to search observations: %v", err)
	}
	if len(observations) != 0 {
		t.Errorf("Expected 0 observations for Entity1, got %d", len(observations))
	}

	// Try to update with non-existent entity ID
	err = db.UpdateObservationEntity(obsID, 99999)
	if err == nil {
		t.Error("Expected error when updating with non-existent entity ID")
	}

	// Try to update non-existent observation
	err = db.UpdateObservationEntity(99999, entity1ID)
	if err == nil {
		t.Error("Expected error when updating non-existent observation")
	}
}

func TestSearchEntities(t *testing.T) {
	dbPath := t.TempDir() + "/test_search_entities.db"
	key := "testkey123456789012"

	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Add test entities
	_, err = db.AddEntity("Alice")
	if err != nil {
		t.Fatalf("Failed to add entity: %v", err)
	}
	_, err = db.AddEntity("Bob")
	if err != nil {
		t.Fatalf("Failed to add entity: %v", err)
	}
	_, err = db.AddEntity("Charlie")
	if err != nil {
		t.Fatalf("Failed to add entity: %v", err)
	}
	_, err = db.AddEntity("Alice Smith")
	if err != nil {
		t.Fatalf("Failed to add entity: %v", err)
	}

	// Search with no keywords - should return all
	entities, err := db.SearchEntities(nil, false)
	if err != nil {
		t.Fatalf("Failed to search entities: %v", err)
	}
	if len(entities) != 4 {
		t.Errorf("Expected 4 entities, got %d", len(entities))
	}

	// Search with single keyword
	entities, err = db.SearchEntities([]string{"Alice"}, false)
	if err != nil {
		t.Fatalf("Failed to search entities: %v", err)
	}
	if len(entities) != 2 {
		t.Errorf("Expected 2 entities matching 'Alice', got %d", len(entities))
	}

	// Search with multiple keywords (AND logic)
	entities, err = db.SearchEntities([]string{"Alice", "Smith"}, false)
	if err != nil {
		t.Fatalf("Failed to search entities: %v", err)
	}
	if len(entities) != 1 {
		t.Errorf("Expected 1 entity matching 'Alice' AND 'Smith', got %d", len(entities))
	}
	if len(entities) > 0 && entities[0].Text != "Alice Smith" {
		t.Errorf("Expected 'Alice Smith', got '%s'", entities[0].Text)
	}

	// Search with no matches
	entities, err = db.SearchEntities([]string{"Nonexistent"}, false)
	if err != nil {
		t.Fatalf("Failed to search entities: %v", err)
	}
	if len(entities) != 0 {
		t.Errorf("Expected 0 entities, got %d", len(entities))
	}

	// Verify results are ordered by text
	entities, err = db.SearchEntities(nil, false)
	if err != nil {
		t.Fatalf("Failed to search entities: %v", err)
	}
	if len(entities) > 1 && entities[0].Text > entities[1].Text {
		t.Error("Expected entities to be ordered by text")
	}
}

func TestSearchObservations(t *testing.T) {
	dbPath := t.TempDir() + "/test_search_observations.db"
	key := "testkey123456789012"

	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Add test data
	_, err = db.AddObservation("Alice", "Likes coffee")
	if err != nil {
		t.Fatalf("Failed to add observation: %v", err)
	}
	_, err = db.AddObservation("Alice", "Works remotely")
	if err != nil {
		t.Fatalf("Failed to add observation: %v", err)
	}
	_, err = db.AddObservation("Bob", "Likes coffee")
	if err != nil {
		t.Fatalf("Failed to add observation: %v", err)
	}
	_, err = db.AddObservation("Charlie", "Plays guitar")
	if err != nil {
		t.Fatalf("Failed to add observation: %v", err)
	}

	// Search all observations
	observations, err := db.SearchObservations("", nil, false)
	if err != nil {
		t.Fatalf("Failed to search observations: %v", err)
	}
	if len(observations) != 4 {
		t.Errorf("Expected 4 observations, got %d", len(observations))
	}

	// Search by entity text
	observations, err = db.SearchObservations("Alice", nil, false)
	if err != nil {
		t.Fatalf("Failed to search observations: %v", err)
	}
	if len(observations) != 2 {
		t.Errorf("Expected 2 observations for Alice, got %d", len(observations))
	}

	// Search by keywords
	observations, err = db.SearchObservations("", []string{"coffee"}, false)
	if err != nil {
		t.Fatalf("Failed to search observations: %v", err)
	}
	if len(observations) != 2 {
		t.Errorf("Expected 2 observations matching 'coffee', got %d", len(observations))
	}

	// Search by entity and keywords
	observations, err = db.SearchObservations("Alice", []string{"coffee"}, false)
	if err != nil {
		t.Fatalf("Failed to search observations: %v", err)
	}
	if len(observations) != 1 {
		t.Errorf("Expected 1 observation for Alice with 'coffee', got %d", len(observations))
	}
	if len(observations) > 0 && observations[0].Text != "Likes coffee" {
		t.Errorf("Expected 'Likes coffee', got '%s'", observations[0].Text)
	}

	// Search with multiple keywords (AND logic)
	observations, err = db.SearchObservations("", []string{"Alice", "coffee"}, false)
	if err != nil {
		t.Fatalf("Failed to search observations: %v", err)
	}
	if len(observations) != 1 {
		t.Errorf("Expected 1 observation matching 'Alice' AND 'coffee', got %d", len(observations))
	}

	// Search with no matches
	observations, err = db.SearchObservations("", []string{"Nonexistent"}, false)
	if err != nil {
		t.Fatalf("Failed to search observations: %v", err)
	}
	if len(observations) != 0 {
		t.Errorf("Expected 0 observations, got %d", len(observations))
	}

	// Verify observation fields are populated correctly
	observations, err = db.SearchObservations("Alice", []string{"coffee"}, false)
	if err != nil {
		t.Fatalf("Failed to search observations: %v", err)
	}
	if len(observations) > 0 {
		obs := observations[0]
		if obs.EntityText != "Alice" {
			t.Errorf("Expected entity text 'Alice', got '%s'", obs.EntityText)
		}
		if obs.EntityID == 0 {
			t.Error("Expected non-zero entity ID")
		}
		if obs.Timestamp == "" {
			t.Error("Expected non-empty timestamp")
		}
	}
}

func TestSearchRelationships(t *testing.T) {
	dbPath := t.TempDir() + "/test_search_relationships.db"
	key := "testkey123456789012"

	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Add test data
	_, err = db.AddRelationship("Alice", "Bob", "knows")
	if err != nil {
		t.Fatalf("Failed to add relationship: %v", err)
	}
	_, err = db.AddRelationship("Bob", "Charlie", "knows")
	if err != nil {
		t.Fatalf("Failed to add relationship: %v", err)
	}
	_, err = db.AddRelationship("Alice", "Charlie", "manages")
	if err != nil {
		t.Fatalf("Failed to add relationship: %v", err)
	}
	_, err = db.AddRelationship("Dave", "Alice", "reports_to")
	if err != nil {
		t.Fatalf("Failed to add relationship: %v", err)
	}

	// Search all relationships
	relationships, err := db.SearchRelationships("", "", "", nil, false)
	if err != nil {
		t.Fatalf("Failed to search relationships: %v", err)
	}
	if len(relationships) != 4 {
		t.Errorf("Expected 4 relationships, got %d", len(relationships))
	}

	// Search by from entity
	relationships, err = db.SearchRelationships("Alice", "", "", nil, false)
	if err != nil {
		t.Fatalf("Failed to search relationships: %v", err)
	}
	if len(relationships) != 2 {
		t.Errorf("Expected 2 relationships from Alice, got %d", len(relationships))
	}

	// Search by to entity
	relationships, err = db.SearchRelationships("", "Alice", "", nil, false)
	if err != nil {
		t.Fatalf("Failed to search relationships: %v", err)
	}
	if len(relationships) != 1 {
		t.Errorf("Expected 1 relationship to Alice, got %d", len(relationships))
	}

	// Search by relationship type
	relationships, err = db.SearchRelationships("", "", "knows", nil, false)
	if err != nil {
		t.Fatalf("Failed to search relationships: %v", err)
	}
	if len(relationships) != 2 {
		t.Errorf("Expected 2 'knows' relationships, got %d", len(relationships))
	}

	// Search with multiple filters
	relationships, err = db.SearchRelationships("Alice", "Bob", "knows", nil, false)
	if err != nil {
		t.Fatalf("Failed to search relationships: %v", err)
	}
	if len(relationships) != 1 {
		t.Errorf("Expected 1 relationship, got %d", len(relationships))
	}
	if len(relationships) > 0 {
		rel := relationships[0]
		if rel.FromText != "Alice" || rel.ToText != "Bob" || rel.Type != "knows" {
			t.Errorf("Expected Alice->Bob:knows, got %s->%s:%s", rel.FromText, rel.ToText, rel.Type)
		}
	}

	// Search by keywords
	relationships, err = db.SearchRelationships("", "", "", []string{"manages"}, false)
	if err != nil {
		t.Fatalf("Failed to search relationships: %v", err)
	}
	if len(relationships) != 1 {
		t.Errorf("Expected 1 relationship matching 'manages', got %d", len(relationships))
	}

	// Search with multiple keywords (AND logic)
	relationships, err = db.SearchRelationships("", "", "", []string{"Alice", "Bob"}, false)
	if err != nil {
		t.Fatalf("Failed to search relationships: %v", err)
	}
	if len(relationships) != 1 {
		t.Errorf("Expected 1 relationship matching 'Alice' AND 'Bob', got %d", len(relationships))
	}

	// Search with keywords and filters
	relationships, err = db.SearchRelationships("Alice", "", "", []string{"Charlie"}, false)
	if err != nil {
		t.Fatalf("Failed to search relationships: %v", err)
	}
	if len(relationships) != 1 {
		t.Errorf("Expected 1 relationship from Alice with 'Charlie', got %d", len(relationships))
	}

	// Search with no matches
	relationships, err = db.SearchRelationships("", "", "", []string{"Nonexistent"}, false)
	if err != nil {
		t.Fatalf("Failed to search relationships: %v", err)
	}
	if len(relationships) != 0 {
		t.Errorf("Expected 0 relationships, got %d", len(relationships))
	}

	// Verify relationship fields are populated correctly
	relationships, err = db.SearchRelationships("Alice", "Bob", "", nil, false)
	if err != nil {
		t.Fatalf("Failed to search relationships: %v", err)
	}
	if len(relationships) > 0 {
		rel := relationships[0]
		if rel.FromID == 0 || rel.ToID == 0 {
			t.Error("Expected non-zero entity IDs")
		}
		if rel.Timestamp == "" {
			t.Error("Expected non-empty timestamp")
		}
	}
}

func TestSearchAll(t *testing.T) {
	dbPath := t.TempDir() + "/test_search_all.db"
	key := "testkey123456789012"

	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Add test data with common keyword
	_, err = db.AddEntity("Project Alpha")
	if err != nil {
		t.Fatalf("Failed to add entity: %v", err)
	}
	_, err = db.AddObservation("Alice", "Working on Project Alpha")
	if err != nil {
		t.Fatalf("Failed to add observation: %v", err)
	}
	_, err = db.AddRelationship("Bob", "Project Alpha", "manages")
	if err != nil {
		t.Fatalf("Failed to add relationship: %v", err)
	}

	// Search across all types
	entities, observations, relationships, err := db.SearchAll([]string{"Alpha"}, false)
	if err != nil {
		t.Fatalf("Failed to search all: %v", err)
	}

	if len(entities) != 1 {
		t.Errorf("Expected 1 entity matching 'Alpha', got %d", len(entities))
	}
	if len(observations) != 1 {
		t.Errorf("Expected 1 observation matching 'Alpha', got %d", len(observations))
	}
	if len(relationships) != 1 {
		t.Errorf("Expected 1 relationship matching 'Alpha', got %d", len(relationships))
	}

	// Search with multiple keywords
	entities, observations, relationships, err = db.SearchAll([]string{"Project", "Alpha"}, false)
	if err != nil {
		t.Fatalf("Failed to search all: %v", err)
	}

	if len(entities) != 1 {
		t.Errorf("Expected 1 entity matching 'Project' AND 'Alpha', got %d", len(entities))
	}
	if len(observations) != 1 {
		t.Errorf("Expected 1 observation matching 'Project' AND 'Alpha', got %d", len(observations))
	}
	if len(relationships) != 1 {
		t.Errorf("Expected 1 relationship matching 'Project' AND 'Alpha', got %d", len(relationships))
	}

	// Search with no keywords
	entities, observations, relationships, err = db.SearchAll(nil, false)
	if err != nil {
		t.Fatalf("Failed to search all: %v", err)
	}

	// Should return all records
	if len(entities) < 1 {
		t.Error("Expected at least 1 entity")
	}
	if len(observations) < 1 {
		t.Error("Expected at least 1 observation")
	}
	if len(relationships) < 1 {
		t.Error("Expected at least 1 relationship")
	}

	// Search with no matches
	entities, observations, relationships, err = db.SearchAll([]string{"Nonexistent"}, false)
	if err != nil {
		t.Fatalf("Failed to search all: %v", err)
	}

	if len(entities) != 0 {
		t.Errorf("Expected 0 entities, got %d", len(entities))
	}
	if len(observations) != 0 {
		t.Errorf("Expected 0 observations, got %d", len(observations))
	}
	if len(relationships) != 0 {
		t.Errorf("Expected 0 relationships, got %d", len(relationships))
	}
}

func TestEmptyStrings(t *testing.T) {
	dbPath := t.TempDir() + "/test_empty_strings.db"
	key := "testkey123456789012"

	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Empty entity text should work
	id, err := db.AddEntity("")
	if err != nil {
		t.Fatalf("Failed to add empty entity: %v", err)
	}
	if id == 0 {
		t.Error("Expected non-zero ID for empty entity")
	}

	// Empty observation text should work
	obsID, err := db.AddObservation("test", "")
	if err != nil {
		t.Fatalf("Failed to add observation with empty text: %v", err)
	}
	if obsID == 0 {
		t.Error("Expected non-zero ID for empty observation")
	}

	// Empty relationship type should work
	relID, err := db.AddRelationship("A", "B", "")
	if err != nil {
		t.Fatalf("Failed to add relationship with empty type: %v", err)
	}
	if relID == 0 {
		t.Error("Expected non-zero ID for relationship with empty type")
	}
}

func TestSpecialCharacters(t *testing.T) {
	dbPath := t.TempDir() + "/test_special_chars.db"
	key := "testkey123456789012"

	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Test SQL injection patterns
	sqlInjection := "'; DROP TABLE entities; --"
	_, err = db.AddEntity(sqlInjection)
	if err != nil {
		t.Fatalf("Failed to add entity with SQL injection pattern: %v", err)
	}

	// Verify entity was added safely
	entities, err := db.SearchEntities([]string{sqlInjection}, false)
	if err != nil {
		t.Fatalf("Failed to search for SQL injection pattern: %v", err)
	}
	if len(entities) != 1 {
		t.Errorf("Expected 1 entity, got %d", len(entities))
	}

	// Verify tables still exist
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM entities").Scan(&count)
	if err != nil {
		t.Fatalf("Tables were damaged by SQL injection: %v", err)
	}

	// Test unicode and special characters
	unicode := "用户™ ñ 🚀"
	id2, err := db.AddEntity(unicode)
	if err != nil {
		t.Fatalf("Failed to add entity with unicode: %v", err)
	}

	entities, err = db.SearchEntities([]string{unicode}, false)
	if err != nil {
		t.Fatalf("Failed to search for unicode: %v", err)
	}
	if len(entities) != 1 {
		t.Errorf("Expected 1 unicode entity, got %d", len(entities))
	}
	if len(entities) > 0 && entities[0].ID != id2 {
		t.Errorf("Unicode entity not found correctly")
	}
}

func TestWrongEncryptionKey(t *testing.T) {
	dbPath := t.TempDir() + "/test_wrong_key.db"
	key := "testkey123456789012"

	// Create database with correct key
	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	_ = db.Close()

	// Try to open with wrong key
	wrongKey := "wrongkey123456789012"
	_, err = Open(dbPath, wrongKey)
	if err == nil {
		t.Error("Expected error when opening with wrong key")
	}
}

func TestEmptyEncryptionKey(t *testing.T) {
	dbPath := t.TempDir() + "/test_empty_key.db"

	// Try to open with empty key
	_, err := Open(dbPath, "")
	if err == nil {
		t.Error("Expected error when opening with empty key")
	}
}

func TestCountFunctions(t *testing.T) {
	dbPath := t.TempDir() + "/test_count.db"
	key := "testkey123456789012"

	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Initial counts should be zero
	count, err := db.CountEntities()
	if err != nil {
		t.Fatalf("Failed to count entities: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 entities, got %d", count)
	}

	count, err = db.CountObservations()
	if err != nil {
		t.Fatalf("Failed to count observations: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 observations, got %d", count)
	}

	count, err = db.CountRelationships()
	if err != nil {
		t.Fatalf("Failed to count relationships: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 relationships, got %d", count)
	}

	// Add some data
	_, err = db.AddEntity("Entity1")
	if err != nil {
		t.Fatalf("Failed to add entity: %v", err)
	}
	_, err = db.AddEntity("Entity2")
	if err != nil {
		t.Fatalf("Failed to add entity: %v", err)
	}

	_, err = db.AddObservation("Entity1", "Obs1")
	if err != nil {
		t.Fatalf("Failed to add observation: %v", err)
	}
	_, err = db.AddObservation("Entity1", "Obs2")
	if err != nil {
		t.Fatalf("Failed to add observation: %v", err)
	}
	_, err = db.AddObservation("Entity2", "Obs3")
	if err != nil {
		t.Fatalf("Failed to add observation: %v", err)
	}

	_, err = db.AddRelationship("Entity1", "Entity2", "knows")
	if err != nil {
		t.Fatalf("Failed to add relationship: %v", err)
	}

	// Verify counts
	count, err = db.CountEntities()
	if err != nil {
		t.Fatalf("Failed to count entities: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 entities, got %d", count)
	}

	count, err = db.CountObservations()
	if err != nil {
		t.Fatalf("Failed to count observations: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected 3 observations, got %d", count)
	}

	count, err = db.CountRelationships()
	if err != nil {
		t.Fatalf("Failed to count relationships: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 relationship, got %d", count)
	}
}

func TestWhitespaceHandling(t *testing.T) {
	dbPath := t.TempDir() + "/test_whitespace.db"
	key := "testkey123456789012"

	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Entities with different whitespace are considered different
	id1, err := db.AddEntity("Alice")
	if err != nil {
		t.Fatalf("Failed to add entity: %v", err)
	}

	id2, err := db.AddEntity(" Alice")
	if err != nil {
		t.Fatalf("Failed to add entity with leading space: %v", err)
	}

	id3, err := db.AddEntity("Alice ")
	if err != nil {
		t.Fatalf("Failed to add entity with trailing space: %v", err)
	}

	// All three should be different entities
	if id1 == id2 || id1 == id3 || id2 == id3 {
		t.Error("Entities with different whitespace should be distinct")
	}

	// Search should find all three
	entities, err := db.SearchEntities([]string{"Alice"}, false)
	if err != nil {
		t.Fatalf("Failed to search entities: %v", err)
	}
	if len(entities) != 3 {
		t.Errorf("Expected 3 entities with 'Alice', got %d", len(entities))
	}
}

func TestDuplicateRelationships(t *testing.T) {
	dbPath := t.TempDir() + "/test_dup_rels.db"
	key := "testkey123456789012"

	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Add same relationship twice
	id1, err := db.AddRelationship("Alice", "Bob", "knows")
	if err != nil {
		t.Fatalf("Failed to add relationship: %v", err)
	}

	id2, err := db.AddRelationship("Alice", "Bob", "knows")
	if err != nil {
		t.Fatalf("Failed to add duplicate relationship: %v", err)
	}

	// Should create two distinct relationships
	if id1 == id2 {
		t.Error("Duplicate relationships should have different IDs")
	}

	// Should find both relationships
	rels, err := db.SearchRelationships("Alice", "Bob", "knows", nil, false)
	if err != nil {
		t.Fatalf("Failed to search relationships: %v", err)
	}
	if len(rels) != 2 {
		t.Errorf("Expected 2 relationships, got %d", len(rels))
	}
}

func TestUpdateEntityConflict(t *testing.T) {
	dbPath := t.TempDir() + "/test_update_conflict.db"
	key := "testkey123456789012"

	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Create two entities
	_, err = db.AddEntity("Alice")
	if err != nil {
		t.Fatalf("Failed to add entity: %v", err)
	}

	_, err = db.AddEntity("Bob")
	if err != nil {
		t.Fatalf("Failed to add entity: %v", err)
	}

	// Try to update Alice to Bob (should fail due to unique constraint)
	err = db.UpdateEntity("Alice", "Bob")
	if err == nil {
		t.Error("Expected error when updating entity to existing name")
	}

	// Verify Alice still exists
	entities, err := db.SearchEntities([]string{"Alice"}, false)
	if err != nil {
		t.Fatalf("Failed to search entities: %v", err)
	}
	if len(entities) != 1 {
		t.Error("Alice should still exist after failed update")
	}
}

func TestUpdateObservationEmpty(t *testing.T) {
	dbPath := t.TempDir() + "/test_update_obs_empty.db"
	key := "testkey123456789012"

	db, err := Init(dbPath, key)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Add observation
	obsID, err := db.AddObservation("Alice", "Original text")
	if err != nil {
		t.Fatalf("Failed to add observation: %v", err)
	}

	// Update to empty string
	err = db.UpdateObservation(obsID, "")
	if err != nil {
		t.Fatalf("Failed to update observation to empty string: %v", err)
	}

	// Verify update worked
	observations, err := db.SearchObservations("Alice", nil, false)
	if err != nil {
		t.Fatalf("Failed to search observations: %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("Expected 1 observation, got %d", len(observations))
	}
	if observations[0].Text != "" {
		t.Errorf("Expected empty observation text, got '%s'", observations[0].Text)
	}
}

func TestEntityFormat(t *testing.T) {
	entity := Entity{ID: 42, Text: "Alice"}

	// Test without ID
	result := entity.Format(false)
	expected := "Alice"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test with ID
	result = entity.Format(true)
	expected = "[42] Alice"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test empty text
	emptyEntity := Entity{ID: 1, Text: ""}
	result = emptyEntity.Format(false)
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}

	result = emptyEntity.Format(true)
	expected = "[1] "
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestObservationFormat(t *testing.T) {
	obs := Observation{
		ID:         123,
		EntityText: "Alice",
		Text:       "Likes coffee",
		Timestamp:  "2024-01-15 10:30:00",
	}

	// Test without ID
	result := obs.Format(false)
	expected := "Alice: Likes coffee (2024-01-15 10:30:00)"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test with ID
	result = obs.Format(true)
	expected = "[123] Alice: Likes coffee (2024-01-15 10:30:00)"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test empty text
	emptyObs := Observation{
		ID:         1,
		EntityText: "Bob",
		Text:       "",
		Timestamp:  "2024-01-15 10:30:00",
	}
	result = emptyObs.Format(false)
	expected = "Bob:  (2024-01-15 10:30:00)"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestRelationshipFormat(t *testing.T) {
	rel := Relationship{
		ID:        456,
		FromText:  "Alice",
		ToText:    "Bob",
		Type:      "knows",
		Timestamp: "2024-01-15 10:30:00",
	}

	// Test without ID
	result := rel.Format(false)
	expected := "Alice -[knows]-> Bob (2024-01-15 10:30:00)"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test with ID
	result = rel.Format(true)
	expected = "[456] Alice -[knows]-> Bob (2024-01-15 10:30:00)"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test empty type
	emptyRel := Relationship{
		ID:        1,
		FromText:  "Alice",
		ToText:    "Bob",
		Type:      "",
		Timestamp: "2024-01-15 10:30:00",
	}
	result = emptyRel.Format(false)
	expected = "Alice -[]-> Bob (2024-01-15 10:30:00)"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}
