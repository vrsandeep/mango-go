package db

import (
	"database/sql"
	"fmt"

	// Import the sqlite3 driver. The blank import is used because we only
	// need the driver to be registered with database/sql.
	_ "github.com/mattn/go-sqlite3"
)

// InitDB opens a connection to the SQLite database at the specified path
// and ensures the connection is valid.
func InitDB(path string) (*sql.DB, error) {
	// The DSN for SQLite is just the file path.
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Ping the database to verify the connection is alive.
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Enable foreign key support in SQLite
	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		return nil, fmt.Errorf("failed to enable foreign key support: %w", err)
	}

	return db, nil
}
