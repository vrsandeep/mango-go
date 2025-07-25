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

	t.Run("Get Chapter By ID", func(t *testing.T) {
		chap, err := s.GetChapterByID(1, 1)
		if err != nil {
			t.Fatalf("GetChapterByID failed: %v", err)
		}
		if chap.ID != 1 {
			t.Errorf("Expected chapter ID 1, got %d", chap.ID)
		}
		if chap.FolderID != folder.ID {
			t.Errorf("Expected folder ID %d, got %d", folder.ID, chap.FolderID)
		}
		if chap.Path != "/library/Series A/ch1.cbz" {
			t.Errorf("Expected path '/library/Series A/ch1.cbz', got '%s'", chap.Path)
		}
		if chap.ContentHash != "hash1" {
			t.Errorf("Expected content hash 'hash1', got '%s'", chap.ContentHash)
		}
		if chap.PageCount != 20 {
			t.Errorf("Expected page count 20, got %d", chap.PageCount)
		}
		if chap.Thumbnail != "thumb1" {
			t.Errorf("Expected thumbnail 'thumb1', got '%s'", chap.Thumbnail)
		}
		if chap.Read != false {
			t.Errorf("Expected chapter 'read' status to be false, got true")
		}
		if chap.ProgressPercent != 0 {
			t.Errorf("Expected chapter 'progress_percent' to be 0, got %d", chap.ProgressPercent)
		}
		if chap.CreatedAt.IsZero() {
			t.Errorf("Expected created_at to be set")
		}
		if chap.UpdatedAt.IsZero() {
			t.Errorf("Expected updated_at to be set")
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

	t.Run("Update Chapter Thumbnail", func(t *testing.T) {
		chapMap, _ := s.GetAllChaptersByHash()
		chapID := chapMap["hash1"].ID
		newThumbnail := "data:image/jpeg;base64,newthumb"
		err := s.UpdateChapterThumbnail(chapID, newThumbnail)
		if err != nil {
			t.Fatalf("UpdateChapterThumbnail failed: %v", err)
		}
		updatedChap, _ := s.GetChapterByID(chapID, 1)
		if updatedChap.Thumbnail != newThumbnail {
			t.Errorf("Chapter thumbnail was not updated")
		}
	})

	t.Run("Update Chapter Progress", func(t *testing.T) {
		chapMap, _ := s.GetAllChaptersByHash()
		chapID := chapMap["hash1"].ID
		newProgress := 100
		newReadStatus := true

		// Create a user
		user, _ := s.CreateUser("testuser", "testuser@example.com", "user")

		err := s.UpdateChapterProgress(chapID, user.ID, newProgress, newReadStatus)
		if err != nil {
			t.Fatalf("UpdateChapterProgress failed: %v", err)
		}
		updatedChap, err := s.GetChapterByID(chapID, 1)
		if err != nil {
			t.Fatalf("Failed to get chapter after update: %v", err)
		}
		if updatedChap.ProgressPercent != newProgress {
			t.Errorf("Chapter progress was not updated")
		}
		if updatedChap.Read != newReadStatus {
			t.Errorf("Chapter read status was not updated")
		}
	})

	t.Run("Get Folder Stats for a read chapter", func(t *testing.T) {
		chapMap, _ := s.GetAllChaptersByHash()
		chapter, _ := s.GetChapterByID(chapMap["hash1"].ID, 1)
		totalChapters, readChapters, err := s.GetFolderStats(chapter.FolderID, 1)
		if err != nil {
			t.Fatalf("GetFolderStats failed: %v", err)
		}
		if totalChapters != 1 {
			t.Errorf("Expected 1 total chapters, got %d", totalChapters)
		}
		if readChapters != 1 {
			t.Errorf("Expected 1 read chapters, got %d", readChapters)
		}
	})

	t.Run("Get Folder Stats for a untouched/unread chapter", func(t *testing.T) {
		folder, _ := s.CreateFolder("/library/Untouched Series", "Untouched Series", nil)
		chapter, _ := s.CreateChapter(folder.ID, "/library/Untouched Series/ch1.cbz", "hash2", 20, "thumb2")

		totalChapters, readChapters, err := s.GetFolderStats(chapter.FolderID, 1)
		if err != nil {
			t.Fatalf("GetFolderStats failed: %v", err)
		}
		if totalChapters != 1 {
			t.Errorf("Expected 1 total chapters, got %d", totalChapters)
		}
		if readChapters != 0 {
			t.Errorf("Expected 0 read chapters, got %d", readChapters)
		}
	})

	t.Run("Delete Chapter By Hash", func(t *testing.T) {
		err := s.DeleteChapterByHash("hash1")
		if err != nil {
			t.Fatalf("DeleteChapterByHash failed: %v", err)
		}
		err = s.DeleteChapterByHash("hash2")
		if err != nil {
			t.Fatalf("DeleteChapterByHash failed: %v", err)
		}
		chapMap, _ := s.GetAllChaptersByHash()
		if len(chapMap) != 0 {
			t.Errorf("Expected 0 chapters after delete, got %d", len(chapMap))
		}
	})

	t.Run("Get Chapter Title", func(t *testing.T) {
		// Create a test chapter
		folder, _ := s.CreateFolder("/library/Test Series", "Test Series", nil)
		chapter, _ := s.CreateChapter(folder.ID, "/library/Test Series/Chapter 1.cbz", "hash3", 20, "thumb3")

		title := store.GetChapterTitle(chapter)
		expectedTitle := "Chapter 1"
		if title != expectedTitle {
			t.Errorf("Expected title '%s', got '%s'", expectedTitle, title)
		}

		// Test with different file extensions
		chapter.Path = "/library/Test Series/Chapter 2.pdf"
		title = store.GetChapterTitle(chapter)
		expectedTitle = "Chapter 2"
		if title != expectedTitle {
			t.Errorf("Expected title '%s', got '%s'", expectedTitle, title)
		}

		// Test with no extension
		chapter.Path = "/library/Test Series/Chapter 3"
		title = store.GetChapterTitle(chapter)
		expectedTitle = "Chapter 3"
		if title != expectedTitle {
			t.Errorf("Expected title '%s', got '%s'", expectedTitle, title)
		}

		// Test with empty path
		chapter.Path = ""
		title = store.GetChapterTitle(chapter)
		expectedTitle = ""
		if title != expectedTitle {
			t.Errorf("Expected title '%s', got '%s'", expectedTitle, title)
		}
	})

	t.Run("Get Chapter Neighbors", func(t *testing.T) {
		// Create a folder with multiple chapters
		folder, _ := s.CreateFolder("/library/Neighbor Series", "Neighbor Series", nil)

		// Create multiple chapters
		ch1, _ := s.CreateChapter(folder.ID, "/library/Neighbor Series/ch1.cbz", "hash4", 20, "thumb4")
		ch2, _ := s.CreateChapter(folder.ID, "/library/Neighbor Series/ch2.cbz", "hash5", 20, "thumb5")
		ch3, _ := s.CreateChapter(folder.ID, "/library/Neighbor Series/ch3.cbz", "hash6", 20, "thumb6")

		// Create a user for the test
		user, _ := s.CreateUser("neighboruser", "neighbor@example.com", "user")

		// Test middle chapter (should have both prev and next)
		neighbors, err := s.GetChapterNeighbors(folder.ID, ch2.ID, user.ID)
		if err != nil {
			t.Fatalf("GetChapterNeighbors failed: %v", err)
		}
		if neighbors["prev"] == nil {
			t.Error("Expected previous chapter for middle chapter")
		}
		if *neighbors["prev"] != ch1.ID {
			t.Errorf("Expected previous chapter ID %d, got %d", ch1.ID, *neighbors["prev"])
		}
		if neighbors["next"] == nil {
			t.Error("Expected next chapter for middle chapter")
		}
		if *neighbors["next"] != ch3.ID {
			t.Errorf("Expected next chapter ID %d, got %d", ch3.ID, *neighbors["next"])
		}

		// Test first chapter (should have next but no prev)
		neighbors, err = s.GetChapterNeighbors(folder.ID, ch1.ID, user.ID)
		if err != nil {
			t.Fatalf("GetChapterNeighbors failed: %v", err)
		}
		if neighbors["prev"] != nil {
			t.Error("Expected no previous chapter for first chapter")
		}
		if neighbors["next"] == nil {
			t.Error("Expected next chapter for first chapter")
		}
		if *neighbors["next"] != ch2.ID {
			t.Errorf("Expected next chapter ID %d, got %d", ch2.ID, *neighbors["next"])
		}

		// Test last chapter (should have prev but no next)
		neighbors, err = s.GetChapterNeighbors(folder.ID, ch3.ID, user.ID)
		if err != nil {
			t.Fatalf("GetChapterNeighbors failed: %v", err)
		}
		if neighbors["prev"] == nil {
			t.Error("Expected previous chapter for last chapter")
		}
		if *neighbors["prev"] != ch2.ID {
			t.Errorf("Expected previous chapter ID %d, got %d", ch2.ID, *neighbors["prev"])
		}
		if neighbors["next"] != nil {
			t.Error("Expected no next chapter for last chapter")
		}

		// Test with non-existent chapter ID
		neighbors, err = s.GetChapterNeighbors(folder.ID, 99999, user.ID)
		if err != nil {
			t.Fatalf("GetChapterNeighbors failed: %v", err)
		}
		if neighbors["prev"] != nil {
			t.Error("Expected no previous chapter for non-existent chapter")
		}
		if neighbors["next"] != nil {
			t.Error("Expected no next chapter for non-existent chapter")
		}
	})

}
