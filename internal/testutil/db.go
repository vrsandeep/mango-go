package testutil

import (
	"database/sql"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file" // Blank import for migration driver
	_ "github.com/mattn/go-sqlite3"                      // Blank import for sql driver
)

// SetupTestDB creates an in-memory SQLite database and applies all migrations.
// It returns the database connection, ready for use in tests.
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Use in-memory database for testing to ensure tests are fast and isolated.
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}

	// Attach a cleanup function to automatically close the DB when the test completes.
	t.Cleanup(func() {
		db.Close()
	})

	// Get a migration driver instance
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		t.Fatalf("Failed to create migration driver: %v", err)
	}

	// We need to find the migrations directory. This path assumes tests are run
	// from a package two levels deep (e.g., internal/api). Adjust if needed.
	m, err := migrate.NewWithDatabaseInstance("file://../../migrations", "sqlite3", driver)
	if err != nil {
		t.Fatalf("Failed to create migrate instance: %v", err)
	}

	// Apply all "up" migrations.
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	return db
}
