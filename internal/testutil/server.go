// A NEW file to hold a shared test server setup utility, which simplifies all API tests.

package testutil

import (
	"database/sql"
	"testing"

	"github.com/vrsandeep/mango-go/internal/api"
	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/core"
	"github.com/vrsandeep/mango-go/internal/websocket"
)

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
	// Register providers for the test environment
	// providers.Register(mockadex.New())
	server := api.NewServer(app)
	return server, db
}


// SetupTestServer initializes a full core.App and api.Server for integration testing.
// func SetupTestServerEmbedded(t *testing.T) *api.Server {
// 	t.Helper()

// 	db := SetupTestDB(t)

// 	// Pass the real embedded filesystems to the test app instance.
// 	app, err := core.New(embed.WebFS, embed.MigrationsFS)
// 	if err != nil {
// 		t.Fatalf("Failed to create core app for test server: %v", err)
// 	}
// 	// Override the DB with the in-memory test DB
// 	app.DB = db

// 	// Register providers for the test environment
// 	providers.Register(mockadex.New())

// 	return api.NewServer(app)
// }
