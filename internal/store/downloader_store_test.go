package store_test

import (
	"testing"
	"time"

	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

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

func TestGetDownloadQueue(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	db.Exec(`INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Manga', 'Ch 1', 'id1', 'p1', ?, 'queued'), ('Manga', 'Ch 2', 'id2', 'p1', ?, 'in_progress')`, time.Now(), time.Now())

	items, err := s.GetDownloadQueue()
	if err != nil {
		t.Fatalf("GetDownloadQueue failed: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}
}

func TestGetQueuedDownloadItems(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	db.Exec(`INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Manga', 'Ch 1', 'id1', 'p1', ?, 'queued'), ('Manga', 'Ch 2', 'id2', 'p1', ?, 'in_progress')`, time.Now(), time.Now())

	items, err := s.GetQueuedDownloadItems(5)
	if err != nil {
		t.Fatalf("GetQueuedDownloadItems failed: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("Expected 1 queued item, got %d", len(items))
	}
	if items[0].Status != "queued" {
		t.Errorf("Expected status 'queued', got '%s'", items[0].Status)
	}
}

func TestUpdateQueueItemStatus(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	res, _ := db.Exec(`INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Manga', 'Ch 1', 'id1', 'p1', ?, 'queued')`, time.Now())
	id, _ := res.LastInsertId()

	err := s.UpdateQueueItemStatus(id, "completed", "Done")
	if err != nil {
		t.Fatalf("UpdateQueueItemStatus failed: %v", err)
	}

	var status, message string
	db.QueryRow("SELECT status, message FROM download_queue WHERE id = ?", id).Scan(&status, &message)
	if status != "completed" || message != "Done" {
		t.Errorf("Expected status 'completed' and message 'Done', got '%s' and '%s'", status, message)
	}
}

func TestResetFailedQueueItems(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	res, _ := db.Exec(`INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Manga', 'Ch 1', 'id1', 'p1', ?, 'failed')`, time.Now())
	id, _ := res.LastInsertId()

	err := s.ResetFailedQueueItems()
	if err != nil {
		t.Fatalf("ResetFailedQueueItems failed: %v", err)
	}

	var status string
	db.QueryRow("SELECT status FROM download_queue WHERE id = ?", id).Scan(&status)
	if status != "queued" {
		t.Errorf("Expected status 'queued' after reset, got '%s'", status)
	}
}

func TestDeleteCompletedQueueItems(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	db.Exec(`INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Manga', 'Ch 1', 'id1', 'p1', ?, 'completed')`, time.Now())

	err := s.DeleteCompletedQueueItems()
	if err != nil {
		t.Fatalf("DeleteCompletedQueueItems failed: %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM download_queue").Scan(&count)
	if count != 0 {
		t.Errorf("Expected queue to be empty, but count is %d", count)
	}
}

func TestEmptyQueue(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	db.Exec(`INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Manga', 'Ch 1', 'id1', 'p1', ?, 'queued')`, time.Now())
	db.Exec(`INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Manga', 'Ch 2', 'id2', 'p1', ?, 'failed')`, time.Now())
	db.Exec(`INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Manga', 'Ch 3', 'id3', 'p1', ?, 'in_progress')`, time.Now())
	db.Exec(`INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Manga', 'Ch 4', 'id4', 'p1', ?, 'completed')`, time.Now())

	err := s.EmptyQueue()
	if err != nil {
		t.Fatalf("EmptyQueue failed: %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM download_queue").Scan(&count)
	if count != 2 {
		t.Errorf("Expected 2 items (in_progress, completed) to remain, but count is %d", count)
	}
}

func TestGetDownloadQueueItem(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Add a test item
	_, err := db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status, progress, message) VALUES ('Test Manga', 'Test Chapter', 'test-id', 'test-provider', ?, 'queued', 0, 'Test message')", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// Get the inserted item ID
	var itemID int64
	err = db.QueryRow("SELECT id FROM download_queue WHERE series_title = 'Test Manga'").Scan(&itemID)
	if err != nil {
		t.Fatal(err)
	}

	// Test getting the item
	item, err := s.GetDownloadQueueItem(itemID)
	if err != nil {
		t.Fatalf("GetDownloadQueueItem failed: %v", err)
	}

	if item == nil {
		t.Fatal("Expected item to be returned, got nil")
	}

	if item.ID != itemID {
		t.Errorf("Expected item ID %d, got %d", itemID, item.ID)
	}

	if item.SeriesTitle != "Test Manga" {
		t.Errorf("Expected series title 'Test Manga', got '%s'", item.SeriesTitle)
	}

	if item.ChapterTitle != "Test Chapter" {
		t.Errorf("Expected chapter title 'Test Chapter', got '%s'", item.ChapterTitle)
	}

	if item.Status != "queued" {
		t.Errorf("Expected status 'queued', got '%s'", item.Status)
	}

	if item.Message != "Test message" {
		t.Errorf("Expected message 'Test message', got '%s'", item.Message)
	}

	// Test getting non-existent item
	_, err = s.GetDownloadQueueItem(99999)
	if err == nil {
		t.Error("Expected error when getting non-existent item, got nil")
	}
}

func TestUpdateQueueItemProgress(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Add some test items to the queue
	chapters := []models.ChapterResult{
		{Title: "Chapter 1", Identifier: "ch1"},
		{Title: "Chapter 2", Identifier: "ch2"},
	}
	err := s.AddChaptersToQueue("Test Series", "mockadex", chapters)
	if err != nil {
		t.Fatalf("Failed to add chapters to queue: %v", err)
	}

	// Get a queue item
	items, err := s.GetQueuedDownloadItems(10)
	if err != nil {
		t.Fatalf("Failed to get queued items: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("No queued items found")
	}

	itemID := items[0].ID
	err = s.UpdateQueueItemProgress(itemID, 50)
	if err != nil {
		t.Fatalf("UpdateQueueItemProgress failed: %v", err)
	}

	// Verify progress was updated
	item, err := s.GetDownloadQueueItem(itemID)
	if err != nil {
		t.Fatalf("Failed to get updated item: %v", err)
	}
	if item.Progress != 50 {
		t.Errorf("Expected progress 50, got %d", item.Progress)
	}
}

func TestResetInProgressQueueItems(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Add some test items to the queue
	chapters := []models.ChapterResult{
		{Title: "Chapter 1", Identifier: "ch1"},
		{Title: "Chapter 2", Identifier: "ch2"},
	}
	err := s.AddChaptersToQueue("Test Series", "mockadex", chapters)
	if err != nil {
		t.Fatalf("Failed to add chapters to queue: %v", err)
	}

	// Set an item to in_progress
	items, err := s.GetQueuedDownloadItems(10)
	if err != nil {
		t.Fatalf("Failed to get queued items: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("No queued items found")
	}

	itemID := items[0].ID
	err = s.UpdateQueueItemStatus(itemID, "in_progress", "")
	if err != nil {
		t.Fatalf("Failed to set item to in_progress: %v", err)
	}

	// Reset in progress items
	err = s.ResetInProgressQueueItems()
	if err != nil {
		t.Fatalf("ResetInProgressQueueItems failed: %v", err)
	}

	// Verify item was reset to queued
	item, err := s.GetDownloadQueueItem(itemID)
	if err != nil {
		t.Fatalf("Failed to get reset item: %v", err)
	}
	if item.Status != "queued" {
		t.Errorf("Expected status 'queued', got '%s'", item.Status)
	}
}

func TestPauseAllQueueItems(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Add some test items to the queue
	chapters := []models.ChapterResult{
		{Title: "Chapter 1", Identifier: "ch1"},
		{Title: "Chapter 2", Identifier: "ch2"},
	}
	err := s.AddChaptersToQueue("Test Series", "mockadex", chapters)
	if err != nil {
		t.Fatalf("Failed to add chapters to queue: %v", err)
	}

	err = s.PauseAllQueueItems()
	if err != nil {
		t.Fatalf("PauseAllQueueItems failed: %v", err)
	}

	// Verify all items are paused
	items, err := s.GetDownloadQueue()
	if err != nil {
		t.Fatalf("Failed to get queue items: %v", err)
	}

	for _, item := range items {
		if item.Status != "paused" {
			t.Errorf("Expected status 'paused', got '%s'", item.Status)
		}
	}
}

func TestResumeAllQueueItems(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Clean up first
	db.Exec("DELETE FROM download_queue")

	// Add some test items to the queue
	chapters := []models.ChapterResult{
		{Title: "Chapter 1", Identifier: "ch1"},
		{Title: "Chapter 2", Identifier: "ch2"},
	}
	err := s.AddChaptersToQueue("Test Series", "mockadex", chapters)
	if err != nil {
		t.Fatalf("Failed to add chapters to queue: %v", err)
	}

	// First pause all items
	err = s.PauseAllQueueItems()
	if err != nil {
		t.Fatalf("PauseAllQueueItems failed: %v", err)
	}

	// Then resume all items
	err = s.ResumeAllQueueItems()
	if err != nil {
		t.Fatalf("ResumeAllQueueItems failed: %v", err)
	}

	// Verify all items are queued
	items, err := s.GetDownloadQueue()
	if err != nil {
		t.Fatalf("Failed to get queue items: %v", err)
	}

	for _, item := range items {
		if item.Status != "queued" {
			t.Errorf("Expected status 'queued', got '%s'", item.Status)
		}
	}
}

func TestPauseQueueItem(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Add some test items to the queue
	chapters := []models.ChapterResult{
		{Title: "Chapter 1", Identifier: "ch1"},
		{Title: "Chapter 2", Identifier: "ch2"},
	}
	err := s.AddChaptersToQueue("Test Series", "mockadex", chapters)
	if err != nil {
		t.Fatalf("Failed to add chapters to queue: %v", err)
	}

	items, err := s.GetQueuedDownloadItems(10)
	if err != nil {
		t.Fatalf("Failed to get queued items: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("No queued items found")
	}

	itemID := items[0].ID
	err = s.PauseQueueItem(itemID)
	if err != nil {
		t.Fatalf("PauseQueueItem failed: %v", err)
	}

	// Verify item is paused
	item, err := s.GetDownloadQueueItem(itemID)
	if err != nil {
		t.Fatalf("Failed to get paused item: %v", err)
	}
	if item.Status != "paused" {
		t.Errorf("Expected status 'paused', got '%s'", item.Status)
	}
}

func TestResumeQueueItem(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Add some test items to the queue
	chapters := []models.ChapterResult{
		{Title: "Chapter 1", Identifier: "ch1"},
		{Title: "Chapter 2", Identifier: "ch2"},
	}
	err := s.AddChaptersToQueue("Test Series", "mockadex", chapters)
	if err != nil {
		t.Fatalf("Failed to add chapters to queue: %v", err)
	}

	// First pause an item
	items, err := s.GetQueuedDownloadItems(10)
	if err != nil {
		t.Fatalf("Failed to get queued items: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("No queued items found")
	}

	itemID := items[0].ID
	err = s.PauseQueueItem(itemID)
	if err != nil {
		t.Fatalf("PauseQueueItem failed: %v", err)
	}

	// Then resume the item
	err = s.ResumeQueueItem(itemID)
	if err != nil {
		t.Fatalf("ResumeQueueItem failed: %v", err)
	}

	// Verify item is queued
	item, err := s.GetDownloadQueueItem(itemID)
	if err != nil {
		t.Fatalf("Failed to get resumed item: %v", err)
	}
	if item.Status != "queued" {
		t.Errorf("Expected status 'queued', got '%s'", item.Status)
	}
}

func TestRetryQueueItem(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Add a failed item to the queue
	_, err := db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status, progress) VALUES ('Test Manga', 'Test Chapter', 'test-id', 'test-provider', ?, 'failed', 75)", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// Get the inserted item ID
	var itemID int64
	err = db.QueryRow("SELECT id FROM download_queue WHERE series_title = 'Test Manga'").Scan(&itemID)
	if err != nil {
		t.Fatal(err)
	}

	// Test retrying the item
	err = s.RetryQueueItem(itemID)
	if err != nil {
		t.Fatalf("RetryQueueItem failed: %v", err)
	}

	// Verify the item was retried in the database
	var status string
	var progress int
	var message string
	err = db.QueryRow("SELECT status, progress, message FROM download_queue WHERE id = ?", itemID).Scan(&status, &progress, &message)
	if err != nil {
		t.Fatal(err)
	}

	if status != "queued" {
		t.Errorf("Expected status 'queued', got '%s'", status)
	}

	if progress != 0 {
		t.Errorf("Expected progress 0, got %d", progress)
	}

	if message != "Re-queued for retry by user" {
		t.Errorf("Expected message 'Re-queued for retry by user', got '%s'", message)
	}
}

func TestRetryQueueItemWithNonFailedStatus(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Add a queued item (not failed) to the queue
	_, err := db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Test Manga', 'Test Chapter', 'test-id', 'test-provider', ?, 'queued')", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// Get the inserted item ID
	var itemID int64
	err = db.QueryRow("SELECT id FROM download_queue WHERE series_title = 'Test Manga'").Scan(&itemID)
	if err != nil {
		t.Fatal(err)
	}

	// Test retrying a non-failed item should fail
	err = s.RetryQueueItem(itemID)
	if err == nil {
		t.Error("Expected error when retrying non-failed item, got nil")
	}
}

func TestRetryQueueItemWithNonExistentItem(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Test retrying a non-existent item
	err := s.RetryQueueItem(99999)
	if err == nil {
		t.Error("Expected error when retrying non-existent item, got nil")
	}
}

func TestDeleteQueueItem(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Add some test items to the queue
	chapters := []models.ChapterResult{
		{Title: "Chapter 1", Identifier: "ch1"},
		{Title: "Chapter 2", Identifier: "ch2"},
	}
	err := s.AddChaptersToQueue("Test Series", "mockadex", chapters)
	if err != nil {
		t.Fatalf("Failed to add chapters to queue: %v", err)
	}

	items, err := s.GetDownloadQueue()
	if err != nil {
		t.Fatalf("Failed to get queue items: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("No queue items found")
	}

	itemID := items[0].ID
	err = s.DeleteQueueItem(itemID)
	if err != nil {
		t.Fatalf("DeleteQueueItem failed: %v", err)
	}

	// Verify item was deleted
	_, err = s.GetDownloadQueueItem(itemID)
	if err == nil {
		t.Error("Expected item to be deleted")
	}
}

func TestGetChapterIdentifiersInQueue(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Add some items to the queue
	chapters := []models.ChapterResult{
		{Title: "Chapter 1", Identifier: "ch1"},
		{Title: "Chapter 2", Identifier: "ch2"},
	}
	err := s.AddChaptersToQueue("Test Series", "mockadex", chapters)
	if err != nil {
		t.Fatalf("Failed to add chapters to queue: %v", err)
	}

	identifiers, err := s.GetChapterIdentifiersInQueue("Test Series", "mockadex")
	if err != nil {
		t.Fatalf("GetChapterIdentifiersInQueue failed: %v", err)
	}

	if len(identifiers) != 2 {
		t.Errorf("Expected 2 identifiers, got %d", len(identifiers))
	}

	// Check that we have the expected identifiers
	expected := map[string]bool{"ch1": true, "ch2": true}
	foundExpected := 0
	for _, id := range identifiers {
		if expected[id] {
			foundExpected++
		}
	}
	if foundExpected != 2 {
		t.Errorf("Expected to find 2 expected identifiers, found %d", foundExpected)
	}
}
