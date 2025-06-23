package store

import (
	"testing"
	"time"

	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestGetDownloadQueue(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := New(db)
	s.db.Exec(`INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Manga', 'Ch 1', 'id1', 'p1', ?, 'queued'), ('Manga', 'Ch 2', 'id2', 'p1', ?, 'in_progress')`, time.Now(), time.Now())

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
	s := New(db)
	s.db.Exec(`INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Manga', 'Ch 1', 'id1', 'p1', ?, 'queued'), ('Manga', 'Ch 2', 'id2', 'p1', ?, 'in_progress')`, time.Now(), time.Now())

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
	s := New(db)
	res, _ := s.db.Exec(`INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Manga', 'Ch 1', 'id1', 'p1', ?, 'queued')`, time.Now())
	id, _ := res.LastInsertId()

	err := s.UpdateQueueItemStatus(id, "completed", "Done")
	if err != nil {
		t.Fatalf("UpdateQueueItemStatus failed: %v", err)
	}

	var status, message string
	s.db.QueryRow("SELECT status, message FROM download_queue WHERE id = ?", id).Scan(&status, &message)
	if status != "completed" || message != "Done" {
		t.Errorf("Expected status 'completed' and message 'Done', got '%s' and '%s'", status, message)
	}
}

func TestResetFailedQueueItems(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := New(db)
	s.db.Exec(`INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Manga', 'Ch 1', 'id1', 'p1', ?, 'failed')`, time.Now())

	err := s.ResetFailedQueueItems()
	if err != nil {
		t.Fatalf("ResetFailedQueueItems failed: %v", err)
	}

	var status string
	s.db.QueryRow("SELECT status FROM download_queue WHERE id = 1").Scan(&status)
	if status != "queued" {
		t.Errorf("Expected status 'queued' after reset, got '%s'", status)
	}
}

func TestDeleteCompletedQueueItems(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := New(db)
	s.db.Exec(`INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Manga', 'Ch 1', 'id1', 'p1', ?, 'completed')`, time.Now())

	err := s.DeleteCompletedQueueItems()
	if err != nil {
		t.Fatalf("DeleteCompletedQueueItems failed: %v", err)
	}

	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM download_queue").Scan(&count)
	if count != 0 {
		t.Errorf("Expected queue to be empty, but count is %d", count)
	}
}
