package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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

	// Run database migrations
	if err := runMigrations(database); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	// Create a new scanner with the database connection
	scanner := library.NewScanner(cfg, database)

	log.Printf("Starting scan of library at: %s", cfg.Library.Path)

	// Scan the library for manga. The scanner now saves to the DB.
	err = scanner.Scan()
	if err != nil {
		log.Fatalf("Error scanning library: %v", err)
	}

	log.Println("Scan complete. Data has been saved to the database.")

	// Example: You could now query the DB to show results, but for the
	// CLI, we'll just confirm completion.
	fmt.Println("Library scan finished successfully.")
}

// runMigrations applies the database migrations.
func runMigrations(database *sql.DB) error {
	driver, err := sqlite3.WithInstance(database, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("could not create sqlite3 migration driver: %w", err)
	}

	// The path to the migrations folder.
	// This relative path assumes you run the CLI from the project root.
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"sqlite3",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	log.Println("Applying database migrations...")
	// Up applies all available "up" migrations.
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("an error occurred while applying migrations: %w", err)
	}

	log.Println("Migrations applied successfully.")
	return nil
}
