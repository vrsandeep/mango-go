package core

import (
	"database/sql"
	"embed"
	"fmt"
	"log"

	"github.com/vrsandeep/mango-go/internal/assets"
	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/db"
	"github.com/vrsandeep/mango-go/internal/jobs"
	"github.com/vrsandeep/mango-go/internal/library"
	"github.com/vrsandeep/mango-go/internal/websocket"
)

const Version = "0.0.5" // Application version

// App holds the core components of the application that are shared
// between the server and the CLI.
type App struct {
	config       *config.Config
	dB           *sql.DB
	wsHub        *websocket.Hub
	Version      string
	WebFS        embed.FS
	MigrationsFS embed.FS
	jobManager   *jobs.JobManager
}

func (a *App) DB() *sql.DB                               { return a.dB }
func (a *App) Config() *config.Config                    { return a.config }
func (a *App) WsHub() *websocket.Hub                     { return a.wsHub }
func (a *App) JobManager() *jobs.JobManager              { return a.jobManager }
func (a *App) SetConfig(cfg *config.Config)              { a.config = cfg }
func (a *App) SetDB(db *sql.DB)                          { a.dB = db }
func (a *App) SetWsHub(hub *websocket.Hub)               { a.wsHub = hub }
func (a *App) SetJobManager(jobManager *jobs.JobManager) { a.jobManager = jobManager }

// New sets up and returns a new App instance. It handles loading the
// configuration, initializing the database connection, and running migrations.
func New() (*App, error) {
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
	if err := db.RunMigrations(database, assets.MigrationsFS); err != nil {
		// We can't proceed without a valid database schema.
		// Close the DB connection before failing.
		database.Close()
		return nil, fmt.Errorf("failed to run database migrations: %w", err)
	}

	log.Println("Core application setup complete.")

	// Create and start the WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	app := &App{
		config:  cfg,
		dB:      database,
		wsHub:   hub,
		Version: Version,
	}

	jobManager := jobs.NewManager(app)
	app.jobManager = jobManager
	app.jobManager.Register("library-sync", "Library Sync", library.LibrarySync)
	app.jobManager.Register("regen-thumbnails", "Regenerate Thumbnails", library.RegenerateThumbnails)
	app.jobManager.Register("delete-empty-tags", "Delete Empty Tags", library.DeleteEmptyTags)
	app.jobManager.Register("detect-bad-files", "Detect Bad Files", library.DetectBadFiles)
	return app, nil
}

// Close gracefully closes the application's resources, like the DB connection.
func (a *App) Close() {
	if a.dB != nil {
		a.dB.Close()
	}
}
