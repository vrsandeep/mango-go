package store_test

import (
	"testing"

	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestChapterStore(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Setup: Create a folder first
	folder, _ := s.CreateFolder("/library/Series A", "Series A", nil)

	t.Run("Create and Get Chapter", func(t *testing.T) {
		ch, err := s.CreateChapter(folder.ID, "/library/Series A/ch1.cbz", "hash1", 20, "thumb1")
		if err != nil {
			t.Fatalf("CreateChapter failed: %v", err)
		}
		if ch.FolderID != folder.ID {
			t.Error("Chapter created with incorrect folder ID")
		}
	})

	t.Run("Get All Chapters By Hash", func(t *testing.T) {
		chapMap, err := s.GetAllChaptersByHash()
		if err != nil {
			t.Fatalf("GetAllChaptersByHash failed: %v", err)
		}
		if len(chapMap) != 1 {
			t.Errorf("Expected 1 chapter in map, got %d", len(chapMap))
		}
		if info, ok := chapMap["hash1"]; !ok || info.Path != "/library/Series A/ch1.cbz" {
			t.Error("Chapter map data is incorrect")
		}
	})

	t.Run("Update Chapter Path", func(t *testing.T) {
		newFolder, _ := s.CreateFolder("/library/Series B", "Series B", nil)
		chapMap, _ := s.GetAllChaptersByHash()
		chapID := chapMap["hash1"].ID

		newPath := "/library/Series B/ch1-moved.cbz"
		err := s.UpdateChapterPath(chapID, newPath, newFolder.ID)
		if err != nil {
			t.Fatalf("UpdateChapterPath failed: %v", err)
		}

		updatedChapMap, _ := s.GetAllChaptersByHash()
		if updatedChapMap["hash1"].Path != newPath {
			t.Error("Chapter path was not updated")
		}
	})

	t.Run("Delete Chapter By Hash", func(t *testing.T) {
		err := s.DeleteChapterByHash("hash1")
		if err != nil {
			t.Fatalf("DeleteChapterByHash failed: %v", err)
		}
		chapMap, _ := s.GetAllChaptersByHash()
		if len(chapMap) != 0 {
			t.Errorf("Expected 0 chapters after delete, got %d", len(chapMap))
		}
	})
}
