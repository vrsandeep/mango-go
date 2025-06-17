package jobs

import (
	"testing"

	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/core"
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
// 	var lastUpdate ProgressUpdate
// 	timeout := time.After(5 * time.Second)

// 	for {
// 		// Use a select block to avoid deadlocking
// 		select {
// 		case msgBytes := <-app.WsHub.Broadcast():
// 			var update ProgressUpdate
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
