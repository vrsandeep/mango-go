package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"

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

	// Enable foreign key support in SQLite
	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		return nil, fmt.Errorf("failed to enable foreign key support: %w", err)
	}

	// Ping the database to verify the connection is alive.
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

func RunMigrations(database *sql.DB) error {
	// Enable foreign keys before running migrations
	_, err := database.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		return fmt.Errorf("failed to enable foreign key support before migrations: %w", err)
	}

	driver, err := sqlite3.WithInstance(database, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("could not create sqlite3 migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"sqlite3",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	log.Println("Applying database migrations...")
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("an error occurred while applying migrations: %w", err)
	}

	log.Println("Migrations applied successfully.")
	return nil
}
