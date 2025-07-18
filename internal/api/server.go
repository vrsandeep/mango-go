// It defines the API server, sets up the routes (endpoints)
// using chi, and links them to the handler functions.

package api

import (
	"database/sql"
	"io"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/vrsandeep/mango-go/internal/assets"
	"github.com/vrsandeep/mango-go/internal/core"
	"github.com/vrsandeep/mango-go/internal/store"
)

// Server holds the dependencies for our API.
type Server struct {
	app   *core.App
	db    *sql.DB
	store *store.Store
}

// Store returns the store instance.
func (s *Server) Store() *store.Store {
	return s.store
}

// NewServer creates a new Server instance.
func NewServer(app *core.App) *Server {
	return &Server{
		app:   app,
		db:    app.DB(),
		store: store.New(app.DB()),
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

	// API routes
	r.Post("/api/users/login", s.handleLogin)
	r.Get("/api/version", s.handleGetVersion)
	r.Group(func(r chi.Router) {
		r.Use(s.AuthMiddleware)

		r.Post("/api/users/logout", s.handleLogout)
		r.Get("/api/users/me", s.handleGetMe)

		r.Route("/api", func(r chi.Router) {
			r.Get("/home", s.handleGetHomePageData)

			// r.Get("/series", s.handleListSeries)
			// r.Get("/series/{seriesID}", s.handleGetSeries)
			// r.Post("/series/{seriesID}/cover", s.handleUpdateCover)
			// r.Post("/series/{seriesID}/mark-all-as", s.handleMarkAllAs)
			// r.Post("/series/{seriesID}/settings", s.handleUpdateSettings)
			// r.Post("/series/{seriesID}/tags", s.handleAddTag)
			// r.Delete("/series/{seriesID}/tags/{tagID}", s.handleRemoveTag)
			// r.Get("/series/{seriesID}/chapters/{chapterID}", s.handleGetChapterDetails)
			// r.Get("/series/{seriesID}/chapters/{chapterID}/pages/{pageNumber}", s.handleGetPage)
			r.Post("/chapters/{chapterID}/progress", s.handleUpdateProgress)

			// Browse Routes
			r.Get("/browse", s.handleBrowseFolder)
			r.Get("/browse/breadcrumb", s.handleGetBreadcrumb)

			r.Post("/folders/{folderID}/settings", s.handleUpdateFolderSettings)
			r.Post("/folders/{folderID}/mark-all-as", s.handleMarkFolderAs)
			r.Post("/folders/{folderID}/cover", s.handleUploadFolderCover)
			r.Get("/folders/{folderID}/chapters/{chapterID}/neighbors", s.handleGetChapterNeighbors)

			r.Get("/chapters/{chapterID}/pages/{pageNumber}", s.handleGetPage)
			r.Get("/chapters/{chapterID}", s.handleGetChapterDetails)

			// Folder Tagging Routes
			r.Post("/folders/{folderID}/tags", s.handleAddTagToFolder)
			r.Delete("/folders/{folderID}/tags/{tagID}", s.handleRemoveTagFromFolder)

			// Tag Endpoints
			r.Get("/tags", s.handleListTags)
			r.Get("/tags/{tagID}", s.handleGetTagDetails) // To get a single tag's name
			// r.Get("/tags/{tagID}/series", s.handleListSeriesByTag)
			r.Get("/tags/{tagID}/folders", s.handleListFoldersByTag)

			// Admin Job Triggers
			r.Route("/admin", func(r chi.Router) {
				r.Use(s.AdminOnlyMiddleware)

				r.Get("/jobs/status", s.handleGetAdminJobsStatus)
				r.Post("/jobs/run", s.handleRunAdminJob)

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
		s.app.WsHub().ServeWs(w, r)
	})

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if err := s.db.Ping(); err != nil {
			RespondWithError(w, http.StatusServiceUnavailable, "Database connection failed")
			return
		}
		RespondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Frontend Routes
	webSubFS, err := fs.Sub(assets.WebFS, "web")
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

	// This handler serves a specific HTML file from the embedded FS.
	serveHTML := func(fileName string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			file, err := webSubFS.Open(fileName)
			if err != nil {
				http.NotFound(w, r)
				log.Printf("Error serving embedded file %s: %v", fileName, err)
				return
			}
			http.ServeContent(w, r, fileName, time.Time{}, file.(io.ReadSeeker))
		}
	}

	r.Get("/", serveHTML("home.html"))
	r.Get("/login", serveHTML("login.html"))
	r.Get("/library", serveHTML("library.html"))
	// r.Get("/library", serveHTML("series.html"))
	r.Get("/tags", serveHTML("tags.html"))
	r.Get("/admin", serveHTML("admin.html"))
	r.Get("/admin/users", serveHTML("admin_users.html"))
	r.Get("/downloads/plugins", serveHTML("plugins.html"))
	r.Get("/downloads/manager", serveHTML("download_manager.html"))
	r.Get("/downloads/subscriptions", serveHTML("subscription_manager.html"))

	// Dynamic routes that serve a specific base HTML file
	r.Get("/library/folder/{folderID}", serveHTML("library.html"))
	// r.Get("/series/{id}", serveHTML("chapters.html"))
	// r.Get("/tags/{id}", serveHTML("tag_series.html"))
	r.Get("/tags/{id}", serveHTML("tag_folders.html"))
	r.Get("/reader/series/{seriesID}/chapters/{chapterID}", serveHTML("reader.html"))

	return r
}

// FileServer conveniently sets up a static file server that doesn't list directories.
func FileServer(r chi.Router, path string, root http.FileSystem) {
	fs := http.StripPrefix(path, http.FileServer(root))
	r.Get(path+"*", func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	})
}
