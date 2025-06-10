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

	// A good base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)    // Logs requests to the console
	r.Use(middleware.Recoverer) // Recovers from panics
	r.Use(middleware.Timeout(60 * time.Second))

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/series", s.handleListSeries)
		r.Get("/series/{seriesID}", s.handleGetSeries)
		r.Get("/series/{seriesID}/chapters/{chapterID}/pages/{pageNumber}", s.handleGetPage)
	})

	return r
}
