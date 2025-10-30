package db

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

type DB struct {
	conn *sql.DB
	path string
	key  string
}

type Entity struct {
	ID   int64
	Text string
}

type Observation struct {
	ID         int64
	EntityID   int64
	EntityText string
	Text       string
	Timestamp  string
}

type Relationship struct {
	ID        int64
	FromID    int64
	FromText  string
	ToID      int64
	ToText    string
	Type      string
	Timestamp string
}

// Format returns a formatted string representation of the entity.
func (e Entity) Format(withID bool) string {
	if withID {
		return fmt.Sprintf("[%d] %s", e.ID, e.Text)
	}
	return e.Text
}

// Format returns a formatted string representation of the observation.
func (o Observation) Format(withID bool) string {
	if withID {
		return fmt.Sprintf("[%d] %s: %s (%s)", o.ID, o.EntityText, o.Text, o.Timestamp)
	}
	return fmt.Sprintf("%s: %s (%s)", o.EntityText, o.Text, o.Timestamp)
}

// Format returns a formatted string representation of the relationship.
func (r Relationship) Format(withID bool) string {
	if withID {
		return fmt.Sprintf("[%d] %s -[%s]-> %s (%s)", r.ID, r.FromText, r.Type, r.ToText, r.Timestamp)
	}
	return fmt.Sprintf("%s -[%s]-> %s (%s)", r.FromText, r.Type, r.ToText, r.Timestamp)
}

func Open(path, key string) (*DB, error) {
	if key == "" {
		return nil, fmt.Errorf("encryption key is required")
	}

	dsn := fmt.Sprintf("file:%s?_pragma_key=%s&_pragma_cipher_page_size=4096", path, key)
	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Enable foreign key constraints
	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	return &DB{
		conn: conn,
		path: path,
		key:  key,
	}, nil
}

func Init(path, key string) (*DB, error) {
	db, err := Open(path, key)
	if err != nil {
		return nil, err
	}

	if err := migrate(db.conn); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Path() string {
	return db.path
}

func (db *DB) IsEncrypted() (bool, error) {
	// Check if we can query the database
	var result int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM sqlite_master").Scan(&result)
	return err == nil, err
}

func (db *DB) Exists() bool {
	_, err := os.Stat(db.path)
	return err == nil
}

func (db *DB) Conn() *sql.DB {
	return db.conn
}

// AddEntity adds an entity to the database.
// Returns the entity ID (existing or new).
func (db *DB) AddEntity(text string) (int64, error) {
	// Use INSERT OR IGNORE to avoid duplicate key errors
	_, err := db.conn.Exec("INSERT OR IGNORE INTO entities (text) VALUES (?)", text)
	if err != nil {
		return 0, fmt.Errorf("failed to insert entity: %w", err)
	}

	// Always fetch the ID (works for both new and existing entities)
	var id int64
	err = db.conn.QueryRow("SELECT id FROM entities WHERE text = ?", text).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to get entity id: %w", err)
	}

	return id, nil
}

// getEntityID returns the ID of an entity by text, or creates it if it doesn't exist.
func (db *DB) getEntityID(text string) (int64, error) {
	return db.AddEntity(text)
}

