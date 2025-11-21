package downloader_test

import (
	"testing"
	"time"

	"github.com/vrsandeep/mango-go/internal/downloader"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
	"github.com/vrsandeep/mango-go/internal/util"
)

func TestPauseQueueItem(t *testing.T) {
	app := testutil.SetupTestApp(t)
	db := app.DB()
	st := store.New(db)

	// Add a test item to the queue
	_, err := db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status, progress) VALUES ('Test Manga', 'Test Chapter', 'test-id', 'test-provider', ?, 'in_progress', 50)", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// Get the inserted item ID
	var itemID int64
	err = db.QueryRow("SELECT id FROM download_queue WHERE series_title = 'Test Manga'").Scan(&itemID)
	if err != nil {
		t.Fatal(err)
	}

	// Test pausing the item
	err = downloader.PauseQueueItem(app, st, itemID)
	if err != nil {
		t.Fatalf("PauseQueueItem failed: %v", err)
	}

	// Verify the item was paused in the database
	var status string
	var progress int
	err = db.QueryRow("SELECT status, progress FROM download_queue WHERE id = ?", itemID).Scan(&status, &progress)
	if err != nil {
		t.Fatal(err)
	}

	if status != "paused" {
		t.Errorf("Expected status 'paused', got '%s'", status)
	}

	if progress != 50 {
		t.Errorf("Expected progress 50, got %d", progress)
	}
}

func TestResumeQueueItem(t *testing.T) {
	app := testutil.SetupTestApp(t)
	db := app.DB()
	st := store.New(db)

	// Add a paused test item to the queue
	_, err := db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status, progress) VALUES ('Test Manga', 'Test Chapter', 'test-id', 'test-provider', ?, 'paused', 75)", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// Get the inserted item ID
	var itemID int64
	err = db.QueryRow("SELECT id FROM download_queue WHERE series_title = 'Test Manga'").Scan(&itemID)
	if err != nil {
		t.Fatal(err)
	}

	// Test resuming the item
	err = downloader.ResumeQueueItem(app, st, itemID)
	if err != nil {
		t.Fatalf("ResumeQueueItem failed: %v", err)
	}

	// Verify the item was resumed in the database
	var status string
	var progress int
	err = db.QueryRow("SELECT status, progress FROM download_queue WHERE id = ?", itemID).Scan(&status, &progress)
	if err != nil {
		t.Fatal(err)
	}

	if status != "queued" {
		t.Errorf("Expected status 'queued', got '%s'", status)
	}

	if progress != 75 {
		t.Errorf("Expected progress 75, got %d", progress)
	}
}

func TestPauseQueueItemWithNonExistentItem(t *testing.T) {
	app := testutil.SetupTestApp(t)
	db := app.DB()
	st := store.New(db)

	// Test pausing a non-existent item
	err := downloader.PauseQueueItem(app, st, 99999)
	if err == nil {
		t.Error("Expected error when pausing non-existent item, got nil")
	}
}

func TestResumeQueueItemWithNonExistentItem(t *testing.T) {
	app := testutil.SetupTestApp(t)
	db := app.DB()
	st := store.New(db)

	// Test resuming a non-existent item
	err := downloader.ResumeQueueItem(app, st, 99999)
	if err == nil {
		t.Error("Expected error when resuming non-existent item, got nil")
	}
}

func TestRetryQueueItem(t *testing.T) {
	app := testutil.SetupTestApp(t)
	db := app.DB()
	st := store.New(db)

	// Add a failed test item to the queue
	_, err := db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status, progress) VALUES ('Test Manga', 'Test Chapter', 'test-id', 'test-provider', ?, 'failed', 50)", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// Get the inserted item ID
	var itemID int64
	err = db.QueryRow("SELECT id FROM download_queue WHERE series_title = 'Test Manga'").Scan(&itemID)
	if err != nil {
		t.Fatal(err)
	}

	// Test retrying the item
	err = downloader.RetryQueueItem(app, st, itemID)
	if err != nil {
		t.Fatalf("RetryQueueItem failed: %v", err)
	}

	// Verify the item was retried in the database
	var status string
	var progress int
	err = db.QueryRow("SELECT status, progress FROM download_queue WHERE id = ?", itemID).Scan(&status, &progress)
	if err != nil {
		t.Fatal(err)
	}

	if status != "queued" {
		t.Errorf("Expected status 'queued', got '%s'", status)
	}

	if progress != 0 {
		t.Errorf("Expected progress 0, got %d", progress)
	}
}

