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
