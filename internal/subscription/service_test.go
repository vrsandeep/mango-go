package subscription_test

import (
	"testing"
	"time"

	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/core"
	"github.com/vrsandeep/mango-go/internal/downloader/providers"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/subscription"
	"github.com/vrsandeep/mango-go/internal/testutil"
	"github.com/vrsandeep/mango-go/internal/websocket"
)

// MockProvider for testing the subscription service.
type MockSubProvider struct{}

func (p *MockSubProvider) GetInfo() models.ProviderInfo {
	return models.ProviderInfo{ID: "mocksub", Name: "MockSub"}
}
func (p *MockSubProvider) Search(q string) ([]models.SearchResult, error) { return nil, nil }
func (p *MockSubProvider) GetPageURLs(id string) ([]string, error)        { return nil, nil }
func (p *MockSubProvider) GetChapters(id string) ([]models.ChapterResult, error) {
	return []models.ChapterResult{
		{Identifier: "ch1", PublishedAt: time.Now().Add(-2 * time.Hour)}, // Old chapter
		{Identifier: "ch2", PublishedAt: time.Now().Add(1 * time.Hour)},  // New chapter
	}, nil
}

// setupTestApp creates a mock core.App for testing jobs.
func setupTestApp(t *testing.T) *core.App {
	t.Helper()
	hub := websocket.NewHub()
	go hub.Run() // Run the hub in the background

	return &core.App{
		Config: &config.Config{
			Library: struct {
				Path string `mapstructure:"path"`
			}{Path: t.TempDir()},
		},
		DB:      testutil.SetupTestDB(t),
		WsHub:   hub,
		Version: "test",
	}
}
func TestSubscriptionService(t *testing.T) {
	// Clear the provider registry before each test
	providers.UnregisterAll()

	app := setupTestApp(t)
	st := store.New(app.DB)

	// Register our mock provider for this test
	providers.Register(&MockSubProvider{})

	// Create a subscription that was made 1 hour ago
	subTime := time.Now().Add(-1 * time.Hour)
	app.DB.Exec("INSERT INTO subscriptions (id, series_title, series_identifier, provider_id, created_at) VALUES (?, ?, ?, ?, ?)", 1, "Test Sub", "test-id", "mocksub", subTime)

	service := subscription.NewService(app)

	// Run the check for this specific subscription
	service.CheckSingleSubscription(1)

	// Verify the results in the database
	queuedItems, err := st.GetDownloadQueue()
	if err != nil {
		t.Fatalf("Failed to get download queue: %v", err)
	}

	if len(queuedItems) != 1 {
		t.Fatalf("Expected 1 new chapter to be queued, but got %d", len(queuedItems))
	}

	if queuedItems[0].ChapterIdentifier != "ch2" {
		t.Errorf("Expected chapter 'ch2' to be queued, but got '%s'", queuedItems[0].ChapterIdentifier)
	}

	// Verify that the 'last_checked_at' was updated
	sub, _ := st.GetSubscriptionByID(1)
	if sub.LastCheckedAt == nil {
		t.Error("LastCheckedAt was not updated after check")
	}
	if time.Since(*sub.LastCheckedAt) > 5*time.Second {
		t.Error("LastCheckedAt timestamp is not recent")
	}
}
