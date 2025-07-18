// This new test file covers all the data access layer functions.
// It uses an in-memory SQLite database to ensure tests are fast and isolated.

package store_test

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

// Helper to set up a more complete DB state for tests
func setupFullTestDB(t *testing.T, db *sql.DB) (*store.Store, int64, int64, string) {
	t.Helper()
	s := store.New(db)
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
	s := store.New(db)
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
	s := store.New(db)

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

