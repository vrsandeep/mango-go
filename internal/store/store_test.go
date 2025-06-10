// This new test file covers all the data access layer functions.
// It uses an in-memory SQLite database to ensure tests are fast and isolated.

package store

import (
	"database/sql"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates an in-memory SQLite database and applies migrations.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Use in-memory database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}

	// Run migrations
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		t.Fatalf("Failed to create migration driver: %v", err)
	}

	// We need to find the migrations directory.
	// This path assumes tests are run from the project root.
	m, err := migrate.NewWithDatabaseInstance("file://../../migrations", "sqlite3", driver)
	if err != nil {
		t.Fatalf("Failed to create migrate instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	return db
}

func TestGetOrCreateSeries(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := New(db)

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// First time: should create the series
	seriesID1, err := s.GetOrCreateSeries(tx, "Test Series", "/path/to/series")
	if err != nil {
		t.Fatalf("GetOrCreateSeries (create) failed: %v", err)
	}
	if seriesID1 != 1 {
		t.Errorf("Expected new series ID to be 1, got %d", seriesID1)
	}

	// Second time: should retrieve the existing series
	seriesID2, err := s.GetOrCreateSeries(tx, "Test Series", "/path/to/series")
	if err != nil {
		t.Fatalf("GetOrCreateSeries (get) failed: %v", err)
	}
	if seriesID2 != seriesID1 {
		t.Errorf("Expected existing series ID to be %d, got %d", seriesID1, seriesID2)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}
}

func TestAddOrUpdateChapter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := New(db)

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// First, create a series to associate with the chapter
	seriesID, err := s.GetOrCreateSeries(tx, "Test Series", "/path/to/series")
	if err != nil {
		t.Fatalf("Setup: GetOrCreateSeries failed: %v", err)
	}

	// First time: should add the chapter
	chapterPath := "/path/to/series/ch1.cbz"
	chapterID1, err := s.AddOrUpdateChapter(tx, seriesID, chapterPath, 20)
	if err != nil {
		t.Fatalf("AddOrUpdateChapter (add) failed: %v", err)
	}

	// Check if it was inserted correctly
	var pageCount int
	err = tx.QueryRow("SELECT page_count FROM chapters WHERE id = ?", chapterID1).Scan(&pageCount)
	if err != nil {
		t.Fatalf("Failed to query new chapter: %v", err)
	}
	if pageCount != 20 {
		t.Errorf("Expected page count 20, got %d", pageCount)
	}

	// Second time: should update the chapter
	_, err = s.AddOrUpdateChapter(tx, seriesID, chapterPath, 25)
	if err != nil {
		t.Fatalf("AddOrUpdateChapter (update) failed: %v", err)
	}

	// Check if it was updated correctly
	err = tx.QueryRow("SELECT page_count FROM chapters WHERE id = ?", chapterID1).Scan(&pageCount)
	if err != nil {
		t.Fatalf("Failed to query updated chapter: %v", err)
	}
	if pageCount != 25 {
		t.Errorf("Expected updated page count 25, got %d", pageCount)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}
}
