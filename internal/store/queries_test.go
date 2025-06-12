package store

import (
	"database/sql"
	"testing"
	"time"

	"github.com/vrsandeep/mango-go/internal/testutil"
)

// populateDB is updated to include default progress values
func populateDB(t *testing.T, db *sql.DB) {
	t.Helper()
	db.Exec(`INSERT INTO series (id, title, path, created_at, updated_at) VALUES (1, 'Series B', '/path/b', ?, ?), (2, 'Series A', '/path/a', ?, ?)`, time.Now(), time.Now(), time.Now(), time.Now())
	// Insert with default read=false and current_page=0
	db.Exec(`INSERT INTO chapters (id, series_id, path, page_count, created_at, updated_at) VALUES (1, 1, '/path/b/ch1.cbz', 10, ?, ?), (2, 2, '/path/a/ch1.cbz', 20, ?, ?)`, time.Now(), time.Now(), time.Now(), time.Now())
}

func TestListSeries(t *testing.T) {
	db := testutil.SetupTestDB(t)
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
	db := testutil.SetupTestDB(t)
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
		if series.Chapters[0].Read != false {
			t.Errorf("Expected chapter 'read' status to be false, got true")
		}
		if series.Chapters[0].CurrentPage != 0 {
			t.Errorf("Expected chapter 'current_page' to be 0, got %d", series.Chapters[0].CurrentPage)
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
	db := testutil.SetupTestDB(t)
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
		if chapter.Read != false {
			t.Errorf("Expected chapter 'read' status to be false, got true")
		}
		if chapter.CurrentPage != 0 {
			t.Errorf("Expected chapter 'current_page' to be 0, got %d", chapter.CurrentPage)
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		_, err := s.GetChapterByID(999)
		if err == nil {
			t.Error("Expected an error for non-existent chapter, got nil")
		}
	})
}

func TestUpdateChapterProgress(t *testing.T) {
	db := testutil.SetupTestDB(t)
	populateDB(t, db)
	s := New(db)

	chapterID := int64(1)
	newPage := 50
	newReadStatus := true

	err := s.UpdateChapterProgress(chapterID, newPage, newReadStatus)
	if err != nil {
		t.Fatalf("UpdateChapterProgress failed: %v", err)
	}

	// Verify that the data was updated correctly
	chapter, err := s.GetChapterByID(chapterID)
	if err != nil {
		t.Fatalf("Failed to get chapter after update: %v", err)
	}

	if chapter.CurrentPage != newPage {
		t.Errorf("Expected current_page to be %d, got %d", newPage, chapter.CurrentPage)
	}
	if chapter.Read != newReadStatus {
		t.Errorf("Expected read status to be %t, got %t", newReadStatus, chapter.Read)
	}
}
