// It uses Go's httptest package
// to simulate HTTP requests without needing to run a live server.

package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/vrsandeep/mango-go/internal/assets"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestHandleListSeries(t *testing.T) {
	server, db := testutil.SetupTestServer(t)
	router := server.Router()
	testutil.PersistOneSeriesAndChapter(t, db)

	req, _ := http.NewRequest("GET", "/api/series", nil)
	req.AddCookie(testutil.GetAuthCookie(t, server, "testuser", "password", "user"))
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
	server, db := testutil.SetupTestServer(t)
	router := server.Router()
	testutil.PersistOneSeriesAndChapter(t, db)

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/series/1", nil)
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
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
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})
}

func TestHandleGetPage(t *testing.T) {
	server, db := testutil.SetupTestServer(t)
	router := server.Router()
	testutil.PersistOneSeriesAndChapter(t, db)

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/series/1/chapters/1/pages/2", nil)
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
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
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})

	t.Run("Chapter Not Found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/series/1/chapters/99/pages/1", nil)
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})
}

func TestServeReaderHTML(t *testing.T) {
	server := testutil.SetupTestServerEmbedded(t)
	router := server.Router()

	// --- Get the expected content directly from the embedded source ---
	// The path is now relative to the assets package's embed directive.
	expectedFile, err := assets.WebFS.Open("web/reader.html")
	if err != nil {
		t.Fatalf("Could not read embedded reader.html for test comparison: %v", err)
	}
	expectedBody, err := io.ReadAll(expectedFile)
	if err != nil {
		t.Fatalf("Could not read embedded reader.html content: %v", err)
	}

	// --- Perform the request ---
	req, _ := http.NewRequest("GET", "/reader/series/1/chapters/1", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// --- Assertions ---
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if contentType := rr.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("handler returned wrong content type: got %s want text/html", contentType)
	}

	if rr.Body.String() != string(expectedBody) {
		t.Error("handler returned body that does not match embedded web/reader.html content")
	}
}

// test to ensure the library.html serves series correctly.
func TestServeLibraryHTML(t *testing.T) {
	server, _ := testutil.SetupTestServer(t)
	router := server.Router()

	// The handler's http.ServeFile uses "./web/reader.html", which assumes
	// the app is run from the project root. We must replicate this condition.
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	// Change to project root (which is two levels up from internal/api)
	if err := os.Chdir("../../"); err != nil {
		t.Fatalf("Failed to change directory to project root: %v", err)
	}
	// Ensure we change back to the original directory after the test
	defer os.Chdir(originalWD)

	// Now that we are in the project root, we can read the actual file.
	expectedBody, err := os.ReadFile("./internal/assets/web/series.html")
	if err != nil {
		t.Fatalf("Could not read actual reader.html file: %v", err)
	}

	// Perform the request
	req, _ := http.NewRequest("GET", "/library", nil)
	req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assertions
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if contentType := rr.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, "text/html")
	}

	if rr.Body.String() != string(expectedBody) {
		t.Error("handler returned body that does not match cmd/mango-server/web/reader.html content")
	}
}
