package store_test

import (
	"database/sql"
	"slices"
	"testing"
	"time"

	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

// populateDB is updated to include default progress values
func populateDB(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO series (id, title, path, created_at, updated_at) VALUES (1, 'Series B', '/path/b', ?, ?), (2, 'Series A', '/path/a', ?, ?)`, time.Now(), time.Now(), time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to populate series in test database: %v", err)
		return
	}
	// Insert with default read=false and progress_percent=0
	_, err = db.Exec(`INSERT INTO chapters (id, series_id, path, page_count, created_at, updated_at) VALUES (1, 1, '/path/b/ch1.cbz', 10, ?, ?), (2, 2, '/path/a/ch1.cbz', 20, ?, ?)`, time.Now(), time.Now(), time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to populate chapter in test database: %v", err)
		return
	}

	_, err = db.Exec(`
		INSERT INTO users (id, username, password_hash, role, created_at)
		VALUES (1, 'testuser', 'password', 'user', CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatalf("Failed to populate user in test database: %v", err)
		return
	}
}

func TestListSeries(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	populateDB(t, db)
	s := store.New(db)

	t.Run("Without Search", func(t *testing.T) {
		seriesList, seriesCount, err := s.ListSeries(1, 1, 50, "", "title", "desc")
		if err != nil {
			t.Fatalf("ListSeries failed: %v", err)
		}
		if seriesCount != 2 {
			t.Errorf("Expected 2 series, got %d", seriesCount)
		}
		if len(seriesList) != 2 {
			t.Fatalf("Expected 2 series, got %d", len(seriesList))
		}
		// Check sorting
		if seriesList[0].Title != "Series B" {
			t.Errorf("Expected first series to be 'Series B' due to sorting, got '%s'", seriesList[0].Title)
		}
		if seriesList[0].CustomCoverURL != "" {
			t.Errorf("Expected CustomCoverURL to be empty, got '%s'", seriesList[0].CustomCoverURL)
		}
		if seriesList[0].TotalChapters != 1 {
			t.Errorf("Expected Series B to have 1 chapter, got %d", seriesList[0].TotalChapters)
		}
		if seriesList[1].ReadChapters != 0 {
			t.Errorf("Expected Series A to have 0 read chapters, got %d", seriesList[1].ReadChapters)
		}
	})

	t.Run("With Search", func(t *testing.T) {
		seriesList, seriesCount, err := s.ListSeries(1, 1, 50, "A", "title", "asc")
		if err != nil {
			t.Fatalf("ListSeries with search failed: %v", err)
		}
		if seriesCount != 1 {
			t.Errorf("Expected 1 series, got %d", seriesCount)
		}
		if len(seriesList) != 1 {
			t.Fatalf("Expected 1 series, got %d", len(seriesList))
		}
		if seriesList[0].Title != "Series A" {
			t.Errorf("Expected series to be 'Series A', got '%s'", seriesList[0].Title)
		}
	})
	t.Run("With Invalid Sort Direction", func(t *testing.T) {
		seriesList, seriesCount, err := s.ListSeries(1, 1, 50, "", "title", "invalid")
		if err != nil {
			t.Fatalf("ListSeries with invalid sort direction failed: %v", err)
		}
		if seriesCount != 2 {
			t.Errorf("Expected 2 series, got %d", seriesCount)
		}
		if len(seriesList) != 2 {
			t.Fatalf("Expected 2 series, got %d", len(seriesList))
		}
		if seriesList[0].Title != "Series A" {
			t.Errorf("Expected first series to be 'Series A' due to sorting, got '%s'", seriesList[0].Title)
		}
		if seriesList[0].CustomCoverURL != "" {
			t.Errorf("Expected CustomCoverURL to be empty, got '%s'", seriesList[0].CustomCoverURL)
		}
	})
}

func TestGetSeriesByID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	populateDB(t, db)
	s := store.New(db)

	t.Run("Success", func(t *testing.T) {
		series, count, err := s.GetSeriesByID(2, 1, 1, 10, "ch1", "path", "asc") // Get Series A
		if err != nil {
			t.Fatalf("GetSeriesByID failed: %v", err)
		}
		if series.Title != "Series A" {
			t.Errorf("Expected title 'Series A', got '%s'", series.Title)
		}
		if len(series.Chapters) != 1 {
			t.Errorf("Expected 1 chapter, got %d", len(series.Chapters))
		}
		if count != 1 {
			t.Errorf("Expected number of chapters to be 1, got %d", count)
		}
		if series.Chapters[0].PageCount != 20 {
			t.Errorf("Expected chapter page count 20, got %d", series.Chapters[0].PageCount)
		}
		if series.Chapters[0].Read != false {
			t.Errorf("Expected chapter 'read' status to be false, got true")
		}
		if series.Chapters[0].ProgressPercent != 0 {
			t.Errorf("Expected chapter 'progress_percent' to be 0, got %d", series.Chapters[0].ProgressPercent)
		}
		if series.CustomCoverURL != "" {
			t.Errorf("Expected CustomCoverURL to be empty, got '%s'", series.CustomCoverURL)
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		_, count, err := s.GetSeriesByID(999, 1, 1, 10, "", "", "")
		if err == nil {
			t.Error("Expected an error for non-existent series, got nil")
		}
		if count != 0 {
			t.Errorf("Expected count to be 0 for non-existent series, got %d", count)
		}
	})
}

func TestUpdateSeriesCoverURL(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	seriesID := int64(1)
	newURL := "http://example.com/cover.png"

	// test with non existing series ID
	rowsAffected, err := s.UpdateSeriesCoverURL(999, newURL)
	if err != nil {
		t.Error("Expected no error when updating cover URL for non-existent series")
	}
	if rowsAffected != 0 {
		t.Errorf("Expected 0 rows affected for non-existent series, got %d", rowsAffected)
	}

	populateDB(t, db)
	rowsAffected, err = s.UpdateSeriesCoverURL(seriesID, newURL)
	if err != nil {
		t.Fatalf("UpdateSeriesCoverURL failed: %v", err)
	}
	if rowsAffected == 0 {
		t.Errorf("Expected 1 row affected for existing series, got %d", rowsAffected)
	}

	// Verify the update
	series, count, err := s.GetSeriesByID(seriesID, 1, 1, 1, "", "", "")
	if err != nil {
		t.Fatalf("GetSeriesByID failed after update: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 chapter after update, got %d", count)
	}

	if series.CustomCoverURL != newURL {
		t.Errorf("Expected CustomCoverURL to be '%s', got '%s'", newURL, series.CustomCoverURL)
	}
}

func TestGetChapterByID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	populateDB(t, db)
	s := store.New(db)

	t.Run("Success", func(t *testing.T) {
		chapter, err := s.GetChapterByID(1, 1) // Get chapter for Series B
		if err != nil {
			t.Fatalf("GetChapterByID failed: %v", err)
		}
		if chapter.PageCount != 10 {
			t.Errorf("Expected page count 10, got %d", chapter.PageCount)
		}
		if chapter.FolderID != 1 {
			t.Errorf("Expected folder ID 1, got %d", chapter.FolderID)
		}
		if chapter.Read != false {
			t.Errorf("Expected chapter 'read' status to be false, got true")
		}
		if chapter.ProgressPercent != 0 {
			t.Errorf("Expected chapter 'progress_percent' to be 0, got %d", chapter.ProgressPercent)
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		_, err := s.GetChapterByID(999, 1)
		if err == nil {
			t.Error("Expected an error for non-existent chapter, got nil")
		}
	})
}

func TestGetAllChapterPaths(t *testing.T) {
	db := testutil.SetupTestDB(t)
	setupFullTestDB(t, db) // Use helper from store_test.go
	s := store.New(db)

	paths, err := s.GetAllChapterPaths()
	if err != nil {
		t.Fatalf("GetAllChapterPaths failed: %v", err)
	}
	if len(paths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(paths))
	}
	if slices.Equal(paths, []string{"/path/a/ch1.cbz", "/path/a/ch2.cbz"}) == false {
		t.Errorf("Expected paths to be ['/path/a/ch1.cbz', '/path/a/ch2.cbz'], got %v", paths)
	}
}

func TestGetAllChaptersForThumbnailing(t *testing.T) {
	db := testutil.SetupTestDB(t)
	setupFullTestDB(t, db)
	s := store.New(db)

	chapters, err := s.GetAllChaptersForThumbnailing()
	if err != nil {
		t.Fatalf("GetAllChaptersForThumbnailing failed: %v", err)
	}
	if len(chapters) != 2 {
		t.Errorf("Expected 2 chapters, got %d", len(chapters))
	}
	if chapters[0].ID == 0 || chapters[0].Path == "" {
		t.Errorf("Chapter data is incomplete: %+v", chapters[0])
	}
}