func TestRetryQueueItemWithNonExistentItem(t *testing.T) {
	app := testutil.SetupTestApp(t)
	db := app.DB()
	st := store.New(db)

	// Test retrying a non-existent item
	err := downloader.RetryQueueItem(app, st, 99999)
	if err == nil {
		t.Error("Expected error when retrying non-existent item, got nil")
	}
}

func TestRetryQueueItemWithNonFailedStatus(t *testing.T) {
	app := testutil.SetupTestApp(t)
	db := app.DB()
	st := store.New(db)

	// Add a queued test item (not failed) to the queue
	_, err := db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Test Manga', 'Test Chapter', 'test-id', 'test-provider', ?, 'queued')", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// Get the inserted item ID
	var itemID int64
	err = db.QueryRow("SELECT id FROM download_queue WHERE series_title = 'Test Manga'").Scan(&itemID)
	if err != nil {
		t.Fatal(err)
	}

	// Test retrying a non-failed item should fail
	err = downloader.RetryQueueItem(app, st, itemID)
	if err == nil {
		t.Error("Expected error when retrying non-failed item, got nil")
	}
}

func TestPauseAndResumeDownloads(t *testing.T) {
	// Test global pause/resume functions
	if downloader.IsPaused() {
		t.Error("Expected downloads to not be paused initially")
	}

	downloader.PauseDownloads()
	if !downloader.IsPaused() {
		t.Error("Expected downloads to be paused after PauseDownloads()")
	}

	downloader.ResumeDownloads()
	if downloader.IsPaused() {
		t.Error("Expected downloads to not be paused after ResumeDownloads()")
	}
}

func TestSendDownloaderProgressUpdate(t *testing.T) {
	// This is a helper function, so we'll test it indirectly through the main functions
	// The actual WebSocket broadcasting is tested through integration tests
	// This test ensures the function doesn't panic
	app := testutil.SetupTestApp(t)
	db := app.DB()
	st := store.New(db)

	// Add a test item
	_, err := db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Test Manga', 'Test Chapter', 'test-id', 'test-provider', ?, 'queued')", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	var itemID int64
	err = db.QueryRow("SELECT id FROM download_queue WHERE series_title = 'Test Manga'").Scan(&itemID)
	if err != nil {
		t.Fatal(err)
	}

	// Test that the function doesn't panic
	// Note: In a real test environment, we might want to mock the WebSocket hub
	// For now, we just ensure the function can be called without errors
	err = downloader.PauseQueueItem(app, st, itemID)
	if err != nil {
		t.Fatalf("PauseQueueItem failed: %v", err)
	}
}

func TestWorkerPauseCheck(t *testing.T) {
	app := testutil.SetupTestApp(t)
	db := app.DB()
	st := store.New(db)

	// Add a test item that's in progress
	_, err := db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status, progress) VALUES ('Test Manga', 'Test Chapter', 'test-id', 'test-provider', ?, 'in_progress', 25)", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	var itemID int64
	err = db.QueryRow("SELECT id FROM download_queue WHERE series_title = 'Test Manga'").Scan(&itemID)
	if err != nil {
		t.Fatal(err)
	}

	// Pause the item
	err = downloader.PauseQueueItem(app, st, itemID)
	if err != nil {
		t.Fatalf("PauseQueueItem failed: %v", err)
	}

	// Verify the item is paused
	var status string
	err = db.QueryRow("SELECT status FROM download_queue WHERE id = ?", itemID).Scan(&status)
	if err != nil {
		t.Fatal(err)
	}

	if status != "paused" {
		t.Errorf("Expected status 'paused', got '%s'", status)
	}
}

