package store_test

import (
	"database/sql"
	"testing"
	"time"

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
	db.Exec(`INSERT INTO series (id, title, path, created_at, updated_at) VALUES (1, 'Series A', '/a', ?, ?)`, time.Now().Add(-5*time.Hour), time.Now().Add(-5*time.Hour))
	db.Exec(`INSERT INTO series (id, title, path, created_at, updated_at) VALUES (2, 'Series B', '/b', ?, ?)`, time.Now().Add(-4*time.Hour), time.Now().Add(-4*time.Hour))
	db.Exec(`INSERT INTO series (id, title, path, created_at, updated_at) VALUES (3, 'Series C', '/c', ?, ?)`, time.Now().Add(-3*time.Hour), time.Now().Add(-3*time.Hour))
	db.Exec(`INSERT INTO series (id, title, path, created_at, updated_at) VALUES (4, 'Series D', '/d', ?, ?)`, time.Now().Add(-2*time.Hour), time.Now().Add(-2*time.Hour))

	// Create chapters
	db.Exec(`INSERT INTO chapters (id, series_id, path, page_count, created_at, updated_at) VALUES (1, 1, 'A-1', 2, ?, ?)`, time.Now().Add(-5*time.Hour), time.Now().Add(-5*time.Hour)) // Continue Reading
	db.Exec(`INSERT INTO chapters (id, series_id, path, page_count, created_at, updated_at) VALUES (2, 2, 'B-1', 2, ?, ?)`, time.Now().Add(-4*time.Hour), time.Now().Add(-4*time.Hour)) // Next Up (read)
	db.Exec(`INSERT INTO chapters (id, series_id, path, page_count, created_at, updated_at) VALUES (3, 2, 'B-2', 2, ?, ?)`, time.Now().Add(-4*time.Hour), time.Now().Add(-4*time.Hour)) // Next Up (next)
	db.Exec(`INSERT INTO chapters (id, series_id, path, page_count, created_at, updated_at) VALUES (4, 3, 'C-1', 2, ?, ?)`, time.Now().Add(-1*time.Hour), time.Now().Add(-1*time.Hour)) // Recently Added
	db.Exec(`INSERT INTO chapters (id, series_id, path, page_count, created_at, updated_at) VALUES (5, 3, 'C-2', 2, ?, ?)`, time.Now(), time.Now())                                     // Recently Added
	// Series 4 has no chapters read by user1 -> Start Reading

	// Create user progress
	// User 1: Continue Reading Series A, Chapter 1
	s.UpdateChapterProgress(1, u1.ID, 50, false)
	// User 1: Finished Series B, Chapter 1
	s.UpdateChapterProgress(2, u1.ID, 100, true)
	// User 2: Has progress on Series C, which User 1 hasn't started
	s.UpdateChapterProgress(4, u2.ID, 20, false)

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
		if *items[0].ChapterID != 1 {
			t.Errorf("Expected chapter ID 1, got %d", *items[0].ChapterID)
		}
	})

	t.Run("Next Up", func(t *testing.T) {
		items, err := s.GetNextUp(user1.ID, 10)
		if err != nil {
			t.Fatalf("GetNextUp failed: %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("Expected 1 item in Next Up, got %d", len(items))
		}
		if *items[0].ChapterID != 3 {
			t.Errorf("Expected next up chapter to be ID 3, got %d", *items[0].ChapterID)
		}
		if items[0].ChapterTitle != "B-2" {
			t.Errorf("Expected next up chapter title to be B-2, got %s", items[0].ChapterTitle)
		}
	})

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
		// Should be Series C and D
		if len(items) != 2 {
			t.Fatalf("Expected 2 items in Start Reading, got %d", len(items))
		}
		if items[0].SeriesID != 4 { // Ordered by recency
			t.Errorf("Expected first Start Reading series to be ID 4, got %d", items[0].SeriesID)
		}
	})
}
