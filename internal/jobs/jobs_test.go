package jobs

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

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

func TestRunPruneDatabase(t *testing.T) {
	app := setupTestApp(t)
	// Add a dummy chapter to the DB that doesn't exist on disk
	_, err := app.DB.Exec(
		"INSERT INTO series (id, title, path, created_at, updated_at) VALUES (1, 'Prune Test', '/tmp', ?, ?)",
		time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert series: %v", err)
	}
	_, err = app.DB.Exec(
		"INSERT INTO chapters (series_id, path, page_count, created_at, updated_at) VALUES (1, '/non/existent/path.cbz', 10, ?, ?)",
		time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert chapter: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	var receivedUpdate ProgressUpdate // Use the local jobs.ProgressUpdate struct
	go func() {
		defer wg.Done()
		// Listen for the final broadcast from the job
		msg := <-app.WsHub.Broadcast
		json.Unmarshal(msg, &receivedUpdate)
		t.Logf("Received update: %v", receivedUpdate)
		for !receivedUpdate.Done {
			msg := <-app.WsHub.Broadcast
			json.Unmarshal(msg, &receivedUpdate)
		}
	}()

	RunPruneDatabase(app)
	wg.Wait()

	if !receivedUpdate.Done {
		t.Error("Expected final progress update to have Done=true")
	}
	if !strings.Contains(receivedUpdate.Message, "Removed 1") {
		t.Errorf("Expected final message to report 1 removal, got: %s", receivedUpdate.Message)
	}

	// Verify the chapter was deleted
	var count int
	app.DB.QueryRow("SELECT COUNT(*) FROM chapters").Scan(&count)
	if count != 0 {
		t.Errorf("Expected database to have 0 chapters after prune, but got %d", count)
	}
}
