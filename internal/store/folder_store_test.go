package store_test

import (
	"testing"

	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestCreateAndGetFolder(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

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
}

func TestGetAllFoldersByPath(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Create test folders
	_, err := s.CreateFolder("/library/Series A", "Series A", nil)
	if err != nil {
		t.Fatalf("CreateFolder failed: %v", err)
	}

	_, err = s.CreateFolder("/library/Series B", "Series B", nil)
	if err != nil {
		t.Fatalf("CreateFolder failed: %v", err)
	}

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
}

func TestDeleteFolder(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Create test folders
	root, err := s.CreateFolder("/library/Series A", "Series A", nil)
	if err != nil {
		t.Fatalf("CreateFolder failed: %v", err)
	}

	_, err = s.CreateFolder("/library/Series A/Vol 1", "Vol 1", &root.ID)
	if err != nil {
		t.Fatalf("CreateFolder failed: %v", err)
	}

	err = s.DeleteFolder(root.ID) // Deleting the parent should cascade
	if err != nil {
		t.Fatalf("DeleteFolder failed: %v", err)
	}

	remainingFolders, _ := s.GetAllFoldersByPath()
	if len(remainingFolders) != 0 {
		t.Errorf("Expected 0 folders after cascading delete, got %d", len(remainingFolders))
	}
}

func TestUpdateAllFolderThumbnails(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Create a hierarchical folder structure
	series, err := s.CreateFolder("/library/Test Series", "Test Series", nil)
	if err != nil {
		t.Fatalf("Failed to create series folder: %v", err)
	}

	vol1, err := s.CreateFolder("/library/Test Series/Volume 1", "Volume 1", &series.ID)
	if err != nil {
		t.Fatalf("Failed to create volume 1 folder: %v", err)
	}

	vol2, err := s.CreateFolder("/library/Test Series/Volume 2", "Volume 2", &series.ID)
	if err != nil {
		t.Fatalf("Failed to create volume 2 folder: %v", err)
	}

	// Create chapters with different thumbnails
	// Use the CreateChapter method from the store

	// Insert chapters with thumbnails in non-natural order
	_, err = s.CreateChapter(vol1.ID, "/library/Test Series/Volume 1/chapter-10.cbz", "hash10", 20, "thumb10.jpg")
	if err != nil {
		t.Fatalf("Failed to create chapter 10: %v", err)
	}

	_, err = s.CreateChapter(vol1.ID, "/library/Test Series/Volume 1/chapter-1.cbz", "hash1", 18, "thumb1.jpg")
	if err != nil {
		t.Fatalf("Failed to create chapter 1: %v", err)
	}

	_, err = s.CreateChapter(vol1.ID, "/library/Test Series/Volume 1/chapter-2.cbz", "hash2", 19, "thumb2.jpg")
	if err != nil {
		t.Fatalf("Failed to create chapter 2: %v", err)
	}

	_, err = s.CreateChapter(vol2.ID, "/library/Test Series/Volume 2/chapter-1.cbz", "hash21", 22, "thumb21.jpg")
	if err != nil {
		t.Fatalf("Failed to create chapter 2-1: %v", err)
	}

	// Create an empty folder (no chapters)
	emptyFolder, err := s.CreateFolder("/library/Test Series/Empty Volume", "Empty Volume", &series.ID)
	if err != nil {
		t.Fatalf("Failed to create empty folder: %v", err)
	}

	// Run UpdateAllFolderThumbnails
	err = s.UpdateAllFolderThumbnails()
	if err != nil {
		t.Fatalf("UpdateAllFolderThumbnails failed: %v", err)
	}

	// Verify that folders got the correct thumbnails based on natural sorting
	// Volume 1 should have thumb1.jpg (from chapter-1.cbz, which comes first in natural sort)
	updatedVol1, err := s.GetFolder(vol1.ID)
	if err != nil {
		t.Fatalf("Failed to get updated volume 1: %v", err)
	}
	if updatedVol1.Thumbnail != "thumb1.jpg" {
		t.Errorf("Volume 1 should have thumbnail 'thumb1.jpg' (from chapter-1.cbz), got '%s'", updatedVol1.Thumbnail)
	}

	// Volume 2 should have thumb21.jpg (from chapter-1.cbz)
	updatedVol2, err := s.GetFolder(vol2.ID)
	if err != nil {
		t.Fatalf("Failed to get updated volume 2: %v", err)
	}
	if updatedVol2.Thumbnail != "thumb21.jpg" {
		t.Errorf("Volume 2 should have thumbnail 'thumb21.jpg', got '%s'", updatedVol2.Thumbnail)
	}

	// Series folder should have thumb1.jpg (from the first chapter in its subtree)
	updatedSeries, err := s.GetFolder(series.ID)
	if err != nil {
		t.Fatalf("Failed to get updated series: %v", err)
	}
	if updatedSeries.Thumbnail != "thumb1.jpg" {
		t.Errorf("Series should have thumbnail 'thumb1.jpg' (from first chapter in subtree), got '%s'", updatedSeries.Thumbnail)
	}

	// Empty folder should have no thumbnail
	updatedEmpty, err := s.GetFolder(emptyFolder.ID)
	if err != nil {
		t.Fatalf("Failed to get updated empty folder: %v", err)
	}
	if updatedEmpty.Thumbnail != "" {
		t.Errorf("Empty folder should have no thumbnail, got '%s'", updatedEmpty.Thumbnail)
	}
}

func TestUpdateAllFolderThumbnailsWithChaptersWithoutThumbnails(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Create a folder with chapters that have no thumbnails
	folder, err := s.CreateFolder("/library/No Thumbnails", "No Thumbnails", nil)
	if err != nil {
		t.Fatalf("Failed to create folder: %v", err)
	}

	// Insert chapter without thumbnail using CreateChapter
	_, err = s.CreateChapter(folder.ID, "/library/No Thumbnails/chapter.cbz", "hash", 15, "")
	if err != nil {
		t.Fatalf("Failed to create chapter without thumbnail: %v", err)
	}

	// Run UpdateAllFolderThumbnails
	err = s.UpdateAllFolderThumbnails()
	if err != nil {
		t.Fatalf("UpdateAllFolderThumbnails failed: %v", err)
	}

	// Folder should still have no thumbnail since its chapter has no thumbnail
	updatedFolder, err := s.GetFolder(folder.ID)
	if err != nil {
		t.Fatalf("Failed to get updated folder: %v", err)
	}
	if updatedFolder.Thumbnail != "" {
		t.Errorf("Folder should have no thumbnail when its chapters have no thumbnails, got '%s'", updatedFolder.Thumbnail)
	}
}

func TestGetFolderByPath(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Create a folder with a specific path
	folder, err := s.CreateFolder("/library/Test Path", "Test Path", nil)
	if err != nil {
		t.Fatalf("Failed to create folder: %v", err)
	}

	// Get folder by path
	retrieved, err := s.GetFolderByPath("/library/Test Path")
	if err != nil {
		t.Fatalf("GetFolderByPath failed: %v", err)
	}
	if retrieved.ID != folder.ID {
		t.Errorf("Expected folder ID %d, got %d", folder.ID, retrieved.ID)
	}
	if retrieved.Name != "Test Path" {
		t.Errorf("Expected folder name 'Test Path', got '%s'", retrieved.Name)
	}

	// Test non-existent path
	_, err = s.GetFolderByPath("/library/Non Existent")
	if err == nil {
		t.Error("Expected error for non-existent path")
	}
}

func TestListItems(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Create a hierarchical structure for testing
	series, err := s.CreateFolder("/library/List Test Series", "List Test Series", nil)
	if err != nil {
		t.Fatalf("Failed to create series folder: %v", err)
	}

	vol1, err := s.CreateFolder("/library/List Test Series/Volume 1", "Volume 1", &series.ID)
	if err != nil {
		t.Fatalf("Failed to create volume 1 folder: %v", err)
	}

	vol2, err := s.CreateFolder("/library/List Test Series/Volume 2", "Volume 2", &series.ID)
	if err != nil {
		t.Fatalf("Failed to create volume 2 folder: %v", err)
	}

	// Create chapters
	_, err = s.CreateChapter(vol1.ID, "/library/List Test Series/Volume 1/chapter-1.cbz", "hash1", 20, "thumb1.jpg")
	if err != nil {
		t.Fatalf("Failed to create chapter 1: %v", err)
	}

	_, err = s.CreateChapter(vol1.ID, "/library/List Test Series/Volume 1/chapter-2.cbz", "hash2", 22, "thumb2.jpg")
	if err != nil {
		t.Fatalf("Failed to create chapter 2: %v", err)
	}

	_, err = s.CreateChapter(vol2.ID, "/library/List Test Series/Volume 2/chapter-1.cbz", "hash3", 25, "thumb3.jpg")
	if err != nil {
		t.Fatalf("Failed to create chapter 3: %v", err)
	}

	// Test listing root level (should show series folders)
	opts := store.ListItemsOptions{
		UserID:   1,
		ParentID: &[]int64{0}[0], // Root level
		Page:     1,
		PerPage:  10,
	}
	currentFolder, subfolders, chapters, total, err := s.ListItems(opts)
	if err != nil {
		t.Fatalf("ListItems failed: %v", err)
	}
	if currentFolder != nil {
		t.Error("Current folder should be nil for root level")
	}
	if len(subfolders) != 1 {
		t.Error("Expected subfolders at root level")
	}
	if len(chapters) != 0 {
		t.Error("Expected no chapters at root level")
	}
	if total <= 0 {
		t.Error("Expected positive total count")
	}

	// Test listing series level (should show volumes and chapters)
	opts.ParentID = &series.ID
	currentFolder, subfolders, chapters, total, err = s.ListItems(opts)
	if err != nil {
		t.Fatalf("ListItems failed: %v", err)
	}
	if currentFolder == nil {
		t.Error("Current folder should not be nil")
	}
	if currentFolder != nil && currentFolder.ID != series.ID {
		t.Errorf("Expected current folder ID %d, got %d", series.ID, currentFolder.ID)
	}
	if len(subfolders) != 2 {
		t.Errorf("Expected 2 subfolders, got %d", len(subfolders))
	}
	if len(chapters) != 0 {
		t.Error("Expected no chapters at series level")
	}
	if total != 2 {
		t.Errorf("Expected 2 total items, got %d", total)
	}

	// Test listing volume level (should show chapters)
	opts.ParentID = &vol1.ID
	_, subfolders, chapters, total, err = s.ListItems(opts)
	if err != nil {
		t.Fatalf("ListItems failed: %v", err)
	}
	if len(subfolders) != 0 {
		t.Error("Expected no subfolders at volume level")
	}
	if len(chapters) != 2 {
		t.Errorf("Expected 2 chapters, got %d", len(chapters))
	}
	if chapters[0].Path != "/library/List Test Series/Volume 1/chapter-1.cbz" {
		t.Errorf("Expected chapter path '/library/List Test Series/Volume 1/chapter-1.cbz', got '%s'", chapters[0].Path)
	}
	if chapters[1].Path != "/library/List Test Series/Volume 1/chapter-2.cbz" {
		t.Errorf("Expected chapter path '/library/List Test Series/Volume 1/chapter-2.cbz', got '%s'", chapters[1].Path)
	}
	if total != 2 {
		t.Errorf("Expected 2 total items, got %d", total)
	}

	// Test search functionality
	opts.ParentID = &series.ID
	opts.Search = "Volume 1"
	_, subfolders, chapters, total, err = s.ListItems(opts)
	if err != nil {
		t.Fatalf("ListItems with search failed: %v", err)
	}
	if len(subfolders) != 1 {
		t.Errorf("Expected 1 subfolder matching search, got %d", len(subfolders))
	}
	if subfolders[0].Name != "Volume 1" {
		t.Errorf("Expected subfolder name 'Volume 1', got '%s'", subfolders[0].Name)
	}
	if chapters != nil {
		t.Errorf("Expected chapters to be nil, got %v", chapters)
	}
	if total != 1 {
		t.Errorf("Expected 1 total item, got %d", total)
	}
}

func TestGetFolderPath(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Create a deep hierarchical structure
	root, err := s.CreateFolder("/library/Deep Series", "Deep Series", nil)
	if err != nil {
		t.Fatalf("Failed to create root folder: %v", err)
	}

	level1, err := s.CreateFolder("/library/Deep Series/Level 1", "Level 1", &root.ID)
	if err != nil {
		t.Fatalf("Failed to create level 1 folder: %v", err)
	}

	level2, err := s.CreateFolder("/library/Deep Series/Level 1/Level 2", "Level 2", &level1.ID)
	if err != nil {
		t.Fatalf("Failed to create level 2 folder: %v", err)
	}

	level3, err := s.CreateFolder("/library/Deep Series/Level 1/Level 2/Level 3", "Level 3", &level2.ID)
	if err != nil {
		t.Fatalf("Failed to create level 3 folder: %v", err)
	}

	// Get path for deepest level
	path, err := s.GetFolderPath(level3.ID)
	if err != nil {
		t.Fatalf("GetFolderPath failed: %v", err)
	}

	if len(path) != 4 {
		t.Errorf("Expected 4 folders in path, got %d", len(path))
	}

	// Check path order (should be from root to leaf)
	if path[0].ID != root.ID {
		t.Errorf("First folder should be root, got ID %d", path[0].ID)
	}
	if path[1].ID != level1.ID {
		t.Errorf("Second folder should be level 1, got ID %d", path[1].ID)
	}
	if path[2].ID != level2.ID {
		t.Errorf("Third folder should be level 2, got ID %d", path[2].ID)
	}
	if path[3].ID != level3.ID {
		t.Errorf("Fourth folder should be level 3, got ID %d", path[3].ID)
	}

	// Test with root folder
	rootPath, err := s.GetFolderPath(root.ID)
	if err != nil {
		t.Fatalf("GetFolderPath for root failed: %v", err)
	}
	if len(rootPath) != 1 {
		t.Errorf("Expected 1 folder in root path, got %d", len(rootPath))
	}
	if rootPath[0].ID != root.ID {
		t.Errorf("Root path should contain root folder, got ID %d", rootPath[0].ID)
	}
}

func TestGetAndUpdateFolderSettings(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Create a folder for testing
	folder, err := s.CreateFolder("/library/Settings Test", "Settings Test", nil)
	if err != nil {
		t.Fatalf("Failed to create folder: %v", err)
	}

	userID := int64(1)

	// Test getting default settings (should return defaults when no settings exist)
	settings, err := s.GetFolderSettings(folder.ID, userID)
	if err != nil {
		t.Fatalf("GetFolderSettings failed: %v", err)
	}
	if settings.SortBy != "auto" {
		t.Errorf("Expected default sort_by 'auto', got '%s'", settings.SortBy)
	}
	if settings.SortDir != "asc" {
		t.Errorf("Expected default sort_dir 'asc', got '%s'", settings.SortDir)
	}

	// Test updating settings
	err = s.UpdateFolderSettings(folder.ID, userID, "name", "desc")
	if err != nil {
		t.Fatalf("UpdateFolderSettings failed: %v", err)
	}

	// Test getting updated settings
	updatedSettings, err := s.GetFolderSettings(folder.ID, userID)
	if err != nil {
		t.Fatalf("GetFolderSettings after update failed: %v", err)
	}
	if updatedSettings.SortBy != "name" {
		t.Errorf("Expected sort_by 'name', got '%s'", updatedSettings.SortBy)
	}
	if updatedSettings.SortDir != "desc" {
		t.Errorf("Expected sort_dir 'desc', got '%s'", updatedSettings.SortDir)
	}

	// Test updating settings again (should update existing record)
	err = s.UpdateFolderSettings(folder.ID, userID, "created_at", "asc")
	if err != nil {
		t.Fatalf("UpdateFolderSettings second time failed: %v", err)
	}

	finalSettings, err := s.GetFolderSettings(folder.ID, userID)
	if err != nil {
		t.Fatalf("GetFolderSettings after second update failed: %v", err)
	}
	if finalSettings.SortBy != "created_at" {
		t.Errorf("Expected sort_by 'created_at', got '%s'", finalSettings.SortBy)
	}
	if finalSettings.SortDir != "asc" {
		t.Errorf("Expected sort_dir 'asc', got '%s'", finalSettings.SortDir)
	}

	// Test that different users have separate settings
	user2ID := int64(2)
	settings2, err := s.GetFolderSettings(folder.ID, user2ID)
	if err != nil {
		t.Fatalf("GetFolderSettings for user 2 failed: %v", err)
	}
	if settings2.SortBy != "auto" {
		t.Errorf("User 2 should have default settings, got sort_by '%s'", settings2.SortBy)
	}
}

func TestGetFolderWithTags(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Create a folder
	folder, err := s.CreateFolder("/library/Tagged Folder", "Tagged Folder", nil)
	if err != nil {
		t.Fatalf("Failed to create folder: %v", err)
	}

	// Add tags to the folder
	_, err = s.AddTagToFolder(folder.ID, "action")
	if err != nil {
		t.Fatalf("Failed to add tag 'action': %v", err)
	}

	_, err = s.AddTagToFolder(folder.ID, "adventure")
	if err != nil {
		t.Fatalf("Failed to add tag 'adventure': %v", err)
	}

	// Get folder and verify tags are loaded
	retrieved, err := s.GetFolder(folder.ID)
	if err != nil {
		t.Fatalf("GetFolder failed: %v", err)
	}
	if len(retrieved.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(retrieved.Tags))
	}

	// Check tag names
	tagNames := make(map[string]bool)
	for _, tag := range retrieved.Tags {
		tagNames[tag.Name] = true
	}
	if !tagNames["action"] {
		t.Error("Missing 'action' tag")
	}
	if !tagNames["adventure"] {
		t.Error("Missing 'adventure' tag")
	}
}

func TestGetFolderWithInvalidID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Test with ID 0
	_, err := s.GetFolder(0)
	if err == nil {
		t.Error("Expected error for folder ID 0")
	}

	// Test with non-existent ID
	_, err = s.GetFolder(99999)
	if err == nil {
		t.Error("Expected error for non-existent folder ID")
	}
}
