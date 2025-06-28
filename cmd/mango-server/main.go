package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/vrsandeep/mango-go/internal/api"
	"github.com/vrsandeep/mango-go/internal/auth"
	"github.com/vrsandeep/mango-go/internal/core"
	"github.com/vrsandeep/mango-go/internal/downloader"
	"github.com/vrsandeep/mango-go/internal/downloader/providers"
	"github.com/vrsandeep/mango-go/internal/downloader/providers/mangadex"
	"github.com/vrsandeep/mango-go/internal/downloader/providers/weebcentral"
	"github.com/vrsandeep/mango-go/internal/library"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/subscription"
)

func main() {
	// Initialize the core application components
	app, err := core.New()
	if err != nil {
		log.Fatalf("Fatal error during application setup: %v", err)
	}
	defer app.Close()

	// --- First User Provisioning ---
	st := store.New(app.DB)
	userCount, err := st.CountUsers()
	if err != nil {
		log.Fatalf("Could not check user count: %v", err)
	}
	if userCount == 0 {
		log.Println("No users found. Creating default admin account.")
		password := generateRandomPassword(12)
		passwordHash, _ := auth.HashPassword(password)
		_, err := st.CreateUser("admin", passwordHash, "admin")
		if err != nil {
			log.Fatalf("Could not create default admin user: %v", err)
		}
		log.Println("==================================================")
		log.Println("Default admin user created.")
		log.Printf("Username: admin")
		log.Printf("Password: %s", password)
		log.Println("Please change this password immediately.")
		log.Println("==================================================")
	}

	// Start periodic scanning in the background
	scanner := library.NewScanner(app.Config, app.DB)
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

func generateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
