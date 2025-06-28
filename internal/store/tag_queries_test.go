package store

import (
	"testing"
	"time"

	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestListTagsWithCounts(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := New(db)

	// db.Exec("INSERT INTO series (id, title, path) VALUES (1, 'Series A', '/a')")
	db.Exec(`INSERT INTO series (id, title, path, created_at, updated_at)
			VALUES (1, 'Series A', '/a', datetime('now'), datetime('now'))`)
	db.Exec("INSERT INTO tags (id, name) VALUES (1, 'action'), (2, 'comedy')")
	db.Exec("INSERT INTO series_tags (series_id, tag_id) VALUES (1, 1)")

	tags, err := s.ListTagsWithCounts()
	if err != nil {
		t.Fatalf("ListTagsWithCounts failed: %v", err)
	}

	if len(tags) != 2 {
		t.Fatalf("Expected 2 tags, got %d", len(tags))
	}
	if tags[0].Name != "action" || tags[0].SeriesCount != 1 {
		t.Errorf("Expected 'action (1)', got '%s (%d)'", tags[0].Name, tags[0].SeriesCount)
	}
	if tags[1].Name != "comedy" || tags[1].SeriesCount != 0 {
		t.Errorf("Expected 'comedy (0)', got '%s (%d)'", tags[1].Name, tags[1].SeriesCount)
	}
}

func TestListSeriesByTagID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := New(db)

	db.Exec(`INSERT INTO series (id, title, path, created_at, updated_at) VALUES (1, 'Series A', '/a', ?, ?)`, time.Now(), time.Now())
	db.Exec("INSERT INTO series (id, title, path, created_at, updated_at) VALUES (2, 'Series B', '/b', ?, ?)", time.Now(), time.Now())
	db.Exec("INSERT INTO tags (id, name) VALUES (1, 'action')")
	db.Exec("INSERT INTO series_tags (series_id, tag_id) VALUES (1, 1)")

	t.Run("Get series by tag", func(t *testing.T) {
		series, total, err := s.ListSeriesByTagID(1, 1, 1, 10, "", "", "")
		if err != nil {
			t.Fatalf("ListSeriesByTagID failed: %v", err)
		}
		if total != 1 {
			t.Errorf("Expected 1 series, got %d", total)
		}
		if len(series) != 1 {
			t.Errorf("Expected 1 series, got %d", len(series))
		}
		if series[0].Title != "Series A" {
			t.Errorf("Expected series title 'Series A', got '%s'", series[0].Title)
		}
	})

	t.Run("Get series by tag with search", func(t *testing.T) {
		_, total, _ := s.ListSeriesByTagID(1, 1, 1, 10, "Nonexistent", "title", "asc")
		if total != 0 {
			t.Errorf("Expected 0 series for search 'Nonexistent', got %d", total)
		}

		_, total, _ = s.ListSeriesByTagID(1, 1, 1, 10, "Series A", "title", "asc")
		if total != 1 {
			t.Errorf("Expected 1 series for search 'Series A', got %d", total)
		}
	})
}
