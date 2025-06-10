// A NEW test file specifically for the read-only database queries used by the API.

package store

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

// // setupTestDB is a helper function from store_test.go, duplicated here for clarity.
// // In a real project, you might move this to a shared test helper package.
// func setupTestDB(t *testing.T) *sql.DB {
// 	t.Helper()
// 	db, err := sql.Open("sqlite3", ":memory:")
// 	if err != nil {
// 		t.Fatalf("Failed to open in-memory database: %v", err)
// 	}
// 	driver, _ := sqlite3.WithInstance(db, &sqlite3.Config{})
// 	m, _ := migrate.NewWithDatabaseInstance("file://../../migrations", "sqlite3", driver)
// 	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
// 		t.Fatalf("Failed to apply migrations: %v", err)
// 	}
// 	return db
// }

// populateDB adds test data to the database for query tests.
func populateDB(t *testing.T, db *sql.DB) {
	t.Helper()
	db.Exec(`INSERT INTO series (id, title, path, created_at, updated_at) VALUES (1, 'Series B', '/path/b', ?, ?), (2, 'Series A', '/path/a', ?, ?)`, time.Now(), time.Now(), time.Now(), time.Now())
	db.Exec(`INSERT INTO chapters (id, series_id, path, page_count, created_at, updated_at) VALUES (1, 1, '/path/b/ch1.cbz', 10, ?, ?), (2, 2, '/path/a/ch1.cbz', 20, ?, ?)`, time.Now(), time.Now(), time.Now(), time.Now())
}

func TestListSeries(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	populateDB(t, db)
	s := New(db)

	seriesList, err := s.ListSeries()
	if err != nil {
		t.Fatalf("ListSeries failed: %v", err)
	}

	if len(seriesList) != 2 {
		t.Fatalf("Expected 2 series, got %d", len(seriesList))
	}

	// Test sorting
	if seriesList[0].Title != "Series A" {
		t.Errorf("Expected first series to be 'Series A' due to sorting, got '%s'", seriesList[0].Title)
	}
}

func TestGetSeriesByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	populateDB(t, db)
	s := New(db)

	t.Run("Success", func(t *testing.T) {
		series, err := s.GetSeriesByID(2) // Get Series A
		if err != nil {
			t.Fatalf("GetSeriesByID failed: %v", err)
		}
		if series.Title != "Series A" {
			t.Errorf("Expected title 'Series A', got '%s'", series.Title)
		}
		if len(series.Chapters) != 1 {
			t.Errorf("Expected 1 chapter, got %d", len(series.Chapters))
		}
		if series.Chapters[0].PageCount != 20 {
			t.Errorf("Expected chapter page count 20, got %d", series.Chapters[0].PageCount)
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		_, err := s.GetSeriesByID(999)
		if err == nil {
			t.Error("Expected an error for non-existent series, got nil")
		}
	})
}

func TestGetChapterByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	populateDB(t, db)
	s := New(db)

	t.Run("Success", func(t *testing.T) {
		chapter, err := s.GetChapterByID(1) // Get chapter for Series B
		if err != nil {
			t.Fatalf("GetChapterByID failed: %v", err)
		}
		if chapter.PageCount != 10 {
			t.Errorf("Expected page count 10, got %d", chapter.PageCount)
		}
		if chapter.SeriesID != 1 {
			t.Errorf("Expected series ID 1, got %d", chapter.SeriesID)
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		_, err := s.GetChapterByID(999)
		if err == nil {
			t.Error("Expected an error for non-existent chapter, got nil")
		}
	})
}
