// It defines the API server, sets up the routes (endpoints)
// using chi, and links them to the handler functions.

package api

import (
	"database/sql"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/vrsandeep/mango-go/internal/core"
	"github.com/vrsandeep/mango-go/internal/store"
)

// Server holds the dependencies for our API.
type Server struct {
	app *core.App
	// config *config.Config
	db    *sql.DB
	store *store.Store
}

// NewServer creates a new Server instance.
func NewServer(app *core.App) *Server {
	return &Server{
		app: app,
		// config: cfg,
		db:    app.DB,
		store: store.New(app.DB),
	}
}

// Router sets up and returns the main router for the application.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)    // Logs requests to the console
	r.Use(middleware.Recoverer) // Recovers from panics
	r.Use(middleware.Timeout(60 * time.Second))

	// Add a file server for the /static directory
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "web", "static"))
	FileServer(r, "/static", filesDir)

	// API routes
	r.Post("/api/users/login", s.handleLogin)
	r.Get("/api/version", s.handleGetVersion)
	r.Group(func(r chi.Router) {
		r.Use(s.AuthMiddleware)

		r.Post("/api/users/logout", s.handleLogout)
		r.Get("/api/users/me", s.handleGetMe)

		r.Route("/api", func(r chi.Router) {
			r.Get("/home", s.handleGetHomePageData)

			r.Get("/series", s.handleListSeries)
			r.Get("/series/{seriesID}", s.handleGetSeries)
			r.Post("/series/{seriesID}/cover", s.handleUpdateCover)
			r.Post("/series/{seriesID}/mark-all-as", s.handleMarkAllAs)
			r.Post("/series/{seriesID}/settings", s.handleUpdateSettings)
			r.Post("/series/{seriesID}/tags", s.handleAddTag)
			r.Delete("/series/{seriesID}/tags/{tagID}", s.handleRemoveTag)
			r.Get("/series/{seriesID}/chapters/{chapterID}", s.handleGetChapterDetails)
			r.Get("/series/{seriesID}/chapters/{chapterID}/pages/{pageNumber}", s.handleGetPage)
			r.Get("/series/{seriesID}/chapters/{chapterID}/neighbors", s.handleGetChapterNeighbors)
			r.Post("/chapters/{chapterID}/progress", s.handleUpdateProgress)

			// New Tag Endpoints
			r.Get("/tags", s.handleListTags)
			r.Get("/tags/{tagID}", s.handleGetTagDetails) // To get a single tag's name
			r.Get("/tags/{tagID}/series", s.handleListSeriesByTag)

			// Admin Job Triggers
			r.Route("/admin", func(r chi.Router) {
				r.Use(s.AdminOnlyMiddleware)

				r.Post("/scan-library", s.handleScanLibrary)
				r.Post("/scan-incremental", s.handleScanIncremental)
				r.Post("/prune-database", s.handlePruneDatabase)
				r.Post("/generate-thumbnails", s.handleGenerateThumbnails)

				// New User Management Routes
				r.Get("/users", s.handleAdminListUsers)
				r.Post("/users", s.handleAdminCreateUser)
				r.Put("/users/{userID}", s.handleAdminUpdateUser)
				r.Delete("/users/{userID}", s.handleAdminDeleteUser)
			})

			// Downloader Routes
			r.Get("/providers", s.handleListProviders)
			r.Get("/providers/{providerID}/search", s.handleProviderSearch)
			r.Get("/providers/{providerID}/series/{seriesIdentifier}", s.handleProviderGetChapters)
			r.Post("/downloads/queue", s.handleAddChaptersToQueue)
			r.Get("/downloads/queue", s.handleGetDownloadQueue)
			r.Post("/downloads/action", s.handleQueueAction)

			// Subscription Routes
			r.Post("/subscriptions", s.handleSubscribeToSeries)
			r.Get("/subscriptions", s.handleListSubscriptions)
			r.Post("/subscriptions/{subID}/recheck", s.handleRecheckSubscription)
			r.Delete("/subscriptions/{subID}", s.handleDeleteSubscription)
		})
	})

	// WebSocket route
	r.Get("/ws/admin/progress", func(w http.ResponseWriter, r *http.Request) {
		s.app.WsHub.ServeWs(w, r)
	})

	// Frontend Routes
	r.Get("/login", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/login.html")
	})
	r.Get("/admin/users", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/admin_users.html")
	})

	// Downloader Frontend Routes
	r.Route("/downloads", func(r chi.Router) {
		r.Get("/plugins", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./web/plugins.html")
		})
		r.Get("/manager", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./web/download_manager.html")
		})
		r.Get("/subscriptions", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./web/subscription_manager.html")
		})
	})

	r.Get("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/admin.html")
	})

	r.Get("/series/{seriesID}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/chapters.html")
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/home.html")
	})

	r.Get("/library", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/series.html")
	})

	r.Get("/library", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/series.html")
	})

	// Tag Frontend Routes
	r.Get("/tags", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/tags.html")
	})
	r.Get("/tags/{tagID}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/tag_series.html")
	})

	r.Get("/reader/series/{seriesID}/chapters/{chapterID}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/reader.html")
	})

	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/static/images/favicon.ico")
	})

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if err := s.db.Ping(); err != nil {
			RespondWithError(w, http.StatusServiceUnavailable, "Database connection failed")
			return
		}
		RespondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	return r
}

// FileServer conveniently sets up a static file server that doesn't list directories.
func FileServer(r chi.Router, path string, root http.FileSystem) {
	fs := http.StripPrefix(path, http.FileServer(root))
	r.Get(path+"*", func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	})
}
