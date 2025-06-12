// It defines the API server, sets up the routes (endpoints)
// using chi, and links them to the handler functions.

package api

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/store"
)

// Server holds the dependencies for our API.
type Server struct {
	config *config.Config
	db     *sql.DB
	store  *store.Store
}

// NewServer creates a new Server instance.
func NewServer(cfg *config.Config, db *sql.DB) *Server {
	return &Server{
		config: cfg,
		db:     db,
		store:  store.New(db),
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
	r.Route("/api", func(r chi.Router) {
		r.Get("/series", s.handleListSeries)
		r.Get("/series/{seriesID}", s.handleGetSeries)
		r.Get("/series/{seriesID}/chapters/{chapterID}", s.handleGetChapterDetails)
		r.Get("/series/{seriesID}/chapters/{chapterID}/pages/{pageNumber}", s.handleGetPage)
		// New endpoints for progress tracking
		r.Post("/chapters/{chapterID}/progress", s.handleUpdateProgress)
	})

	// Route to serve the web reader frontend
	r.Get("/reader/series/{seriesID}/chapters/{chapterID}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/reader.html")
	})

	// Serve static files for the web reader
	// r.Handle("/reader/static/*", http.StripPrefix("/reader/static/", http.FileServer(http.Dir("./web/static"))))

	return r
}
