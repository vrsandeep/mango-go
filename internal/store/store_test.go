// This new test file covers all the data access layer functions.
// It uses an in-memory SQLite database to ensure tests are fast and isolated.

package store

import (
	"testing"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestGetOrCreateSeries(t *testing.T) {
	db := testutil.SetupTestDB(t)
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
	db := testutil.SetupTestDB(t)
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
	chapterID1, err := s.AddOrUpdateChapter(tx, seriesID, chapterPath, 20, "")
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
	_, err = s.AddOrUpdateChapter(tx, seriesID, chapterPath, 25, "")
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
