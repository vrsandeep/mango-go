package testutil

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file" // Blank import for migration driver
	_ "github.com/mattn/go-sqlite3"                      // Blank import for sql driver
)

// https://gist.github.com/ondrek/7413434
const (
	tinyPNG_A = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII="             // Transparent
	tinyPNG_B = "iVBORw0KGgoAAAANSUhEUgAAAAoAAAAKCAYAAACNMs+9AAAAFUlEQVR42mP8z8BQz0AEYBxVSF+FABJADveWkH6oAAAAAElFTkSuQmCC" // Red
	tinyPNG_C = "iVBORw0KGgoAAAANSUhEUgAAAAoAAAAKCAYAAACNMs+9AAAAFUlEQVR42mNk+M9Qz0AEYBxVSF+FAAhKDveksOjmAAAAAElFTkSuQmCC" // Green
	tinyPNG_D = "iVBORw0KGgoAAAANSUhEUgAAAAoAAAAKCAYAAACNMs+9AAAAFUlEQVR42mNkYPhfz0AEYBxVSF+FAP5FDvcfRYWgAAAAAElFTkSuQmCC" // Blue
	tinyPNG_E = "iVBORw0KGgoAAAANSUhEUgAAAAoAAAAKCAYAAACNMs+9AAAAFUlEQVR42mP8/5+hnoEIwDiqkL4KAcT9GO0U4BxoAAAAAElFTkSuQmCC" // Yellow
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

func SetupTestLibraryAndDB(t *testing.T) (string, *sql.DB) {
	t.Helper()
	// Setup DB
	db := SetupTestDB(t)

	// Setup Library
	rootDir := t.TempDir()
	seriesADir := filepath.Join(rootDir, "Series A")
	os.Mkdir(seriesADir, 0755)
	// Create two chapters. The scanner processes files in alphabetical order,
	// so "ch1.cbz" will be scanned first. Its thumbnail should be used for the series.
	CreateTestCBZWithThumbnail(t, seriesADir, "ch1.cbz", []string{"pageA1.jpg"}, tinyPNG_A)
	CreateTestCBZWithThumbnail(t, seriesADir, "ch2.cbz", []string{"pageB1.jpg"}, tinyPNG_B)

	return rootDir, db
}
