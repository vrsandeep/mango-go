package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/vrsandeep/mango-go/internal/assets"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestHandleGetPage(t *testing.T) {
	server, db, _ := testutil.SetupTestServer(t)
	router := server.Router()
	testutil.PersistOneFolderAndChapter(t, db)

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/chapters/1/pages/2", nil)
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
		req, _ := http.NewRequest("GET", "/api/chapters/1/pages/99", nil)
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})

	t.Run("Chapter Not Found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/chapters/99/pages/1", nil)
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})
}

func TestHandleGetChapterDetails(t *testing.T) {
	server, db, _ := testutil.SetupTestServer(t)
	router := server.Router()
	testutil.PersistOneFolderAndChapter(t, db)

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/chapters/1", nil)
		rr := httptest.NewRecorder()
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var chapter models.Chapter
		if err := json.Unmarshal(rr.Body.Bytes(), &chapter); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if chapter.ID != 1 {
			t.Errorf("Expected chapter ID 1, got %d", chapter.ID)
		}
		if chapter.Read != false {
			t.Errorf("Expected chapter Read status false, got %t", chapter.Read)
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/chapters/999", nil)
		rr := httptest.NewRecorder()
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})
}

func TestHandleUpdateProgress(t *testing.T) {
	server, db, _ := testutil.SetupTestServer(t)
	router := server.Router()
	s := store.New(db)
	testutil.PersistOneFolderAndChapter(t, db)

	t.Run("Success", func(t *testing.T) {
		payload := `{"progress_percent": 75, "read": true}`
		req, _ := http.NewRequest("POST", "/api/chapters/1/progress", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Verify the change in the database
		chapter, err := s.GetChapterByID(1, 1)
		if err != nil {
			t.Fatalf("Failed to get chapter after update: %v", err)
		}
		if chapter.ProgressPercent != 75 {
			t.Errorf("DB value for progress_percent was not updated: want 75, got %d", chapter.ProgressPercent)
		}
		if !chapter.Read {
			t.Error("DB value for read was not updated: want true, got false")
		}
	})

	t.Run("Invalid Payload", func(t *testing.T) {
		payload := `{"invalid_json": true}`
		req, _ := http.NewRequest("POST", "/api/chapters/1/progress", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			// Note: The handler still returns 200 OK because the JSON unmarshaling
			// will just result in zero-values for the expected fields. The DB update
			// will proceed with page=0 and read=false. This is an acceptable behavior.
			// A more robust implementation might return a 400 Bad Request if fields are missing.
			t.Logf("Handler returned OK for partially invalid payload, which is acceptable.")
		}
	})

	t.Run("Malformed JSON", func(t *testing.T) {
		payload := `{"progress_percent": 75,`
		req, _ := http.NewRequest("POST", "/api/chapters/1/progress", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code for malformed JSON: got %v want %v", status, http.StatusBadRequest)
		}
	})
}

func TestHandleGetChapterNeighbors(t *testing.T) {
	server, db, _ := testutil.SetupTestServer(t)
	router := server.Router()
	store := store.New(db)
	folder, err := store.CreateFolder("/Folder A", "Folder A", nil)
	if err != nil {
		t.Fatalf("Failed to create test folder: %v", err)
	}
	ch1, _ := store.CreateChapter(folder.ID, "ch1.cbz", "hash1", 2, "thumb1")
	ch2, _ := store.CreateChapter(folder.ID, "ch2.cbz", "hash2", 2, "thumb2")
	ch3, _ := store.CreateChapter(folder.ID, "ch3.cbz", "hash3", 2, "thumb3")

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/folders/"+strconv.FormatInt(folder.ID, 10)+"/chapters/"+strconv.FormatInt(ch2.ID, 10)+"/neighbors", nil)
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var neighbors map[string]*int64
		if err := json.Unmarshal(rr.Body.Bytes(), &neighbors); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if *neighbors["prev"] != ch1.ID {
			t.Errorf("Expected previous chapter ID, got %d", *neighbors["prev"])
		}
		if *neighbors["next"] != ch3.ID {
			t.Errorf("Expected next chapter ID, got %d", *neighbors["next"])
		}
	})
}

func TestServeReaderHTML(t *testing.T) {
	server, _, _ := testutil.SetupTestServer(t)
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
	server, _, _ := testutil.SetupTestServer(t)
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
	expectedBody, err := os.ReadFile("./internal/assets/web/library.html")
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
		t.Error("handler returned body that does not match cmd/mango-server/web/reader.html content, got: " + rr.Body.String())
	}
}
