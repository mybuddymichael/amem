package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

type DB struct {
	conn *sql.DB
	path string
	key  string
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
