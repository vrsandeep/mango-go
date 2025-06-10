// This file tests the main library scanner.

package library

import (
	"archive/zip"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/store"
)

// createTestCBZ is a helper function that creates a temporary CBZ file
// for testing purposes. It returns the path to the created file.
// (This helper is from Milestone 1 tests, copied here for completeness)
func createTestCBZFile(t *testing.T, dir, name string) string {
	t.Helper()
	file, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("Failed to create temp cbz file: %v", err)
	}
	defer file.Close()
	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()
	files := []string{"01.jpg", "03.png", "02.jpeg"}
	for _, f := range files {
		_, err := zipWriter.Create(f)
		if err != nil {
			t.Fatalf("Failed to create entry in zip: %v", err)
		}
	}
	return file.Name()
}

// setupTestLibraryAndDB creates a temporary library structure and an in-memory DB.
func setupTestLibraryAndDB(t *testing.T) (string, *sql.DB) {
	t.Helper()
	// Setup DB
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	driver, _ := sqlite3.WithInstance(db, &sqlite3.Config{})
	m, _ := migrate.NewWithDatabaseInstance("file://../../migrations", "sqlite3", driver)
	if err := m.Up(); err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	// Setup Library
	rootDir := t.TempDir()
	seriesADir := filepath.Join(rootDir, "Series A")
	os.Mkdir(seriesADir, 0755)
	createTestCBZFile(t, seriesADir, "ch1.cbz")
	return rootDir, db
}

func TestScannerIntegration(t *testing.T) {
	libraryPath, db := setupTestLibraryAndDB(t)
	defer db.Close()

	// Configure and run the scanner
	cfg := &config.Config{Library: struct {
		Path string `mapstructure:"path"`
	}{Path: libraryPath}}
	scanner := NewScanner(cfg, db)
	if err := scanner.Scan(); err != nil {
		t.Fatalf("scanner.Scan() failed: %v", err)
	}

	// Verify the database state after the scan
	s := store.New(db)
	tx, _ := db.Begin()
	defer tx.Rollback()

	// Check if "Series A" was created
	seriesID, err := s.GetOrCreateSeries(tx, "Series A", filepath.Join(libraryPath, "Series A"))
	if err != nil {
		t.Fatalf("Failed to find 'Series A' in database after scan: %v", err)
	}
	if seriesID == 0 {
		t.Fatal("Series ID should not be 0")
	}

	// Check if the chapter was created correctly
	var pageCount int
	var chapterPath string
	err = tx.QueryRow("SELECT path, page_count FROM chapters WHERE series_id = ?", seriesID).Scan(&chapterPath, &pageCount)
	if err != nil {
		t.Fatalf("Failed to find chapter for 'Series A': %v", err)
	}
	if pageCount != 3 {
		t.Errorf("Expected page count of 3 for scanned chapter, got %d", pageCount)
	}
	expectedPath := filepath.Join(libraryPath, "Series A", "ch1.cbz")
	if chapterPath != expectedPath {
		t.Errorf("Expected chapter path '%s', got '%s'", expectedPath, chapterPath)
	}
}