// AddObservation adds an observation about an entity.
// Creates the entity if it doesn't exist. Returns the observation ID.
func (db *DB) AddObservation(entityText, observationText string) (int64, error) {
	entityID, err := db.getEntityID(entityText)
	if err != nil {
		return 0, err
	}

	result, err := db.conn.Exec("INSERT INTO observations (entity_id, text) VALUES (?, ?)", entityID, observationText)
	if err != nil {
		return 0, fmt.Errorf("failed to insert observation: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return id, nil
}

// AddRelationship adds a relationship between two entities.
// Creates entities if they don't exist. Returns the relationship ID.
func (db *DB) AddRelationship(fromText, toText, relType string) (int64, error) {
	fromID, err := db.getEntityID(fromText)
	if err != nil {
		return 0, err
	}

	toID, err := db.getEntityID(toText)
	if err != nil {
		return 0, err
	}

	result, err := db.conn.Exec("INSERT INTO relationships (from_id, to_id, type) VALUES (?, ?, ?)", fromID, toID, relType)
	if err != nil {
		return 0, fmt.Errorf("failed to insert relationship: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return id, nil
}

// buildWhereClause builds a WHERE clause for keyword matching across multiple columns.
func buildWhereClause(keywords []string, columns []string, useUnion bool) (string, []interface{}) {
	if len(keywords) == 0 {
		return "", nil
	}

	var conditions []string
	var args []interface{}

	for _, keyword := range keywords {
		var columnConditions []string
		for _, col := range columns {
			columnConditions = append(columnConditions, fmt.Sprintf("%s LIKE ?", col))
			args = append(args, "%"+keyword+"%")
		}
		conditions = append(conditions, "("+strings.Join(columnConditions, " OR ")+")")
	}

	joiner := " AND "
	if useUnion {
		joiner = " OR "
	}
	return strings.Join(conditions, joiner), args
}

// DeleteEntity deletes an entity by ID.
// Observations and relationships are cascade deleted by the database.
func (db *DB) DeleteEntity(id int64) error {
	result, err := db.conn.Exec("DELETE FROM entities WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("entity with ID %d not found", id)
	}

	return nil
}

// DeleteEntityByText deletes an entity by text.
// Observations and relationships are cascade deleted by the database.
func (db *DB) DeleteEntityByText(text string) error {
	result, err := db.conn.Exec("DELETE FROM entities WHERE text = ?", text)
	if err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("entity '%s' not found", text)
	}

	return nil
}

// DeleteObservation deletes an observation by ID.
func (db *DB) DeleteObservation(id int64) error {
	result, err := db.conn.Exec("DELETE FROM observations WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete observation: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("observation with ID %d not found", id)
	}

	return nil
}

// DeleteRelationship deletes a relationship by ID.
func (db *DB) DeleteRelationship(id int64) error {
	result, err := db.conn.Exec("DELETE FROM relationships WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete relationship: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("relationship with ID %d not found", id)
	}

	return nil
}

// SearchEntities searches entities by keywords.
func (db *DB) SearchEntities(keywords []string, useUnion bool) ([]Entity, error) {
	query := "SELECT id, text FROM entities"
	var args []interface{}

	if len(keywords) > 0 {
		whereClause, whereArgs := buildWhereClause(keywords, []string{"text"}, useUnion)
		query += " WHERE " + whereClause
		args = whereArgs
	}

	query += " ORDER BY text"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search entities: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []Entity
	for rows.Next() {
		var e Entity
		if err := rows.Scan(&e.ID, &e.Text); err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}
		results = append(results, e)
	}

	return results, rows.Err()
}

// SearchObservations searches observations with optional entity filter and keywords.
func (db *DB) SearchObservations(entityText string, keywords []string, useUnion bool) ([]Observation, error) {
	query := `
		SELECT o.id, o.entity_id, e.text, o.text, o.timestamp
		FROM observations o
		JOIN entities e ON o.entity_id = e.id
	`
	var args []interface{}
	var whereClauses []string

	if entityText != "" {
		whereClauses = append(whereClauses, "e.text LIKE ?")
		args = append(args, "%"+entityText+"%")
	}

	if len(keywords) > 0 {
		whereClause, whereArgs := buildWhereClause(keywords, []string{"o.text", "e.text"}, useUnion)
		whereClauses = append(whereClauses, "("+whereClause+")")
		args = append(args, whereArgs...)
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += " ORDER BY o.timestamp DESC"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search observations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []Observation
	for rows.Next() {
		var o Observation
		if err := rows.Scan(&o.ID, &o.EntityID, &o.EntityText, &o.Text, &o.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan observation: %w", err)
		}
		results = append(results, o)
	}

	return results, rows.Err()
}

// SearchRelationships searches relationships with optional filters.
func (db *DB) SearchRelationships(fromText, toText, relType string, keywords []string, useUnion bool) ([]Relationship, error) {
	query := `
		SELECT r.id, r.from_id, e1.text, r.to_id, e2.text, r.type, r.timestamp
		FROM relationships r
		JOIN entities e1 ON r.from_id = e1.id
		JOIN entities e2 ON r.to_id = e2.id
	`
	var args []interface{}
	var whereClauses []string

	if fromText != "" {
		whereClauses = append(whereClauses, "e1.text LIKE ?")
		args = append(args, "%"+fromText+"%")
	}

	if toText != "" {
		whereClauses = append(whereClauses, "e2.text LIKE ?")
		args = append(args, "%"+toText+"%")
	}

	if relType != "" {
		whereClauses = append(whereClauses, "r.type LIKE ?")
		args = append(args, "%"+relType+"%")
	}

	if len(keywords) > 0 {
		whereClause, whereArgs := buildWhereClause(keywords, []string{"e1.text", "e2.text", "r.type"}, useUnion)
		whereClauses = append(whereClauses, "("+whereClause+")")
		args = append(args, whereArgs...)
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += " ORDER BY r.timestamp DESC"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search relationships: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []Relationship
	for rows.Next() {
		var r Relationship
		if err := rows.Scan(&r.ID, &r.FromID, &r.FromText, &r.ToID, &r.ToText, &r.Type, &r.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan relationship: %w", err)
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

// SearchAll searches across all types (entities, observations, relationships).
func (db *DB) SearchAll(keywords []string, useUnion bool) ([]Entity, []Observation, []Relationship, error) {
	entities, err := db.SearchEntities(keywords, useUnion)
	if err != nil {
		return nil, nil, nil, err
	}

	observations, err := db.SearchObservations("", keywords, useUnion)
	if err != nil {
		return nil, nil, nil, err
	}

	relationships, err := db.SearchRelationships("", "", "", keywords, useUnion)
	if err != nil {
		return nil, nil, nil, err
	}

	return entities, observations, relationships, nil
}

// CountEntities returns the total number of entities.
func (db *DB) CountEntities() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM entities").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count entities: %w", err)
	}
	return count, nil
}

// CountObservations returns the total number of observations.
func (db *DB) CountObservations() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM observations").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count observations: %w", err)
	}
	return count, nil
}

// CountRelationships returns the total number of relationships.
func (db *DB) CountRelationships() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM relationships").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count relationships: %w", err)
	}
	return count, nil
}

// UpdateEntity updates an entity's text by its current text.
func (db *DB) UpdateEntity(text, newText string) error {
	result, err := db.conn.Exec("UPDATE entities SET text = ? WHERE text = ?", newText, text)
	if err != nil {
		return fmt.Errorf("failed to update entity: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("entity '%s' not found", text)
	}

	return nil
}

// UpdateObservation updates an observation's text by ID.
func (db *DB) UpdateObservation(id int64, newText string) error {
	result, err := db.conn.Exec("UPDATE observations SET text = ? WHERE id = ?", newText, id)
	if err != nil {
		return fmt.Errorf("failed to update observation: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("observation with ID %d not found", id)
	}

	return nil
}

// UpdateObservationEntity updates which entity an observation is about.
func (db *DB) UpdateObservationEntity(id int64, newEntityID int64) error {
	// Validate the new entity exists
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM entities WHERE id = ?", newEntityID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check entity: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("entity with ID %d not found", newEntityID)
	}

	// Update the observation's entity_id
	result, err := db.conn.Exec("UPDATE observations SET entity_id = ? WHERE id = ?", newEntityID, id)
	if err != nil {
		return fmt.Errorf("failed to update observation: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("observation with ID %d not found", id)
	}

	return nil
}
