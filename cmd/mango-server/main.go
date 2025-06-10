package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/vrsandeep/mango-go/internal/api"
	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/db"
	"github.com/vrsandeep/mango-go/internal/library"
)

func main() {
	// Load configuration from config.yml
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize the database connection
	database, err := db.InitDB(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Run database migrations (same as in CLI)
	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	// Initial library scan on startup
	log.Println("Performing initial library scan...")
	scanner := library.NewScanner(cfg, database)
	if err := scanner.Scan(); err != nil {
		log.Printf("Warning: initial library scan failed: %v", err)
	}
	log.Println("Initial scan complete.")

	// Start periodic scanning in the background
	go func() {
		ticker := time.NewTicker(time.Duration(cfg.ScanInterval) * time.Minute)
		for range ticker.C {
			log.Println("Performing periodic library scan...")
			if err := scanner.Scan(); err != nil {
				log.Printf("Warning: periodic library scan failed: %v", err)
			}
			log.Println("Periodic scan complete.")
		}
	}()

	// Setup the API server
	server := api.NewServer(cfg, database)
	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Starting web server on %s", addr)

	// Start the HTTP server
	if err := http.ListenAndServe(addr, server.Router()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
