// A NEW file to hold a shared test server setup utility, which simplifies all API tests.

package testutil

// SetupTestServer initializes a full core.App and api.Server for integration testing.
// func SetupTestServer(t *testing.T) (*api.Server, *sql.DB) {
// 	t.Helper()
// 	db := SetupTestDB(t)

// 	cfg := &config.Config{}
// 	hub := websocket.NewHub()
// 	go hub.Run()
// 	app := &core.App{
// 		Config:  cfg,
// 		DB:      db,
// 		WsHub:   hub,
// 		Version: "test",
// 	}
// 	// Register providers for the test environment
// 	// providers.Register(mockadex.New())
// 	server := api.NewServer(app)
// 	return server, db
// }
