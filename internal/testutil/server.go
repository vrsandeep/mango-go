// A NEW file to hold a shared test server setup utility, which simplifies all API tests.

package testutil

import (
	"database/sql"
	"testing"

	"github.com/vrsandeep/mango-go/internal/api"
	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/core"
	"github.com/vrsandeep/mango-go/internal/downloader/providers"
	"github.com/vrsandeep/mango-go/internal/downloader/providers/mockadex"
	"github.com/vrsandeep/mango-go/internal/jobs"
	"github.com/vrsandeep/mango-go/internal/websocket"
)

func SetupTestApp(t *testing.T) *core.App {
	t.Helper()
	db := SetupTestDB(t)

	// cfg := &config.Config{}
	cfg := &config.Config{
		Library: struct {
			Path string `mapstructure:"path"`
		}{Path: t.TempDir()},
	}
	hub := websocket.NewHub()
	go hub.Run()
	app := &core.App{Version: "test"}
	app.SetConfig(cfg)
	app.SetDB(db)
	app.SetWsHub(hub)

	t.Cleanup(func() {
		providers.UnregisterAll()
	})

	// Register providers for the test environment
	providers.Register(mockadex.New())
	jobManager := jobs.NewManager(app)
	app.SetJobManager(jobManager)
	return app
}

// SetupTestServer initializes a full core.App and api.Server for integration testing.
func SetupTestServer(t *testing.T) (*api.Server, *sql.DB, *jobs.JobManager) {
	t.Helper()
	db := SetupTestDB(t)

	cfg := &config.Config{
		Library: struct {
			Path string `mapstructure:"path"`
		}{Path: t.TempDir()},
	}
	hub := websocket.NewHub()
	go hub.Run()
	app := &core.App{Version: "test"}
	app.SetConfig(cfg)
	app.SetDB(db)
	app.SetWsHub(hub)

	t.Cleanup(func() {
		providers.UnregisterAll()
	})

	// Register providers for the test environment
	providers.Register(mockadex.New())
	server := api.NewServer(app)
	jobManager := jobs.NewManager(app)
	app.SetJobManager(jobManager)
	return server, db, jobManager
}
