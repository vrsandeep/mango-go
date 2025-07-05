package store_test

import (
	"testing"
	"time"

	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

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
