package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vrsandeep/mango-go/internal/api"
	"github.com/vrsandeep/mango-go/internal/auth"
	"github.com/vrsandeep/mango-go/internal/core"
	"github.com/vrsandeep/mango-go/internal/downloader"
	"github.com/vrsandeep/mango-go/internal/downloader/providers"
	"github.com/vrsandeep/mango-go/internal/downloader/providers/mangadex"
	"github.com/vrsandeep/mango-go/internal/downloader/providers/weebcentral"
	"github.com/vrsandeep/mango-go/internal/plugins"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/subscription"
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Initialize the core application components
	app, err := core.New()

	if err != nil {
		log.Fatalf("Fatal error during application setup: %v", err)
	}
	defer app.Close()

	// --- First User Provisioning ---
	st := store.New(app.DB())
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
	go app.JobManager().RunJob("library-sync", app)
	go func() {
		ticker := time.NewTicker(time.Duration(app.Config().ScanInterval) * time.Minute)
		for range ticker.C {
			log.Println("Performing periodic library scan...")
			if err := app.JobManager().RunJob("library-sync", app); err != nil {
				log.Printf("Warning: periodic library scan failed: %v", err)
			}
			log.Println("Periodic scan complete.")
		}
	}()

	// Initialize the downloader providers
	// Register all available downloader providers here.
	providers.Register(mangadex.New())
	providers.Register(weebcentral.New())

	// Initialize plugin manager and load plugins
	pluginManager := plugins.NewPluginManager(app, app.Config().Plugins.Path)
	plugins.SetGlobalManager(pluginManager)
	if err := pluginManager.LoadPlugins(); err != nil {
		log.Printf("Warning: failed to load plugins: %v", err)
	}

	// Start the download worker pool
	downloader.StartWorkerPool(app)

	// Start the subscription service
	subService := subscription.NewService(app)
	subService.Start()

	// Setup the API server
	server := api.NewServer(app)
	addr := fmt.Sprintf(":%d", app.Config().Port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: server.Router(),
	}
	// --- Graceful Shutdown ---
	// Start the server in a goroutine so it doesn't block.
	go func() {
		log.Printf("Starting web server on %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not start server: %v", err)
		}
	}()

	// Wait for an interrupt signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create a context with a timeout to allow existing connections to finish.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt a graceful shutdown.
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting.")
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
