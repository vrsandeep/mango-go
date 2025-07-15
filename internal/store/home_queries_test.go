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

	// Create series
	// db.Exec(`INSERT INTO series (id, title, path, created_at, updated_at) VALUES (1, 'Series A', '/a', ?, ?)`, time.Now().Add(-5*time.Hour), time.Now().Add(-5*time.Hour))
	// db.Exec(`INSERT INTO series (id, title, path, created_at, updated_at) VALUES (2, 'Series B', '/b', ?, ?)`, time.Now().Add(-4*time.Hour), time.Now().Add(-4*time.Hour))
	// db.Exec(`INSERT INTO series (id, title, path, created_at, updated_at) VALUES (3, 'Series C', '/c', ?, ?)`, time.Now().Add(-3*time.Hour), time.Now().Add(-3*time.Hour))
	// db.Exec(`INSERT INTO series (id, title, path, created_at, updated_at) VALUES (4, 'Series D', '/d', ?, ?)`, time.Now().Add(-2*time.Hour), time.Now().Add(-2*time.Hour))

	// FOLDERS: /Series A/Vol 1, /Series A/Vol 2, /Series B (untouched), /Series C (recent)
	fA, _ := s.CreateFolder("/A", "Series A", nil)
	fAV1, _ := s.CreateFolder("/A/V1", "Vol 1", &fA.ID)
	fAV2, _ := s.CreateFolder("/A/V2", "Vol 2", &fA.ID)
	fB, _ := s.CreateFolder("/B", "Series B", nil)
	fC, _ := s.CreateFolder("/C", "Series C", nil)

	// Create chapters
	// db.Exec(`INSERT INTO chapters (id, series_id, path, page_count, created_at, updated_at) VALUES (1, 1, 'A-1', 2, ?, ?)`, time.Now().Add(-5*time.Hour), time.Now().Add(-5*time.Hour)) // Continue Reading
	// db.Exec(`INSERT INTO chapters (id, series_id, path, page_count, created_at, updated_at) VALUES (2, 2, 'B-1', 2, ?, ?)`, time.Now().Add(-4*time.Hour), time.Now().Add(-4*time.Hour)) // Next Up (read)
	// db.Exec(`INSERT INTO chapters (id, series_id, path, page_count, created_at, updated_at) VALUES (3, 2, 'B-2', 2, ?, ?)`, time.Now().Add(-4*time.Hour), time.Now().Add(-4*time.Hour)) // Next Up (next)
	// db.Exec(`INSERT INTO chapters (id, series_id, path, page_count, created_at, updated_at) VALUES (4, 3, 'C-1', 2, ?, ?)`, time.Now().Add(-1*time.Hour), time.Now().Add(-1*time.Hour)) // Recently Added
	// db.Exec(`INSERT INTO chapters (id, series_id, path, page_count, created_at, updated_at) VALUES (5, 3, 'C-2', 2, ?, ?)`, time.Now(), time.Now())                                     // Recently Added
	// // Series 4 has no chapters read by user1 -> Start Reading

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

	// t.Run("Next Up", func(t *testing.T) {
	// 	items, err := s.GetNextUp(user1.ID, 10)
	// 	if err != nil {
	// 		t.Fatalf("GetNextUp failed: %v", err)
	// 	}
	// 	if len(items) != 1 {
	// 		t.Fatalf("Expected 1 item in Next Up, got %d", len(items))
	// 	}
	// 	if *items[0].ChapterID != 3 {
	// 		t.Errorf("Expected next up chapter to be ID 3, got %d", *items[0].ChapterID)
	// 	}
	// 	if items[0].ChapterTitle != "B-2" {
	// 		t.Errorf("Expected next up chapter title to be B-2, got %s", items[0].ChapterTitle)
	// 	}
	// })

	t.Run("Recently Added", func(t *testing.T) {
		items, err := s.GetRecentlyAdded(10)
		if err != nil {
			t.Fatalf("GetRecentlyAdded failed: %v", err)
		}
		// Series D has no chapters at all. Hence, it is not included in the results.
		if len(items) != 3 {
			t.Fatalf("Expected 3 grouped items in Recently Added, got %d", len(items))
		}
		expectedSeriesIDs := []int64{3, 2, 1}
		expectedNewChapterCounts := []int{2, 2, 1}
		expectedChapterTitles := []string{"", "", "A-1"} // The first two series id have two chapters, so the chapter title is empty.
		for index, item := range items {
			if item.SeriesID != expectedSeriesIDs[index] {
				t.Errorf("Expected series ID %d, got %d", expectedSeriesIDs[index], item.SeriesID)
			}
			if item.NewChapterCount != expectedNewChapterCounts[index] {
				t.Errorf("Expected new chapter count to be %d, got %d", expectedNewChapterCounts[index], item.NewChapterCount)
			}
			if item.ChapterTitle != expectedChapterTitles[index] {
				t.Errorf("Expected chapter title to be %s, got %s", expectedChapterTitles[index], item.ChapterTitle)
			}
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
		// if items[0].SeriesID != 4 { // Ordered by recency
		// 	t.Errorf("Expected first Start Reading series to be ID 4, got %d", items[0].SeriesID)
		// }

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

func setupNextUpTestDB(t *testing.T, s *store.Store, user *models.User) {
	t.Helper()

	// PROGRESS: User has read the first chapter of Vol 1.
	// s.UpdateChapterProgress(user.ID, chV1_1.ID, 100, true)
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
		s.UpdateChapterProgress(user.ID, chV1_1.ID, 100, true) // Finish chapter 1
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
		s.UpdateChapterProgress(user.ID, chV1_1.ID, 100, true)
		s.UpdateChapterProgress(user.ID, chV1_2.ID, 100, true) // Finish chapter 2 (last in Vol 1)

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
	})
}
