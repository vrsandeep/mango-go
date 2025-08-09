package downloader_test

import (
	"testing"
	"time"

	"github.com/vrsandeep/mango-go/internal/downloader"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestPauseQueueItem(t *testing.T) {
	app := testutil.SetupTestApp(t)
	db := app.DB()
	st := store.New(db)

	// Add a test item to the queue
	_, err := db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status, progress) VALUES ('Test Manga', 'Test Chapter', 'test-id', 'test-provider', ?, 'in_progress', 50)", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// Get the inserted item ID
	var itemID int64
	err = db.QueryRow("SELECT id FROM download_queue WHERE series_title = 'Test Manga'").Scan(&itemID)
	if err != nil {
		t.Fatal(err)
	}

	// Test pausing the item
	err = downloader.PauseQueueItem(app, st, itemID)
	if err != nil {
		t.Fatalf("PauseQueueItem failed: %v", err)
	}

	// Verify the item was paused in the database
	var status string
	var progress int
	err = db.QueryRow("SELECT status, progress FROM download_queue WHERE id = ?", itemID).Scan(&status, &progress)
	if err != nil {
		t.Fatal(err)
	}

	if status != "paused" {
		t.Errorf("Expected status 'paused', got '%s'", status)
	}

	if progress != 50 {
		t.Errorf("Expected progress 50, got %d", progress)
	}
}

func TestResumeQueueItem(t *testing.T) {
	app := testutil.SetupTestApp(t)
	db := app.DB()
	st := store.New(db)

	// Add a paused test item to the queue
	_, err := db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status, progress) VALUES ('Test Manga', 'Test Chapter', 'test-id', 'test-provider', ?, 'paused', 75)", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// Get the inserted item ID
	var itemID int64
	err = db.QueryRow("SELECT id FROM download_queue WHERE series_title = 'Test Manga'").Scan(&itemID)
	if err != nil {
		t.Fatal(err)
	}

	// Test resuming the item
	err = downloader.ResumeQueueItem(app, st, itemID)
	if err != nil {
		t.Fatalf("ResumeQueueItem failed: %v", err)
	}

	// Verify the item was resumed in the database
	var status string
	var progress int
	err = db.QueryRow("SELECT status, progress FROM download_queue WHERE id = ?", itemID).Scan(&status, &progress)
	if err != nil {
		t.Fatal(err)
	}

	if status != "queued" {
		t.Errorf("Expected status 'queued', got '%s'", status)
	}

	if progress != 75 {
		t.Errorf("Expected progress 75, got %d", progress)
	}
}

func TestPauseQueueItemWithNonExistentItem(t *testing.T) {
	app := testutil.SetupTestApp(t)
	db := app.DB()
	st := store.New(db)

	// Test pausing a non-existent item
	err := downloader.PauseQueueItem(app, st, 99999)
	if err == nil {
		t.Error("Expected error when pausing non-existent item, got nil")
	}
}

func TestResumeQueueItemWithNonExistentItem(t *testing.T) {
	app := testutil.SetupTestApp(t)
	db := app.DB()
	st := store.New(db)

	// Test resuming a non-existent item
	err := downloader.ResumeQueueItem(app, st, 99999)
	if err == nil {
		t.Error("Expected error when resuming non-existent item, got nil")
	}
}

func TestPauseAndResumeDownloads(t *testing.T) {
	// Test global pause/resume functions
	if downloader.IsPaused() {
		t.Error("Expected downloads to not be paused initially")
	}

	downloader.PauseDownloads()
	if !downloader.IsPaused() {
		t.Error("Expected downloads to be paused after PauseDownloads()")
	}

	downloader.ResumeDownloads()
	if downloader.IsPaused() {
		t.Error("Expected downloads to not be paused after ResumeDownloads()")
	}
}

func TestSendDownloaderProgressUpdate(t *testing.T) {
	// This is a helper function, so we'll test it indirectly through the main functions
	// The actual WebSocket broadcasting is tested through integration tests
	// This test ensures the function doesn't panic
	app := testutil.SetupTestApp(t)
	db := app.DB()
	st := store.New(db)

	// Add a test item
	_, err := db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Test Manga', 'Test Chapter', 'test-id', 'test-provider', ?, 'queued')", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	var itemID int64
	err = db.QueryRow("SELECT id FROM download_queue WHERE series_title = 'Test Manga'").Scan(&itemID)
	if err != nil {
		t.Fatal(err)
	}

	// Test that the function doesn't panic
	// Note: In a real test environment, we might want to mock the WebSocket hub
	// For now, we just ensure the function can be called without errors
	err = downloader.PauseQueueItem(app, st, itemID)
	if err != nil {
		t.Fatalf("PauseQueueItem failed: %v", err)
	}
}

func TestWorkerPauseCheck(t *testing.T) {
	app := testutil.SetupTestApp(t)
	db := app.DB()
	st := store.New(db)

	// Add a test item that's in progress
	_, err := db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status, progress) VALUES ('Test Manga', 'Test Chapter', 'test-id', 'test-provider', ?, 'in_progress', 25)", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	var itemID int64
	err = db.QueryRow("SELECT id FROM download_queue WHERE series_title = 'Test Manga'").Scan(&itemID)
	if err != nil {
		t.Fatal(err)
	}

	// Pause the item
	err = downloader.PauseQueueItem(app, st, itemID)
	if err != nil {
		t.Fatalf("PauseQueueItem failed: %v", err)
	}

	// Verify the item is paused
	var status string
	err = db.QueryRow("SELECT status FROM download_queue WHERE id = ?", itemID).Scan(&status)
	if err != nil {
		t.Fatal(err)
	}

	if status != "paused" {
		t.Errorf("Expected status 'paused', got '%s'", status)
	}
}

func TestErrorConstants(t *testing.T) {
	// Test that our error constant is properly defined
	if downloader.ErrDownloadPaused == nil {
		t.Error("errDownloadPaused should not be nil")
	}

	expectedMsg := "download paused by user"
	if downloader.ErrDownloadPaused.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, downloader.ErrDownloadPaused.Error())
	}
}
