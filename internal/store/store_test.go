// This new test file covers all the data access layer functions.
// It uses an in-memory SQLite database to ensure tests are fast and isolated.

package store

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/vrsandeep/mango-go/internal/models"
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

func TestDeleteChapterByPath(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s, seriesID, chapterID, chapterPath := setupFullTestDB(t, db)

	err := s.DeleteChapterByPath(chapterPath)
	if err != nil {
		t.Fatalf("DeleteChapterByPath failed: %v", err)
	}

	_, err = s.GetChapterByID(chapterID, 1)
	if err == nil {
		t.Error("Expected error when getting deleted chapter, but got nil")
	}

	// Verify other chapter still exists
	var chapterCount int
	var series *models.Series
	series, chapterCount, err = s.GetSeriesByID(seriesID, 1, 1, 10, "", "", "")
	if err != nil {
		t.Errorf("Other chapter was deleted unexpectedly: %v", err)
	}
	if chapterCount != 1 {
		t.Errorf("Expected 1 chapter in series after deletion, got %d", chapterCount)
	}
	if series.Chapters[0].ID == chapterID {
		t.Errorf("Expected chapter with ID %d to be deleted", chapterID)
	}
}

func TestDeleteEmptySeries(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s, seriesID, _, _ := setupFullTestDB(t, db)

	// Delete the chapter to make the series empty
	s.db.Exec("DELETE FROM chapters WHERE series_id = ?", seriesID)

	err := s.DeleteEmptySeries()
	if err != nil {
		t.Fatalf("DeleteEmptySeries failed: %v", err)
	}

	var series *models.Series
	var chapterCount int
	_, chapterCount, err = s.GetSeriesByID(seriesID, 1, 1, 1, "", "", "")
	if err == nil {
		t.Error("Expected error when getting deleted series, but got nil")
	}
	if chapterCount != 0 {
		t.Errorf("Expected 0 chapters in series after deletion, got %d", chapterCount)
	}
	err = s.db.QueryRow("SELECT id FROM series WHERE id = ?", seriesID).Scan(&series)
	if err != sql.ErrNoRows {
		t.Errorf("Expected series with ID %d to be deleted, but it still exists", seriesID)
	}
}

func TestUpdateChapterThumbnail(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s, _, chapterID, _ := setupFullTestDB(t, db)
	newThumbnail := "data:image/jpeg;base64,newthumb"

	err := s.UpdateChapterThumbnail(chapterID, newThumbnail)
	if err != nil {
		t.Fatalf("UpdateChapterThumbnail failed: %v", err)
	}

	chapter, _ := s.GetChapterByID(chapterID, 1)
	if chapter.Thumbnail != newThumbnail {
		t.Errorf("Thumbnail was not updated correctly")
	}
}

func TestUpdateAllSeriesThumbnails(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s, seriesID, _, _ := setupFullTestDB(t, db)

	// Set a custom thumbnail on the first chapter
	firstChapterThumb := "data:image/jpeg;base64,first"
	s.db.Exec("UPDATE chapters SET thumbnail = ? WHERE id = 1", firstChapterThumb)

	err := s.UpdateAllSeriesThumbnails()
	if err != nil {
		t.Fatalf("UpdateAllSeriesThumbnails failed: %v", err)
	}

	series, chapterCount, _ := s.GetSeriesByID(seriesID, 1, 1, 1, "", "", "")
	if series.Thumbnail != firstChapterThumb {
		t.Errorf("Series thumbnail was not updated to first chapter's thumbnail")
	}
	if chapterCount != 2 {
		t.Errorf("Expected 2 chapters in series, got %d", chapterCount)
	}
}

