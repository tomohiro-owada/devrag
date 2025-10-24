package vectordb

import (
	"database/sql"
	"fmt"
	"os"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

// Init initializes the SQLite database
func Init(dbPath string) (*DB, error) {
	fmt.Fprintf(os.Stderr, "[INFO] Initializing database: %s\n", dbPath)

	// Enable sqlite-vec extension for all connections
	sqlite_vec.Auto()

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify sqlite-vec is loaded
	var vecVersion string
	err = conn.QueryRow("SELECT vec_version()").Scan(&vecVersion)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to load sqlite-vec extension: %w", err)
	}
	fmt.Fprintf(os.Stderr, "[INFO] sqlite-vec version: %s\n", vecVersion)

	// Enable foreign keys
	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Create tables (including vec_chunks virtual table)
	if _, err := conn.Exec(schemaSQL); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[INFO] Database initialized successfully\n")

	return &DB{conn: conn}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}
