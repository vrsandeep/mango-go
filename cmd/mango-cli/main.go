package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/library"
)

func main() {
	// Basic argument check
	if len(os.Args) < 2 {
		fmt.Println("Usage: mango-cli <path-to-your-manga-library>")
		os.Exit(1)
	}
	libraryPath := os.Args[1]

	// Create a new configuration for the scanner.
	// In the future, this could be loaded from a file.
	cfg := &config.Config{
		LibraryPath: libraryPath,
	}

	log.Printf("Starting scan of library at: %s", cfg.LibraryPath)

	// Scan the library for manga.
	mangaCollection, err := library.ScanLibrary(cfg)
	if err != nil {
		log.Fatalf("Error scanning library: %v", err)
	}

	log.Printf("Scan complete. Found %d manga series.", len(mangaCollection))

	// Print the results as a nicely formatted JSON.
	// This makes it easy to see the structure of the data we've collected.
	output, err := json.MarshalIndent(mangaCollection, "", "  ")
	if err != nil {
		log.Fatalf("Error formatting output: %v", err)
	}

	fmt.Println(string(output))
}
