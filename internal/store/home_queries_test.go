package store_test

import (
	"database/sql"
	"testing"

	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

// setupHomePageTestDB creates a complex DB state for testing all home page sections.
func setupHomePageTestDB(t *testing.T, db *sql.DB) (s *store.Store, user1, user2 *models.User) {
	t.Helper()
	s = store.New(db)

	// Create users
	u1, _ := s.CreateUser("user1", "hash", "user")
	u2, _ := s.CreateUser("user2", "hash", "user")

	// FOLDERS: /Series A/Vol 1, /Series A/Vol 2, /Series B (untouched), /Series C (recent)
	fA, _ := s.CreateFolder("/A", "Series A", nil)
	fAV1, _ := s.CreateFolder("/A/V1", "Vol 1", &fA.ID)
	fAV2, _ := s.CreateFolder("/A/V2", "Vol 2", &fA.ID)
	fB, _ := s.CreateFolder("/B", "Series B", nil)
	fC, _ := s.CreateFolder("/C", "Series C", nil)

	// Create chapters
	chV1_1, _ := s.CreateChapter(fAV1.ID, "/A/V1/ch1.cbz", "h_av1_1", 10, "") // Continue Reading
	s.CreateChapter(fAV1.ID, "/A/V1/ch2.cbz", "h_av1_2", 10, "")              // Next up after ch1
	s.CreateChapter(fAV2.ID, "/A/V2/ch1.cbz", "h_av2_1", 10, "")              // Next up after Vol 1
	s.CreateChapter(fB.ID, "/B/ch1.cbz", "h_b_1", 10, "")                     // Start Reading
	hc1, _ := s.CreateChapter(fC.ID, "/C/ch1.cbz", "h_c_1", 10, "")           // Recently Added

	// Create user progress
	// User 1: Continue Reading Series A, Chapter 1
	s.UpdateChapterProgress(chV1_1.ID, u1.ID, 50, false)
	// User 1: Finished Series B, Chapter 1
	// s.UpdateChapterProgress(hb1.ID, u1.ID, 100, true)
	// User 2: Has progress on Series C, which User 1 hasn't started
	s.UpdateChapterProgress(hc1.ID, u2.ID, 20, false)

	return s, u1, u2
}

func TestHomePageQueries(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s, user1, _ := setupHomePageTestDB(t, db)

	t.Run("Continue Reading", func(t *testing.T) {
		items, err := s.GetContinueReading(user1.ID, 10)
		if err != nil {
			t.Fatalf("GetContinueReading failed: %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("Expected 1 item in Continue Reading, got %d", len(items))
		}
		if *items[0].ChapterID != 1 { // Assuming chV1_1 gets ID 1
			t.Errorf("Expected chapter ID 1, got %d", *items[0].ChapterID)
		}
		// Test that the title is the immediate parent folder name
		if items[0].SeriesTitle != "Vol 1" {
			t.Errorf("Expected folder name 'Vol 1', got '%s'", items[0].SeriesTitle)
		}
	})

	t.Run("Recently Added", func(t *testing.T) {
		items, err := s.GetRecentlyAdded(10)
		if err != nil {
			t.Fatalf("GetRecentlyAdded failed: %v", err)
		}
		if len(items) != 4 {
			t.Fatalf("Expected 4 grouped items in Recently Added, got %d", len(items))
		}
		expectedSeriesIDs := []int64{5, 4, 3, 2}
		expectedNewChapterCounts := []int{1, 1, 1, 2}

		// c/ch1.cbz -> ch1
		// b/ch1.cbz -> ch1
		// a/v2/ch1.cbz -> ch1
		// a/v1/ -> "", since there are two chapters in this folder, the chapter title is empty.
		expectedChapterTitles := []string{"ch1", "ch1", "ch1", ""}
		for index, item := range items {
			if item.SeriesID != expectedSeriesIDs[index] {
				t.Errorf("Expected index: %d, series ID %d, got %d", index, expectedSeriesIDs[index], item.SeriesID)
			}
			if item.NewChapterCount != expectedNewChapterCounts[index] {
				t.Errorf("Expected index: %d, new chapter count to be %d, got %d", index, expectedNewChapterCounts[index], item.NewChapterCount)
			}
			if item.ChapterTitle != expectedChapterTitles[index] {
				t.Errorf("Expected index: %d, chapter title to be %s, got %s", index, expectedChapterTitles[index], item.ChapterTitle)
			}
		}
	})

	t.Run("Next Up", func(t *testing.T) {
		s.UpdateChapterProgress(1, user1.ID, 100, true)

		items, err := s.GetNextUp(user1.ID, 10)
		if err != nil {
			t.Fatalf("GetNextUp failed: %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("Expected 1 item in Next Up, got %d", len(items))
		}
		if *items[0].ChapterID != 2 {
			t.Errorf("Expected next up chapter to be ID 3, got %d", *items[0].ChapterID)
		}
		if items[0].ChapterTitle != "ch2" {
			t.Errorf("Expected next up chapter title to be ch2, got %s", items[0].ChapterTitle)
		}
	})


	t.Run("Start Reading", func(t *testing.T) {
		items, err := s.GetStartReading(user1.ID, 10)
		if err != nil {
			t.Fatalf("GetStartReading failed: %v", err)
		}
		// User1 has progress in Series A tree, so only B and C should appear.
		if len(items) != 2 {
			t.Fatalf("Expected 2 items in Start Reading, got %d", len(items))
		}
		if items[0].SeriesID != 5 { // Ordered by recency
			t.Errorf("Expected first Start Reading series to be ID 5, got %d", items[0].SeriesID)
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
	})
}


func TestGetNextUp(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	user, _ := s.CreateUser("nextup_user", "hash", "user")
	// FOLDERS:
	// /Series A
	//   - /Vol 1
	//     - ch1 (read)
	//     - ch2 (next up)
	//   - /Vol 2
	//     - ch1 (next up after Vol 1)
	fA, _ := s.CreateFolder("/A", "Series A", nil)
	fAV1, _ := s.CreateFolder("/A/V1", "Vol 1", &fA.ID)
	fAV2, _ := s.CreateFolder("/A/V2", "Vol 2", &fA.ID)

	// CHAPTERS
	chV1_1, _ := s.CreateChapter(fAV1.ID, "/A/V1/ch1.cbz", "h_av1_1", 10, "")
	chV1_2, _ := s.CreateChapter(fAV1.ID, "/A/V1/ch2.cbz", "h_av1_2", 10, "")
	chV2_1, _ := s.CreateChapter(fAV2.ID, "/A/V2/ch1.cbz", "h_av2_1", 10, "")

	t.Run("Suggests next chapter in same folder", func(t *testing.T) {
		s.UpdateChapterProgress(chV1_1.ID, user.ID, 100, true) // Finish chapter 1
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
	})

	t.Run("Suggests first chapter in next folder", func(t *testing.T) {
		s.UpdateChapterProgress(chV1_1.ID, user.ID, 100, true)
		s.UpdateChapterProgress(chV1_2.ID, user.ID, 100, true) // Finish chapter 2 (last in Vol 1)

		items, err := s.GetNextUp(user.ID, 10)
		if err != nil {
			t.Fatalf("GetNextUp failed: %v", err)
		}
		// Debug: Print all returned items
		for i, item := range items {
			t.Logf("Item %d: Chapter ID=%d, Title=%s, Series=%s", i, *item.ChapterID, item.ChapterTitle, item.SeriesTitle)
		}
		if len(items) != 1 {
			t.Fatalf("Expected 1 item in Next Up, got %d", len(items))
		}
		t.Logf("Expected chapter ID %d, got %d", chV2_1.ID, *items[0].ChapterID)
		t.Logf("Chapter title: %s", items[0].ChapterTitle)
		t.Logf("Series title: %s", items[0].SeriesTitle)
		if *items[0].ChapterID != chV2_1.ID {
			t.Errorf("Expected next up chapter to be ID %d, but got %d", chV2_1.ID, *items[0].ChapterID)
		}
	})
}
