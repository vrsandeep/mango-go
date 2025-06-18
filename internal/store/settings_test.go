package store_test

import (
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
	"testing"
	"time"
)

func TestSeriesSettings(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Create a dummy series
	res, err := db.Exec(
		`INSERT INTO series (id, title, path, created_at, updated_at) VALUES (1, 'Series B', '/path/b', ?, ?)`,
		time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert series: %v", err)
	}
	seriesID, _ := res.LastInsertId()

	// Test getting default settings
	settings, err := s.GetSeriesSettings(seriesID)
	if err != nil {
		t.Fatalf("GetSeriesSettings failed for new series: %v", err)
	}
	if settings.SortBy != "auto" || settings.SortDir != "asc" {
		t.Errorf("Expected default settings, but got %+v", settings)
	}

	// Test updating settings
	err = s.UpdateSeriesSettings(seriesID, "path", "desc")
	if err != nil {
		t.Fatalf("UpdateSeriesSettings failed: %v", err)
	}

	// Test getting updated settings
	newSettings, err := s.GetSeriesSettings(seriesID)
	if err != nil {
		t.Fatalf("GetSeriesSettings failed after update: %v", err)
	}
	if newSettings.SortBy != "path" || newSettings.SortDir != "desc" {
		t.Errorf("Expected updated settings, but got %+v", newSettings)
	}
}
