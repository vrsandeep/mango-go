package jobs_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/core"
	"github.com/vrsandeep/mango-go/internal/jobs"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/testutil"
	"github.com/vrsandeep/mango-go/internal/websocket"
)

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

func TestRunFullScan(t *testing.T) {
	app := setupTestApp(t)
	// Create a temporary directory and add a test CBZ file
	testutil.CreateTestCBZFile(t, app.Config.Library.Path, "test.cbz")
	// Run the full scan job
	go jobs.RunFullScan(app)
	// Listen for progress updates with a timeout to prevent the test from hanging
	var lastUpdate models.ProgressUpdate
	timeout := time.After(5 * time.Second)
	for {
		select {
		case msgBytes := <-app.WsHub.Broadcast():
			var update models.ProgressUpdate
			if err := json.Unmarshal(msgBytes, &update); err != nil {
				t.Fatalf("Failed to unmarshal progress update: %v", err)
			}
			lastUpdate = update
			// If we've received the final "done" message, we can stop listening.
			if lastUpdate.Done {
				goto verification
			} else {
				t.Logf("%s", lastUpdate.Message)
			}
		case <-timeout:
			t.Fatal("Test timed out waiting for job to complete")
		}
	}
verification:
	if !lastUpdate.Done {
		t.Error("Expected final progress update to have Done=true")
	}
	if lastUpdate.Progress < 100 {
		t.Errorf("Expected final progress update to have Progress=100, got %.2f", lastUpdate.Progress)
	}
	if lastUpdate.JobName != "Full Scan" {
		t.Errorf("Expected job name 'Full Scan', got '%s'", lastUpdate.JobName)
	}
	if lastUpdate.Message != "Full scan completed successfully." {
		t.Errorf("Expected final message 'Full scan completed successfully.', got '%s'", lastUpdate.Message)
	}
	// Verify the chapter was added to the database
	var count int
	if err := app.DB.QueryRow("SELECT COUNT(*) FROM series").Scan(&count); err != nil {
		t.Fatalf("Failed to query series count: %v", err)
	}
	if count == 0 {
		t.Error("Expected at least one series to be added to the database")
	}

}

func TestRunIncrementalScan(t *testing.T) {
	app := setupTestApp(t)

	// Create a series directory
	seriesDir := app.Config.Library.Path + "/Prune Test"
	err := os.MkdirAll(seriesDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create series directory: %v", err)
	}
	// Create new chapter file under the series directory
	newCbzFile := "new-chapter.cbz"
	testutil.CreateTestCBZFile(t, seriesDir, newCbzFile)

	// Run the incremental scan job
	go jobs.RunIncrementalScan(app)
	// Listen for progress updates with a timeout to prevent the test from hanging
	var lastUpdate models.ProgressUpdate
	timeout := time.After(5 * time.Second)
	for {
		select {
		case msgBytes := <-app.WsHub.Broadcast():
			var update models.ProgressUpdate
			if err := json.Unmarshal(msgBytes, &update); err != nil {
				t.Fatalf("Failed to unmarshal progress update: %v", err)
			}
			lastUpdate = update
			// If we've received the final "done" message, we can stop listening.
			if lastUpdate.Done {
				goto verification
			} else {
				t.Logf("%s", lastUpdate.Message)
			}
		case <-timeout:
			t.Fatal("Test timed out waiting for job to complete")
		}
	}
verification:
	if !lastUpdate.Done {
		t.Error("Expected final progress update to have Done=true")
	}
	if lastUpdate.Progress < 100 {
		t.Errorf("Expected final progress update to have Progress=100, got %.2f", lastUpdate.Progress)
	}
	if lastUpdate.JobName != "Incremental Scan" {
		t.Errorf("Expected job name 'Incremental Scan', got '%s'", lastUpdate.JobName)
	}
	if lastUpdate.Message != "Incremental scan completed." {
		t.Errorf("Expected final message 'Incremental scan completed.', got '%s'", lastUpdate.Message)
	}
	// Verify the chapter was added to the database
	var count int
	if err := app.DB.QueryRow("SELECT COUNT(*) FROM series").Scan(&count); err != nil {
		t.Fatalf("Failed to query series count: %v", err)
	}
	if count != 1 {
		t.Fatalf("Expected at least one series to be added to the database %d", count)
	}
	if err := app.DB.QueryRow("SELECT COUNT(*) FROM chapters").Scan(&count); err != nil {
		t.Fatalf("Failed to query chapters count: %v", err)
	}
	if count != 1 {
		t.Fatalf("Expected one chapter to be added to the database %d", count)
	}
}

// func TestRunPruneDatabase(t *testing.T) {
// 	app := setupTestApp(t)
// 	// Add a dummy chapter to the DB that doesn't exist on disk
// 	_, err := app.DB.Exec(
// 		"INSERT INTO series (id, title, path, created_at, updated_at) VALUES (1, 'Prune Test', '/tmp', ?, ?)",
// 		time.Now(), time.Now())
// 	if err != nil {
// 		t.Fatalf("Failed to insert series: %v", err)
// 	}
// 	_, err = app.DB.Exec(
// 		"INSERT INTO chapters (series_id, path, page_count, created_at, updated_at) VALUES (1, '/non/existent/path.cbz', 10, ?, ?)",
// 		time.Now(), time.Now())
// 	if err != nil {
// 		t.Fatalf("Failed to insert chapter: %v", err)
// 	}

// 	go RunPruneDatabase(app)

// 	// Listen for progress updates with a timeout to prevent the test from hanging
// 	var lastUpdate websocket.ProgressUpdate
// 	timeout := time.After(5 * time.Second)

// 	for {
// 		// Use a select block to avoid deadlocking
// 		select {
// 		case msgBytes := <-app.WsHub.Broadcast():
// 			var update websocket.ProgressUpdate
// 			if err := json.Unmarshal(msgBytes, &update); err != nil {
// 				t.Fatalf("Failed to unmarshal progress update: %v", err)
// 			}
// 			lastUpdate = update
// 			// If we've received the final "done" message, we can stop listening.
// 			if lastUpdate.Done {
// 				goto verification
// 			} else {
// 				t.Logf("%s", lastUpdate.Message)
// 			}
// 		case <-timeout:
// 			t.Fatal("Test timed out waiting for job to complete")
// 		}
// 	}

// verification:
// 	// Verify the final message
// 	if !lastUpdate.Done {
// 		t.Error("Expected final progress update to have Done=true")
// 	}
// 	if !strings.Contains(lastUpdate.Message, "Removed 1") {
// 		t.Errorf("Expected final message to report 1 removal, got: %s", lastUpdate.Message)
// 	}

// 	// Verify the chapter was deleted
// 	var count int
// 	app.DB.QueryRow("SELECT COUNT(*) FROM chapters").Scan(&count)
// 	if count != 0 {
// 		t.Errorf("Expected database to have 0 chapters after prune, but got %d", count)
// 	}
// }