// Helper to set up a more complete DB state for tests
func setupFullTestDB(t *testing.T, db *sql.DB) (*Store, int64, int64, string) {
	t.Helper()
	s := New(db)
	res, _ := db.Exec(`INSERT INTO series (id, title, path, created_at, updated_at) VALUES (1, 'Test Series', '/path/a', ?, ?)`, time.Now(), time.Now())
	seriesID, _ := res.LastInsertId()
	chapterPath := "/path/a/ch1.cbz"
	res, _ = db.Exec(`INSERT INTO chapters (id, series_id, path, page_count, created_at, updated_at) VALUES (1, ?, ?, 20, ?, ?)`, seriesID, chapterPath, time.Now(), time.Now())
	chapterID, _ := res.LastInsertId()
	db.Exec(`INSERT INTO chapters (id, series_id, path, page_count, created_at, updated_at) VALUES (2, ?, ?, 20, ?, ?)`, seriesID, "/path/a/ch2.cbz", time.Now(), time.Now())
	return s, seriesID, chapterID, chapterPath
}

func TestAddChaptersToQueue(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := New(db)
	chapters := []models.ChapterResult{
		{Identifier: "q-ch1", Title: "Chapter 1"},
		{Identifier: "q-ch2", Title: "Chapter 2"},
	}

	err := s.AddChaptersToQueue("Queue Manga", "mockadex", chapters)
	if err != nil {
		t.Fatalf("AddChaptersToQueue failed: %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM download_queue").Scan(&count)
	if count != 2 {
		t.Errorf("Expected 2 items in queue, but found %d", count)
	}
}

func TestSubscribeToSeries(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := New(db)

	subscription, err := s.SubscribeToSeries("Sub Manga", "sub-id-1", "mockadex")
	if err != nil {
		t.Fatalf("SubscribeToSeries failed: %v", err)
	}
	if subscription.SeriesTitle != "Sub Manga" || subscription.SeriesIdentifier != "sub-id-1" || subscription.ProviderID != "mockadex" {
		t.Errorf("Subscription data mismatch: got %+v", subscription)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM subscriptions WHERE series_identifier = 'sub-id-1'").Scan(&count)
	if count != 1 {
		t.Error("Expected 1 item in subscriptions, but found none")
	}

	// Test idempotency (subscribing again should not create a duplicate or error)
	subscription, err = s.SubscribeToSeries("Sub Manga", "sub-id-1", "mockadex")
	if err != nil {
		t.Fatalf("Subscribing to an existing series failed: %v", err)
	}
	if subscription.SeriesTitle != "Sub Manga" || subscription.SeriesIdentifier != "sub-id-1" || subscription.ProviderID != "mockadex" {
		t.Errorf("Subscription data mismatch on second call: got %+v", subscription)
	}
	// Verify no duplicate subscriptions created
	db.QueryRow("SELECT COUNT(*) FROM subscriptions WHERE series_identifier = 'sub-id-1'").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 item in subscriptions after second call, but found %d", count)
	}
}

func TestSeriesSettings(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	s := New(db)

	// Create a test user and series
	_, err := db.Exec(`INSERT INTO users (id, username, password_hash, role, created_at) VALUES (1, 'testuser', 'password', 'user', CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	_, err = db.Exec(`INSERT INTO series (id, title, path, created_at, updated_at) VALUES (1, 'Test Series', '/path/to/series', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatalf("Failed to create test series: %v", err)
	}

	// Test default settings
	settings, err := s.GetSeriesSettings(1, 1)
	if err != nil {
		t.Fatalf("GetSeriesSettings failed: %v", err)
	}
	if settings.SortBy != "auto" {
		t.Errorf("Expected default sort_by to be 'auto', got %s", settings.SortBy)
	}
	if settings.SortDir != "asc" {
		t.Errorf("Expected default sort_dir to be 'asc', got %s", settings.SortDir)
	}

	// Test updating settings
	err = s.UpdateSeriesSettings(1, 1, "path", "desc")
	if err != nil {
		t.Fatalf("UpdateSeriesSettings failed: %v", err)
	}

	settings, err = s.GetSeriesSettings(1, 1)
	if err != nil {
		t.Fatalf("GetSeriesSettings failed after update: %v", err)
	}
	if settings.SortBy != "path" {
		t.Errorf("Expected sort_by to be 'path', got %s", settings.SortBy)
	}
	if settings.SortDir != "desc" {
		t.Errorf("Expected sort_dir to be 'desc', got %s", settings.SortDir)
	}
}
