// It defines the API server, sets up the routes (endpoints)
// using chi, and links them to the handler functions.

package api

import (
	"database/sql"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/vrsandeep/mango-go/internal/core"
	"github.com/vrsandeep/mango-go/internal/store"
)

// Server holds the dependencies for our API.
type Server struct {
	app *core.App
	db    *sql.DB
	store *store.Store
}

// NewServer creates a new Server instance.
func NewServer(app *core.App) *Server {
	return &Server{
		app: app,
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

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if err := s.db.Ping(); err != nil {
			RespondWithError(w, http.StatusServiceUnavailable, "Database connection failed")
			return
		}
		RespondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Frontend Routes
	webSubFS, err := fs.Sub(s.app.WebFS, "web")
	if err != nil {
		log.Fatalf("Failed to create web sub-filesystem: %v", err)
	}

	// Create a file server for the static assets within the embedded FS.
	staticFS, err := fs.Sub(webSubFS, "static")
	if err != nil {
		log.Fatalf("Failed to create static sub-filesystem: %v", err)
	}
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Serve the favicon from the embedded FS.
	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		file, _ := staticFS.Open("images/favicon.ico")
		defer file.Close()
		http.ServeContent(w, r, "favicon.ico", time.Time{}, file.(io.ReadSeeker))
	})

	// Serve all the HTML pages from the root of the embedded web FS.
	htmlHandler := func(w http.ResponseWriter, r *http.Request) {
		// Default to serving index.html (our home page)
		filePath := "home.html"

		// Determine which HTML file to serve based on the URL path
		if strings.HasPrefix(r.URL.Path, "/login") { filePath = "login.html" }
		if strings.HasPrefix(r.URL.Path, "/library") { filePath = "series.html" }
		if strings.HasPrefix(r.URL.Path, "/series/") { filePath = "chapters.html" }
		if strings.HasPrefix(r.URL.Path, "/tags") { filePath = "tags.html" }
		if strings.HasPrefix(r.URL.Path, "/tags/") { filePath = "tag_series.html" }
		if strings.HasPrefix(r.URL.Path, "/downloads/plugins") { filePath = "plugins.html" }
		if strings.HasPrefix(r.URL.Path, "/downloads/manager") { filePath = "download_manager.html" }
		if strings.HasPrefix(r.URL.Path, "/downloads/subscriptions") { filePath = "subscription_manager.html" }
		if strings.HasPrefix(r.URL.Path, "/admin") { filePath = "admin.html" }
		if strings.HasPrefix(r.URL.Path, "/admin/users") { filePath = "admin_users.html" }
		if strings.HasPrefix(r.URL.Path, "/reader/") { filePath = "reader.html" }

		file, err := webSubFS.Open(filePath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		http.ServeContent(w, r, filePath, time.Time{}, file.(io.ReadSeeker))
	}
	r.Get("/*", htmlHandler)

	return r
}

// FileServer conveniently sets up a static file server that doesn't list directories.
func FileServer(r chi.Router, path string, root http.FileSystem) {
	fs := http.StripPrefix(path, http.FileServer(root))
	r.Get(path+"*", func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	})
}
