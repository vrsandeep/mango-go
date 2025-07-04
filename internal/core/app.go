package core

import (
	"database/sql"
	"embed"
	"fmt"
	"log"

	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/db"
	"github.com/vrsandeep/mango-go/internal/websocket"
)

const Version = "0.0.1" // Application version

// App holds the core components of the application that are shared
// between the server and the CLI.
type App struct {
	Config  *config.Config
	DB      *sql.DB
	WsHub   *websocket.Hub
	Version string
	WebFS        embed.FS
	MigrationsFS embed.FS
}

// New sets up and returns a new App instance. It handles loading the
// configuration, initializing the database connection, and running migrations.
func New(webFS, migrationsFS embed.FS) (*App, error) {
	// Load configuration from config.yml
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize the database connection
	database, err := db.InitDB(cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Run database migrations
	if err := db.RunMigrations(database, migrationsFS); err != nil {
		// We can't proceed without a valid database schema.
		// Close the DB connection before failing.
		database.Close()
		return nil, fmt.Errorf("failed to run database migrations: %w", err)
	}

	log.Println("Core application setup complete.")

	// Create and start the WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	return &App{
		Config:  cfg,
		DB:      database,
		WsHub:   hub,
		Version: Version,
		WebFS:        webFS,
		MigrationsFS: migrationsFS,
	}, nil
}

// Close gracefully closes the application's resources, like the DB connection.
func (a *App) Close() {
	if a.DB != nil {
		a.DB.Close()
	}
}
