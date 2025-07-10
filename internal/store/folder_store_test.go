package store_test

import (
	"testing"

	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestFolderStore(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	t.Run("Create and Get Folder", func(t *testing.T) {
		// Create root folder
		root, err := s.CreateFolder("/library/Series A", "Series A", nil)
		if err != nil {
			t.Fatalf("CreateFolder (root) failed: %v", err)
		}
		if root.ID == 0 {
			t.Fatal("Expected non-zero ID for root folder")
		}

		// Create child folder
		child, err := s.CreateFolder("/library/Series A/Vol 1", "Vol 1", &root.ID)
		if err != nil {
			t.Fatalf("CreateFolder (child) failed: %v", err)
		}
		if child.ParentID == nil || *child.ParentID != root.ID {
			t.Error("Child folder does not have correct parent ID")
		}

		// Get the child back
		retrieved, err := s.GetFolder(child.ID)
		if err != nil {
			t.Fatalf("GetFolder failed: %v", err)
		}
		if retrieved.Name != "Vol 1" {
			t.Errorf("Expected folder name 'Vol 1', got '%s'", retrieved.Name)
		}
	})

	t.Run("Get All Folders By Path", func(t *testing.T) {
		folderMap, err := s.GetAllFoldersByPath()
		if err != nil {
			t.Fatalf("GetAllFoldersByPath failed: %v", err)
		}
		if len(folderMap) != 2 {
			t.Errorf("Expected 2 folders in map, got %d", len(folderMap))
		}
		if _, ok := folderMap["/library/Series A"]; !ok {
			t.Error("Folder map is missing '/library/Series A'")
		}
	})

	t.Run("Delete Folder", func(t *testing.T) {
		folders, _ := s.GetAllFoldersByPath()
		var rootID int64
		for _, f := range folders {
			if f.ParentID == nil {
				rootID = f.ID
			}
		}

		err := s.DeleteFolder(rootID) // Deleting the parent should cascade
		if err != nil {
			t.Fatalf("DeleteFolder failed: %v", err)
		}

		remainingFolders, _ := s.GetAllFoldersByPath()
		if len(remainingFolders) != 0 {
			t.Errorf("Expected 0 folders after cascading delete, got %d", len(remainingFolders))
		}
	})
}
