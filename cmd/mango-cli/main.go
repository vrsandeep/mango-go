package main

import (
	"fmt"
	"log"

	"github.com/vrsandeep/mango-go/internal/core"
	"github.com/vrsandeep/mango-go/internal/library"
)

func main() {
	// Initialize the core application components
	app, err := core.New()
	if err != nil {
		log.Fatalf("Fatal error during application setup: %v", err)
	}
	defer app.Close()

	// Create a new scanner
	scanner := library.NewScanner(app.Config, app.DB)

	log.Printf("Starting scan of library at: %s", app.Config.Library.Path)

	// Scan the library for manga.
	if err := scanner.Scan(nil, nil); err != nil {
		log.Fatalf("Error scanning library: %v", err)
	}

	log.Println("Scan complete. Data has been saved to the database.")
	fmt.Println("Library scan finished successfully.")
}
