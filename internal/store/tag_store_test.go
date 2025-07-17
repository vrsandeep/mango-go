package store_test

import (
	"testing"

	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)


func TestAddTagToFolder(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Create a folder
	folder, err := s.CreateFolder("/test", "Test Folder", nil)
	if err != nil {
		t.Fatalf("Failed to create folder: %v", err)
	}

	// Add a tag to the folder
	tag, err := s.AddTagToFolder(folder.ID, "action")
	if err != nil {
		t.Fatalf("Failed to add tag to folder: %v", err)
	}
	if tag.Name != "action" {
		t.Errorf("Expected tag name 'action', got '%s'", tag.Name)
	}

	// Add the same tag again (should not error, should not duplicate)
	tag2, err := s.AddTagToFolder(folder.ID, "action")
	if err != nil {
		t.Fatalf("Failed to add duplicate tag to folder: %v", err)
	}
	if tag2.ID != tag.ID {
		t.Errorf("Expected same tag ID, got %d and %d", tag.ID, tag2.ID)
	}
}

func TestRemoveTagFromFolder(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Create a folder and tag, associate them
	folder, err := s.CreateFolder("/test2", "Test Folder 2", nil)
	if err != nil {
		t.Fatalf("Failed to create folder: %v", err)
	}
	tag, err := s.AddTagToFolder(folder.ID, "comedy")
	if err != nil {
		t.Fatalf("Failed to add tag to folder: %v", err)
	}

	// Remove the tag from the folder
	err = s.RemoveTagFromFolder(folder.ID, tag.ID)
	if err != nil {
		t.Fatalf("Failed to remove tag from folder: %v", err)
	}

	// Removing again should not error
	err = s.RemoveTagFromFolder(folder.ID, tag.ID)
	if err != nil {
		t.Fatalf("Failed to remove tag from folder a second time: %v", err)
	}
}
