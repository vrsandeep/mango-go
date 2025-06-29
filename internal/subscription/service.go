package subscription

import (
	"log"
	"time"

	"github.com/vrsandeep/mango-go/internal/core"
	"github.com/vrsandeep/mango-go/internal/downloader/providers"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/store"
)

// Service holds the dependencies for the subscription checker.
type Service struct {
	app *core.App
	st  *store.Store
}

// NewService creates a new subscription service.
func NewService(app *core.App) *Service {
	return &Service{
		app: app,
		st:  store.New(app.DB),
	}
}

// Start runs the subscription service in the background, checking periodically.
func (s *Service) Start() {
	log.Println("Starting subscription service...")
	// Run an initial check on startup after a short delay
	time.AfterFunc(1*time.Minute, s.CheckAllSubscriptions)

	// Then, check on a regular interval defined in the config
	// For now, hardcoding to 6 hours.
	ticker := time.NewTicker(6 * time.Hour)
	go func() {
		for range ticker.C {
			s.CheckAllSubscriptions()
		}
	}()
}

// CheckAllSubscriptions fetches all subscriptions and checks them for new chapters.
func (s *Service) CheckAllSubscriptions() {
	log.Println("Running scheduled subscription check...")
	subscriptions, err := s.st.GetAllSubscriptions("")
	if err != nil {
		log.Printf("Subscription Check Error: Failed to get subscriptions: %v", err)
		return
	}

	for _, sub := range subscriptions {
		s.CheckSingleSubscription(sub.ID)
	}
	log.Println("Finished scheduled subscription check.")
}

// CheckSingleSubscription checks a specific subscription for new chapters.
func (s *Service) CheckSingleSubscription(subID int64) {
	sub, err := s.st.GetSubscriptionByID(subID)
	if err != nil {
		log.Printf("Subscription Check Error: Failed to get subscription %d: %v", subID, err)
		return
	}

	provider, ok := providers.Get(sub.ProviderID)
	if !ok {
		log.Printf("Subscription Check Error: Provider '%s' not found for subscription %d", sub.ProviderID, subID)
		return
	}

	// Fetch latest chapters from the provider
	remoteChapters, err := provider.GetChapters(sub.SeriesIdentifier)
	if err != nil {
		log.Printf("Subscription Check Error: Failed to get remote chapters for '%s': %v", sub.SeriesTitle, err)
		return
	}

	// Fetch existing chapter identifiers from the download queue to avoid duplicates
	existingChapterIDs, err := s.st.GetChapterIdentifiersInQueue(sub.SeriesTitle, sub.ProviderID)
	if err != nil {
		log.Printf("Subscription Check Error: Failed to get queued chapters for '%s': %v", sub.SeriesTitle, err)
		return
	}
	existingSet := make(map[string]bool)
	for _, id := range existingChapterIDs {
		existingSet[id] = true
	}

	// Filter out chapters that are already in the queue
	// and that were published after the subscription was last checked or created.
	var newChapters []models.ChapterResult
	for _, remoteChapter := range remoteChapters {
		// Check if chapter is already queued
		if _, exists := existingSet[remoteChapter.Identifier]; exists {
			continue
		}
		// Check if chapter was published after the subscription was last checked or created.
		lastCheckedAt := sub.LastCheckedAt
		if lastCheckedAt == nil {
			lastCheckedAt = &sub.CreatedAt
		}
		if remoteChapter.PublishedAt.After(*lastCheckedAt) {
			newChapters = append(newChapters, remoteChapter)
		}
	}

	// If new chapters are found, add them to the queue
	if len(newChapters) > 0 {
		log.Printf("Found %d new chapters for '%s'. Queuing for download.", len(newChapters), sub.SeriesTitle)
		err := s.st.AddChaptersToQueue(sub.SeriesTitle, sub.ProviderID, newChapters)
		if err != nil {
			log.Printf("Subscription Check Error: Failed to queue new chapters for '%s': %v", sub.SeriesTitle, err)
		}
	} else {
		log.Printf("No new chapters found for '%s'.", sub.SeriesTitle)
	}

	// Update the last checked time
	s.st.UpdateSubscriptionLastChecked(sub.ID)
}
