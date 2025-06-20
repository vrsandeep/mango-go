// A NEW file to hold a shared test server setup utility, which simplifies all API tests.

package testutil

// import (
// 	"testing"

// 	"github.com/vrsandeep/mango-go/internal/api"
// 	"github.com/vrsandeep/mango-go/internal/config"
// 	"github.com/vrsandeep/mango-go/internal/core"
// 	"github.com/vrsandeep/mango-go/internal/downloader/providers"
// 	"github.com/vrsandeep/mango-go/internal/downloader/providers/mockadex"
// 	"github.com/vrsandeep/mango-go/internal/websocket"
// )

// // SetupTestServer initializes a full core.App and api.Server for integration testing.
// func SetupTestServer(t *testing.T) *api.Server {
// 	t.Helper()
// 	hub := websocket.NewHub()
// 	go hub.Run()

// 	db := SetupTestDB(t)

// 	app := &core.App{
// 		Config: &config.Config{
// 			Library: struct {
// 				Path string `mapstructure:"path"`
// 			}{Path: t.TempDir()},
// 		},
// 		DB:    db,
// 		WsHub: hub,
// 	}

// 	// Register providers for the test environment
// 	providers.Register(mockadex.New())

// 	return api.NewServer(app)
// }
