// Verify all subscription-related database functions.

package store

import (
	"testing"
	"time"

	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestSubscriptionStore(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := New(db)

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
		s.db.Exec("INSERT INTO download_queue (series_title, chapter_identifier, provider_id, chapter_title, created_at) VALUES ('Manga C', 'ch-id-1', 'p3', 'Ch 1', ?)", time.Now())
		s.db.Exec("INSERT INTO download_queue (series_title, chapter_identifier, provider_id, chapter_title, created_at) VALUES ('Manga C', 'ch-id-2', 'p3', 'Ch 2', ?)", time.Now())
		s.db.Exec("INSERT INTO download_queue (series_title, chapter_identifier, provider_id, chapter_title, created_at) VALUES ('Manga D', 'ch-id-3', 'p4', 'Ch 3', ?)", time.Now())

		ids, err := s.GetChapterIdentifiersInQueue("Manga C", "p3")
		if err != nil {
			t.Fatalf("GetChapterIdentifiersInQueue failed: %v", err)
		}
		if len(ids) != 2 {
			t.Errorf("Expected 2 identifiers, got %d", len(ids))
		}
	})
}
