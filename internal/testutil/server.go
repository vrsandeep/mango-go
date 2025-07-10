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
	"github.com/vrsandeep/mango-go/internal/websocket"
)

func SetupTestApp(t *testing.T) *core.App {
	t.Helper()
	db := SetupTestDB(t)

	cfg := &config.Config{}
	hub := websocket.NewHub()
	go hub.Run()
	app := &core.App{
		Config:  cfg,
		DB:      db,
		WsHub:   hub,
		Version: "test",
	}

	t.Cleanup(func() {
		providers.UnregisterAll()
	})

	// Register providers for the test environment
	providers.Register(mockadex.New())
	return app
}

// SetupTestServer initializes a full core.App and api.Server for integration testing.
func SetupTestServer(t *testing.T) (*api.Server, *sql.DB) {
	t.Helper()
	db := SetupTestDB(t)

	cfg := &config.Config{}
	hub := websocket.NewHub()
	go hub.Run()
	app := &core.App{
		Config:  cfg,
		DB:      db,
		WsHub:   hub,
		Version: "test",
	}

	t.Cleanup(func() {
		providers.UnregisterAll()
	})

	// Register providers for the test environment
	providers.Register(mockadex.New())
	server := api.NewServer(app)
	return server, db
}
