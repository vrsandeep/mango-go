package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/vrsandeep/mango-go/internal/api"
	"github.com/vrsandeep/mango-go/internal/core"
	"github.com/vrsandeep/mango-go/internal/downloader"
	"github.com/vrsandeep/mango-go/internal/downloader/providers"
	"github.com/vrsandeep/mango-go/internal/downloader/providers/mangadex"
	"github.com/vrsandeep/mango-go/internal/downloader/providers/weebcentral"
	"github.com/vrsandeep/mango-go/internal/library"
	"github.com/vrsandeep/mango-go/internal/subscription"
)

func main() {
	// Initialize the core application components
	app, err := core.New()
	if err != nil {
		log.Fatalf("Fatal error during application setup: %v", err)
	}
	defer app.Close()

	// Initial library scan on startup
	log.Println("Performing initial library scan...")
	scanner := library.NewScanner(app.Config, app.DB)

	if err := scanner.Scan(nil, nil); err != nil {
		log.Printf("Warning: initial library scan failed: %v", err)
	}
	log.Println("Initial scan complete.")

	// Start periodic scanning in the background
	go func() {
		ticker := time.NewTicker(time.Duration(app.Config.ScanInterval) * time.Minute)
		for range ticker.C {
			log.Println("Performing periodic library scan...")
			if err := scanner.Scan(nil, nil); err != nil {
				log.Printf("Warning: periodic library scan failed: %v", err)
			}
			log.Println("Periodic scan complete.")
		}
	}()

	// Initialize the downloader providers
	// Register all available downloader providers here.
	// providers.Register(mockadex.New())
	providers.Register(mangadex.New())
	providers.Register(weebcentral.New())

	// Start the download worker pool
	downloader.StartWorkerPool(app)

	// Start the subscription service
	subService := subscription.NewService(app)
	subService.Start()

	// Setup the API server
	server := api.NewServer(app)
	addr := fmt.Sprintf(":%d", app.Config.Port)
	log.Printf("Starting web server on %s", addr)

	// Start the HTTP server
	if err := http.ListenAndServe(addr, server.Router()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
