// It uses Go's httptest package
// to simulate HTTP requests without needing to run a live server.

package api

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/models"
)

var testServer *Server

// setupTestServer initializes an in-memory database, populates it with test
// data, and sets up a test server instance.
func setupTestServer(t *testing.T) {
	t.Helper()

	// Use in-memory database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}

	// Run migrations
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		t.Fatalf("Failed to create migration driver: %v", err)
	}
	m, err := migrate.NewWithDatabaseInstance("file://../../migrations", "sqlite3", driver)
	if err != nil {
		t.Fatalf("Failed to create migrate instance: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	// Create a temporary directory for test archives
	tempDir := t.TempDir()

	// Populate database with test data
	series1Path := filepath.Join(tempDir, "Series 1")
	os.Mkdir(series1Path, 0755)
	chapter1Path := createTestCBZ(t, series1Path, "ch1.cbz", []string{"page1.jpg", "page2.jpg"})

	_, err = db.Exec(`INSERT INTO series (id, title, path, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		1, "Series 1", series1Path, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test series: %v", err)
	}
	_, err = db.Exec(`INSERT INTO chapters (id, series_id, path, page_count, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		1, 1, chapter1Path, 2, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test chapter: %v", err)
	}

	// Setup the server instance
	cfg := &config.Config{} // Config is not needed for these tests
	testServer = NewServer(cfg, db)
}

// createTestCBZ is a helper to create a test archive.
func createTestCBZ(t *testing.T, dir, name string, pages []string) string {
	t.Helper()
	filePath := filepath.Join(dir, name)
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create temp cbz file: %v", err)
	}
	defer file.Close()
	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()
	for _, page := range pages {
		_, err := zipWriter.Create(page)
		if err != nil {
			t.Fatalf("Failed to create entry in zip: %v", err)
		}
	}
	return filePath
}

func TestMain(m *testing.M) {
	// Setup can be done here if it's shared across all tests in the package
	// For simplicity, we'll set it up in each test function for now.
	os.Exit(m.Run())
}

func TestHandleListSeries(t *testing.T) {
	setupTestServer(t)
	router := testServer.Router()

	req, _ := http.NewRequest("GET", "/api/series", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var seriesList []*models.Series
	if err := json.Unmarshal(rr.Body.Bytes(), &seriesList); err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)
	}
	if len(seriesList) != 1 {
		t.Errorf("Expected 1 series, got %d", len(seriesList))
	}
	if seriesList[0].Title != "Series 1" {
		t.Errorf("Expected series title 'Series 1', got '%s'", seriesList[0].Title)
	}
}

func TestHandleGetSeries(t *testing.T) {
	setupTestServer(t)
	router := testServer.Router()

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/series/1", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		var series models.Series
		json.Unmarshal(rr.Body.Bytes(), &series)
		if series.Title != "Series 1" {
			t.Errorf("Expected series title 'Series 1', got '%s'", series.Title)
		}
		if len(series.Chapters) != 1 {
			t.Errorf("Expected 1 chapter, got %d", len(series.Chapters))
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/series/999", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})
}

func TestHandleGetPage(t *testing.T) {
	setupTestServer(t)
	router := testServer.Router()

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/series/1/chapters/1/pages/2", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		if contentType := rr.Header().Get("Content-Type"); contentType != "image/jpeg" {
			t.Errorf("handler returned wrong content type: got %v want %v", contentType, "image/jpeg")
		}
	})

	t.Run("Page Not Found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/series/1/chapters/1/pages/99", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})

	t.Run("Chapter Not Found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/series/1/chapters/99/pages/1", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})
}
