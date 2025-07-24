package store_test

import (
	"testing"
	"time"

	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestFolderSettings(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Create a dummy user for testing
	userID := int64(1)
	_, err := db.Exec(`INSERT INTO users (id, username, password_hash, role, created_at) VALUES (?, ?, ?, ?, ?)`,
		userID, "testuser", "password", "user", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	folder, err := s.CreateFolder("/path/b", "Folder B", nil)

	if err != nil {
		t.Fatalf("Failed to create folder: %v", err)
	}
	folderID := folder.ID

	// Test getting default settings
	settings, err := s.GetFolderSettings(folderID, userID)
	if err != nil {
		t.Fatalf("GetFolderSettings failed for new folder: %v", err)
	}
	if settings.SortBy != "auto" || settings.SortDir != "asc" {
		t.Errorf("Expected default settings, but got %+v", settings)
	}

	// Test updating settings
	err = s.UpdateFolderSettings(folderID, userID, "path", "desc")
	if err != nil {
		t.Fatalf("UpdateFolderSettings failed: %v", err)
	}

	// Test getting updated settings
	newSettings, err := s.GetFolderSettings(folderID, userID)
	if err != nil {
		t.Fatalf("GetFolderSettings failed after update: %v", err)
	}
	if newSettings.SortBy != "path" || newSettings.SortDir != "desc" {
		t.Errorf("Expected updated settings, but got %+v", newSettings)
	}
}
