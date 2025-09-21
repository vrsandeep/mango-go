// Verify all subscription-related database functions.

package store_test

import (
	"testing"
	"time"

	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestSubscriptionStore(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	// Add subscriptions for testing
	sub1, _ := s.SubscribeToSeries("Manga A", "id-a", "p1")
	s.SubscribeToSeries("Manga B", "id-b", "p2")

	t.Run("Get All Subscriptions", func(t *testing.T) {
		subs, err := s.GetAllSubscriptions("")
		if err != nil {
			t.Fatalf("GetAllSubscriptions failed: %v", err)
		}
		if len(subs) != 2 {
			t.Errorf("Expected 2 subs, got %d", len(subs))
		}
	})

	t.Run("Get Filtered Subscriptions", func(t *testing.T) {
		subs, err := s.GetAllSubscriptions("p1")
		if err != nil {
			t.Fatalf("GetAllSubscriptions with filter failed: %v", err)
		}
		if len(subs) != 1 {
			t.Errorf("Expected 1 filtered sub, got %d", len(subs))
		}
		if subs[0].ProviderID != "p1" {
			t.Errorf("Wrong provider returned from filter")
		}
	})

	t.Run("Delete Subscription", func(t *testing.T) {
		err := s.DeleteSubscription(sub1.ID)
		if err != nil {
			t.Fatalf("DeleteSubscription failed: %v", err)
		}

		remaining, _ := s.GetAllSubscriptions("")
		if len(remaining) != 1 {
			t.Errorf("Expected 1 sub after delete, got %d", len(remaining))
		}
		if remaining[0].SeriesTitle == "Manga A" {
			t.Error("Deleted subscription still exists")
		}
	})

	t.Run("Update Last Checked", func(t *testing.T) {
		subs, _ := s.GetAllSubscriptions("p2")
		subToUpdate := subs[0]

		// Ensure it's initially nil
		if subToUpdate.LastCheckedAt != nil {
			t.Fatal("Expected LastCheckedAt to be nil initially")
		}

		err := s.UpdateSubscriptionLastChecked(subToUpdate.ID)
		if err != nil {
			t.Fatalf("UpdateSubscriptionLastChecked failed: %v", err)
		}

		updatedSub, _ := s.GetSubscriptionByID(subToUpdate.ID)
		if updatedSub.LastCheckedAt == nil {
			t.Error("LastCheckedAt was not updated")
		}
		// Check if the timestamp is recent
		if time.Since(*updatedSub.LastCheckedAt) > 5*time.Second {
			t.Error("LastCheckedAt timestamp is not recent")
		}
	})

	t.Run("Get Chapter Identifiers In Queue", func(t *testing.T) {
		db.Exec("INSERT INTO download_queue (series_title, chapter_identifier, provider_id, chapter_title, created_at) VALUES ('Manga C', 'ch-id-1', 'p3', 'Ch 1', ?)", time.Now())
		db.Exec("INSERT INTO download_queue (series_title, chapter_identifier, provider_id, chapter_title, created_at) VALUES ('Manga C', 'ch-id-2', 'p3', 'Ch 2', ?)", time.Now())
		db.Exec("INSERT INTO download_queue (series_title, chapter_identifier, provider_id, chapter_title, created_at) VALUES ('Manga D', 'ch-id-3', 'p4', 'Ch 3', ?)", time.Now())

		ids, err := s.GetChapterIdentifiersInQueue("Manga C", "p3")
		if err != nil {
			t.Fatalf("GetChapterIdentifiersInQueue failed: %v", err)
		}
		if len(ids) != 2 {
			t.Errorf("Expected 2 identifiers, got %d", len(ids))
		}
	})

	t.Run("Subscribe with folder path", func(t *testing.T) {
		folderPath := "custom/manga/path"
		sub, err := s.SubscribeToSeriesWithFolder("Manga with Folder", "id-folder", "p1", &folderPath)
		if err != nil {
			t.Fatalf("SubscribeToSeriesWithFolder failed: %v", err)
		}
		if sub.FolderPath == nil || *sub.FolderPath != folderPath {
			t.Errorf("Expected folder path %s, got %v", folderPath, sub.FolderPath)
		}
	})

	t.Run("Subscribe with null folder path", func(t *testing.T) {
		sub, err := s.SubscribeToSeriesWithFolder("Manga without Folder", "id-no-folder", "p1", nil)
		if err != nil {
			t.Fatalf("SubscribeToSeriesWithFolder with nil folder path failed: %v", err)
		}
		if sub.FolderPath != nil {
			t.Errorf("Expected folder path to be nil, got %v", sub.FolderPath)
		}
	})

	t.Run("Get subscription with folder path", func(t *testing.T) {
		folderPath := "another/custom/path"
		sub, _ := s.SubscribeToSeriesWithFolder("Manga for Get Test", "id-get-test", "p1", &folderPath)

		retrievedSub, err := s.GetSubscriptionByID(sub.ID)
		if err != nil {
			t.Fatalf("GetSubscriptionByID failed: %v", err)
		}
		if retrievedSub.FolderPath == nil || *retrievedSub.FolderPath != folderPath {
			t.Errorf("Expected folder path %s, got %v", folderPath, retrievedSub.FolderPath)
		}
	})

	t.Run("Update folder path", func(t *testing.T) {
		// Create subscription without folder path
		sub, _ := s.SubscribeToSeriesWithFolder("Manga for Update", "id-update", "p1", nil)

		// Update with folder path
		newFolderPath := "updated/custom/path"
		err := s.UpdateSubscriptionFolderPath(sub.ID, &newFolderPath)
		if err != nil {
			t.Fatalf("UpdateSubscriptionFolderPath failed: %v", err)
		}

		// Verify update
		updatedSub, _ := s.GetSubscriptionByID(sub.ID)
		if updatedSub.FolderPath == nil || *updatedSub.FolderPath != newFolderPath {
			t.Errorf("Expected folder path %s, got %v", newFolderPath, updatedSub.FolderPath)
		}

		// Update to null
		err = s.UpdateSubscriptionFolderPath(sub.ID, nil)
		if err != nil {
			t.Fatalf("UpdateSubscriptionFolderPath to nil failed: %v", err)
		}

		// Verify null update
		updatedSub, _ = s.GetSubscriptionByID(sub.ID)
		if updatedSub.FolderPath != nil {
			t.Errorf("Expected folder path to be nil after update, got %v", updatedSub.FolderPath)
		}
	})

	t.Run("Get all subscriptions includes folder path", func(t *testing.T) {
		folderPath := "test/folder/path"
		s.SubscribeToSeriesWithFolder("Manga for GetAll", "id-getall", "p1", &folderPath)

		subs, err := s.GetAllSubscriptions("p1")
		if err != nil {
			t.Fatalf("GetAllSubscriptions failed: %v", err)
		}

		// Find our test subscription
		var foundSub *models.Subscription
		for _, sub := range subs {
			if sub.SeriesIdentifier == "id-getall" {
				foundSub = sub
				break
			}
		}

		if foundSub == nil {
			t.Fatal("Could not find test subscription")
		}
		if foundSub.FolderPath == nil || *foundSub.FolderPath != folderPath {
			t.Errorf("Expected folder path %s, got %v", folderPath, foundSub.FolderPath)
		}
	})
}
