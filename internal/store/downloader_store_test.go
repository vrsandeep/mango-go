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
	db.Exec(`INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Manga', 'Ch 1', 'id1', 'p1', ?, 'failed')`, time.Now())

	err := s.ResetFailedQueueItems()
	if err != nil {
		t.Fatalf("ResetFailedQueueItems failed: %v", err)
	}

	var status string
	db.QueryRow("SELECT status FROM download_queue WHERE id = 1").Scan(&status)
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
