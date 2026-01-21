package store_test

import (
	"testing"

	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestGetContinueReading(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Create user
	user, _ := s.CreateUser("user1", "hash", "user")

	// Create folder structure: Series A -> Vol 1
	fA, _ := s.CreateFolder("/A", "Series A", nil)
	fAV1, _ := s.CreateFolder("/A/V1", "Vol 1", &fA.ID)

	// Create chapter with progress
	chV1_1, _ := s.CreateChapter(fAV1.ID, "/A/V1/ch1.cbz", "h_av1_1", 10, "")
	s.UpdateChapterProgress(chV1_1.ID, user.ID, 50, false)

	items, err := s.GetContinueReading(user.ID, 10)
	if err != nil {
		t.Fatalf("GetContinueReading failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 item in Continue Reading, got %d", len(items))
	}
	if *items[0].ChapterID != chV1_1.ID {
		t.Errorf("Expected chapter ID %d, got %d", chV1_1.ID, *items[0].ChapterID)
	}
	if items[0].SeriesTitle != "Vol 1" {
		t.Errorf("Expected folder name 'Vol 1', got '%s'", items[0].SeriesTitle)
	}
}

func TestGetRecentlyAddedWithDateGrouping(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Create user
	user, _ := s.CreateUser("user1", "hash", "user")

	// Create folders
	fA, _ := s.CreateFolder("/A", "Series A", nil)
	fB, _ := s.CreateFolder("/B", "Series B", nil)

	// Create chapters with different creation times by manually inserting them
	// This simulates chapters added on different days
	yesterday := "2024-01-01 10:00:00"
	today := "2024-01-02 10:00:00"

	// Insert chapters with specific creation times
	res, _ := db.Exec("INSERT INTO chapters (folder_id, path, content_hash, page_count, thumbnail, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		fA.ID, "/A/ch1.cbz", "hash_a1", 10, "", yesterday, yesterday)
	chA1ID, _ := res.LastInsertId()
	res, _ = db.Exec("INSERT INTO chapters (folder_id, path, content_hash, page_count, thumbnail, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		fA.ID, "/A/ch2.cbz", "hash_a2", 10, "", yesterday, yesterday) // Same day as ch1
	chA2ID, _ := res.LastInsertId()
	res, _ = db.Exec("INSERT INTO chapters (folder_id, path, content_hash, page_count, thumbnail, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		fA.ID, "/A/ch3.cbz", "hash_a3", 10, "", today, today) // Different day
	chA3ID, _ := res.LastInsertId()
	db.Exec("INSERT INTO chapters (folder_id, path, content_hash, page_count, thumbnail, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		fB.ID, "/B/ch1.cbz", "hash_b1", 10, "", today, today) // Same day as ch3

	// Mark some chapters with progress to test progress display
	s.UpdateChapterProgress(chA1ID, user.ID, 50, false) // 50% progress on chA1
	s.UpdateChapterProgress(chA2ID, user.ID, 100, true) // 100% progress (read) on chA2
	s.UpdateChapterProgress(chA3ID, user.ID, 30, false) // 30% progress on chA3
	// chB1 has no progress

	items, err := s.GetRecentlyAdded(user.ID, 10)
	if err != nil {
		t.Fatalf("GetRecentlyAdded failed: %v", err)
	}

	// Should have 3 items:
	// 1. Series B (today) - 1 chapter
	// 2. Series A (today) - 1 chapter
	// 3. Series A (yesterday) - 2 chapters grouped together
	if len(items) != 3 {
		t.Fatalf("Expected 3 grouped items in Recently Added, got %d", len(items))
	}

	// Check the grouping behavior and progress display
	var seriesAToday, seriesAYesterday, seriesBToday bool
	for _, item := range items {
		if item.SeriesTitle == "Series A" {
			if item.NewChapterCount == 1 && item.ChapterID != nil {
				seriesAToday = true // Single chapter added today (chA3)
				// This is a chapter card, should have chapter progress (30%)
				if item.ProgressPercent == nil {
					t.Error("Expected Series A chapter card (today) to have chapter progress")
				} else if *item.ProgressPercent != 30 {
					t.Errorf("Expected Series A chapter card (today) to have 30%% progress, got %d%%", *item.ProgressPercent)
				}
			} else if item.NewChapterCount == 2 && item.ChapterID == nil {
				seriesAYesterday = true // Two chapters grouped from yesterday (series card)
				// This is a series card, should NOT have progress
				if item.ProgressPercent != nil {
					t.Error("Expected Series A series card (yesterday) to NOT have progress")
				}
			}
		} else if item.SeriesTitle == "Series B" && item.NewChapterCount == 1 && item.ChapterID != nil {
			seriesBToday = true // Single chapter added today (chB1)
			// This is a chapter card, should have chapter progress (0% - no progress set)
			if item.ProgressPercent != nil {
				t.Errorf("Expected Series B chapter card to have no progress (nil), got %d%%", *item.ProgressPercent)
			}
		}
	}

	if !seriesAToday {
		t.Error("Expected Series A to have 1 chapter from today")
	}
	if !seriesAYesterday {
		t.Error("Expected Series A to have 2 chapters grouped from yesterday")
	}
	if !seriesBToday {
		t.Error("Expected Series B to have 1 chapter from today")
	}
}

func TestGetNextUpSameFolder(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Create user
	user, _ := s.CreateUser("nextup_user", "hash", "user")

	// Create folder structure: Series A -> Vol 1
	fA, _ := s.CreateFolder("/A", "Series A", nil)
	fAV1, _ := s.CreateFolder("/A/V1", "Vol 1", &fA.ID)

	// Create chapters
	chV1_1, _ := s.CreateChapter(fAV1.ID, "/A/V1/ch1.cbz", "h_av1_1", 10, "")
	chV1_2, _ := s.CreateChapter(fAV1.ID, "/A/V1/ch2.cbz", "h_av1_2", 10, "")

	// Mark first chapter as read
	s.UpdateChapterProgress(chV1_1.ID, user.ID, 100, true)

	items, err := s.GetNextUp(user.ID, 10)
	if err != nil {
		t.Fatalf("GetNextUp failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 item in Next Up, got %d", len(items))
	}
	if *items[0].ChapterID != chV1_2.ID {
		t.Errorf("Expected next up chapter to be ID %d, but got %d", chV1_2.ID, *items[0].ChapterID)
	}
}

func TestGetNextUpNextFolder(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Create user
	user, _ := s.CreateUser("nextup_user", "hash", "user")

	// Create folder structure: Series A -> Vol 1, Vol 2
	fA, _ := s.CreateFolder("/A", "Series A", nil)
	fAV1, _ := s.CreateFolder("/A/V1", "Vol 1", &fA.ID)
	fAV2, _ := s.CreateFolder("/A/V2", "Vol 2", &fA.ID)

	// Create chapters
	chV1_1, _ := s.CreateChapter(fAV1.ID, "/A/V1/ch1.cbz", "h_av1_1", 10, "")
	chV1_2, _ := s.CreateChapter(fAV1.ID, "/A/V1/ch2.cbz", "h_av1_2", 10, "")
	chV2_1, _ := s.CreateChapter(fAV2.ID, "/A/V2/ch1.cbz", "h_av2_1", 10, "")

	// Mark both chapters in Vol 1 as read
	s.UpdateChapterProgress(chV1_1.ID, user.ID, 100, true)
	s.UpdateChapterProgress(chV1_2.ID, user.ID, 100, true)

	items, err := s.GetNextUp(user.ID, 10)
	if err != nil {
		t.Fatalf("GetNextUp failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 item in Next Up, got %d", len(items))
	}
	if *items[0].ChapterID != chV2_1.ID {
		t.Errorf("Expected next up chapter to be ID %d, but got %d", chV2_1.ID, *items[0].ChapterID)
	}
}

func TestGetStartReading(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Create user
	user, _ := s.CreateUser("user1", "hash", "user")

	// Create folders
	fA, _ := s.CreateFolder("/A", "Series A", nil)
	fAV1, _ := s.CreateFolder("/A/V1", "Vol 1", &fA.ID)
	fB, _ := s.CreateFolder("/B", "Series B", nil)
	fC, _ := s.CreateFolder("/C", "Series C", nil)

	// Create chapters
	chV1_1, _ := s.CreateChapter(fAV1.ID, "/A/V1/ch1.cbz", "h_av1_1", 10, "")
	s.CreateChapter(fB.ID, "/B/ch1.cbz", "h_b_1", 10, "")
	s.CreateChapter(fC.ID, "/C/ch1.cbz", "h_c_1", 10, "")

	// User has progress in Series A tree
	s.UpdateChapterProgress(chV1_1.ID, user.ID, 50, false)

	items, err := s.GetStartReading(user.ID, 10)
	if err != nil {
		t.Fatalf("GetStartReading failed: %v", err)
	}
	// User1 has progress in Series A tree, so only B and C should appear
	if len(items) != 2 {
		t.Fatalf("Expected 2 items in Start Reading, got %d", len(items))
	}

	var foundB, foundC bool
	for _, item := range items {
		if item.SeriesTitle == "Series B" {
			foundB = true
		}
		if item.SeriesTitle == "Series C" {
			foundC = true
		}
	}
	if !foundB || !foundC {
		t.Error("Expected to find Series B and Series C in Start Reading")
	}
}

func TestGetStartReadingNoProgress(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Create user
	user, _ := s.CreateUser("user1", "hash", "user")

	// Create folders
	fA, _ := s.CreateFolder("/A", "Series A", nil)
	fB, _ := s.CreateFolder("/B", "Series B", nil)
	fC, _ := s.CreateFolder("/C", "Series C", nil)

	// Create chapters
	s.CreateChapter(fA.ID, "/A/ch1.cbz", "h_a_1", 10, "")
	s.CreateChapter(fB.ID, "/B/ch1.cbz", "h_b_1", 10, "")
	s.CreateChapter(fC.ID, "/C/ch1.cbz", "h_c_1", 10, "")

	items, err := s.GetStartReading(user.ID, 10)
	if err != nil {
		t.Fatalf("GetStartReading failed: %v", err)
	}
	// User has no progress, so all series should appear
	if len(items) != 3 {
		t.Fatalf("Expected 3 items in Start Reading, got %d", len(items))
	}

	var foundA, foundB, foundC bool
	for _, item := range items {
		if item.SeriesTitle == "Series A" {
			foundA = true
		}
		if item.SeriesTitle == "Series B" {
			foundB = true
		}
		if item.SeriesTitle == "Series C" {
			foundC = true
		}
	}
	if !foundA || !foundB || !foundC {
		t.Error("Expected to find all series in Start Reading")
	}
}