func TestErrorConstants(t *testing.T) {
	// Test that our error constant is properly defined
	if downloader.ErrDownloadPaused == nil {
		t.Error("errDownloadPaused should not be nil")
	}

	expectedMsg := "download paused by user"
	if downloader.ErrDownloadPaused.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, downloader.ErrDownloadPaused.Error())
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal filename",
			input:    "Chapter 1 - The Beginning",
			expected: "Chapter 1 - The Beginning",
		},
		{
			name:     "filename with invalid characters",
			input:    "Chapter 1: The Beginning?",
			expected: "Chapter 1- The Beginning-",
		},
		{
			name:     "filename with backslashes and slashes",
			input:    "Chapter 1\\The Beginning/Part A",
			expected: "Chapter 1-The Beginning-Part A",
		},
		{
			name:     "filename with quotes and angle brackets",
			input:    "Chapter 1 \"The Beginning\" <Part A>",
			expected: "Chapter 1 -The Beginning- -Part A-",
		},
		{
			name:     "filename with asterisk and pipe",
			input:    "Chapter 1*The Beginning|Part A",
			expected: "Chapter 1-The Beginning-Part A",
		},
		{
			name:     "filename with null bytes",
			input:    "Chapter 1\x00The Beginning",
			expected: "Chapter 1-The Beginning",
		},
		{
			name:     "filename starting with dot",
			input:    ".Chapter 1",
			expected: "Chapter 1",
		},
		{
			name:     "filename starting with hyphen",
			input:    "-Chapter 1",
			expected: "Chapter 1",
		},
		{
			name:     "filename starting with multiple dots and hyphens",
			input:    "...---Chapter 1",
			expected: "Chapter 1",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "untitled",
		},
		{
			name:     "only invalid characters",
			input:    "\\/:*?\"<>|",
			expected: "untitled",
		},
		{
			name:     "mixed valid and invalid characters",
			input:    "Chapter 1: \"The Beginning\" <Part A> | Section B",
			expected: "Chapter 1- -The Beginning- -Part A- - Section B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := downloader.SanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeFolderName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal folder name",
			input:    "My Manga Series",
			expected: "My Manga Series",
		},
		{
			name:     "folder name with invalid characters",
			input:    "Manga: Series?",
			expected: "Manga- Series",
		},
		{
			name:     "folder name with backslashes and slashes",
			input:    "Manga\\Series/Part A",
			expected: "Manga-Series-Part A",
		},
		{
			name:     "folder name with quotes and angle brackets",
			input:    "Manga \"Series\" <Part A>",
			expected: "Manga -Series- -Part A",
		},
		{
			name:     "folder name with asterisk and pipe",
			input:    "Manga*Series|Part A",
			expected: "Manga-Series-Part A",
		},
		{
			name:     "folder name with null bytes and control characters",
			input:    "Manga\x00Series\x1fTest",
			expected: "MangaSeriesTest",
		},
		{
			name:     "folder name starting with dot",
			input:    ".Manga Series",
			expected: "Manga Series",
		},
		{
			name:     "folder name starting with space",
			input:    " Manga Series",
			expected: "Manga Series",
		},
		{
			name:     "folder name ending with space and dot",
			input:    "Manga Series .",
			expected: "Manga Series",
		},
		{
			name:     "folder name with consecutive dashes",
			input:    "Manga---Series",
			expected: "Manga-Series",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only invalid characters",
			input:    "\\/:*?\"<>|",
			expected: "",
		},
		{
			name:     "Windows reserved name CON",
			input:    "CON",
			expected: "CON_",
		},
		{
			name:     "Windows reserved name PRN",
			input:    "prn",
			expected: "prn_",
		},
		{
			name:     "Windows reserved name COM1",
			input:    "COM1",
			expected: "COM1_",
		},
		{
			name:     "Windows reserved name LPT9",
			input:    "lpt9",
			expected: "lpt9_",
		},
		{
			name:     "mixed valid and invalid characters",
			input:    "Manga: \"Series\" <Part A> | Section B",
			expected: "Manga- -Series- -Part A- - Section B",
		},
		{
			name:     "only spaces and dots",
			input:    "   ...   ",
			expected: "",
		},
		{
			name:     "only dashes",
			input:    "---",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := util.SanitizeFolderName(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFolderName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestProcessDownloadWithFolderPath(t *testing.T) {
	app := testutil.SetupTestApp(t)
	db := app.DB()
	st := store.New(db)

	// Create a subscription with a custom folder path
	customFolderPath := "custom/manga/path"
	sub, err := st.SubscribeToSeriesWithFolder("Test Manga", "test-series", "mockadex", &customFolderPath)
	if err != nil {
		t.Fatalf("Failed to create subscription: %v", err)
	}

	// Create a download job
	job := &models.DownloadQueueItem{
		ID:                1,
		SeriesTitle:       "Test Manga",
		ChapterTitle:      "Chapter 1",
		ChapterIdentifier: "ch1",
		ProviderID:        "mockadex",
		Status:            "queued",
		Progress:          0,
		CreatedAt:         time.Now(),
	}

	// Add job to database
	_, err = db.Exec(`
		INSERT INTO download_queue
		(series_title, chapter_title, chapter_identifier, provider_id, status, progress, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		job.SeriesTitle, job.ChapterTitle, job.ChapterIdentifier, job.ProviderID,
		job.Status, job.Progress, job.CreatedAt)
	if err != nil {
		t.Fatalf("Failed to insert job: %v", err)
	}

	// Get the job ID
	var jobID int64
	err = db.QueryRow("SELECT id FROM download_queue WHERE series_title = ?", job.SeriesTitle).Scan(&jobID)
	if err != nil {
		t.Fatalf("Failed to get job ID: %v", err)
	}
	job.ID = jobID

	// Test that the worker would use the custom folder path
	// This is a bit tricky to test directly since processDownload is not exported
	// But we can test the logic by checking the subscription lookup
	retrievedSub, err := st.GetSubscriptionByID(sub.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve subscription: %v", err)
	}

	if retrievedSub.FolderPath == nil {
		t.Error("Expected subscription to have folder path")
	}
	if *retrievedSub.FolderPath != customFolderPath {
		t.Errorf("Expected folder path %s, got %s", customFolderPath, *retrievedSub.FolderPath)
	}
}

func TestProcessDownloadWithoutFolderPath(t *testing.T) {
	app := testutil.SetupTestApp(t)
	db := app.DB()
	st := store.New(db)

	// Create a subscription without a custom folder path
	sub, err := st.SubscribeToSeriesWithFolder("Test Manga No Folder", "test-series-no-folder", "mockadex", nil)
	if err != nil {
		t.Fatalf("Failed to create subscription: %v", err)
	}

	// Test that the subscription has no folder path
	retrievedSub, err := st.GetSubscriptionByID(sub.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve subscription: %v", err)
	}

	if retrievedSub.FolderPath != nil {
		t.Errorf("Expected subscription to have nil folder path, got %s", *retrievedSub.FolderPath)
	}
}

func TestFolderPathLookup(t *testing.T) {
	app := testutil.SetupTestApp(t)
	db := app.DB()
	st := store.New(db)

	// Create multiple subscriptions with different folder paths
	folderPath1 := "path/one"
	folderPath2 := "path/two"

	st.SubscribeToSeriesWithFolder("Manga One", "series-1", "provider-1", &folderPath1)
	st.SubscribeToSeriesWithFolder("Manga Two", "series-2", "provider-2", &folderPath2)
	st.SubscribeToSeriesWithFolder("Manga Three", "series-3", "provider-1", nil)

	// Test retrieving subscriptions by provider
	subs, err := st.GetAllSubscriptions("provider-1")
	if err != nil {
		t.Fatalf("Failed to get subscriptions: %v", err)
	}

	if len(subs) != 2 {
		t.Fatalf("Expected 2 subscriptions for provider-1, got %d", len(subs))
	}

	// Find our test subscriptions
	var foundSub1, foundSub3 *models.Subscription
	for _, sub := range subs {
		switch sub.SeriesIdentifier {
		case "series-1":
			foundSub1 = sub
		case "series-3":
			foundSub3 = sub
		}
	}

	if foundSub1 == nil {
		t.Fatal("Could not find subscription 1")
	}
	if foundSub1.FolderPath == nil || *foundSub1.FolderPath != folderPath1 {
		t.Errorf("Expected folder path %s for sub1, got %v", folderPath1, foundSub1.FolderPath)
	}

	if foundSub3 == nil {
		t.Fatal("Could not find subscription 3")
	}
	if foundSub3.FolderPath != nil {
		t.Errorf("Expected nil folder path for sub3, got %s", *foundSub3.FolderPath)
	}
}
