package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestBrowseHandlers(t *testing.T) {
	server, db := testutil.SetupTestServer(t)
	router := server.Router()
	adminCookie := testutil.GetAuthCookie(t, server, "admin", "pw", "admin")

	// Setup: Create a folder hierarchy in the test DB
	var folderA, folderB models.Folder
	db.QueryRow("INSERT INTO folders (path, name) VALUES ('/A', 'Folder A') RETURNING id, path, name").Scan(&folderA.ID, &folderA.Path, &folderA.Name)
	db.QueryRow("INSERT INTO folders (path, name, parent_id) VALUES ('/A/B', 'Folder B', ?) RETURNING id, path, name", folderA.ID).Scan(&folderB.ID, &folderB.Path, &folderB.Name)
	db.Exec("INSERT INTO chapters (folder_id, path, content_hash) VALUES (?, '/A/B/ch1.cbz', 'hash1')", folderB.ID)

	t.Run("Browse Root", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/browse", nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var resp struct {
			Subfolders []*models.Folder `json:"subfolders"`
		}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if len(resp.Subfolders) != 1 || resp.Subfolders[0].Name != "Folder A" {
			t.Errorf("Expected to browse root and find 'Folder A', but got %+v", resp.Subfolders)
		}
	})

	t.Run("Browse Folder A", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/browse?folderId=%d", folderA.ID), nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var resp struct {
			Subfolders []*models.Folder `json:"subfolders"`
		}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if len(resp.Subfolders) != 1 || resp.Subfolders[0].Name != "Folder B" {
			t.Errorf("Expected to browse Folder A and find 'Folder B', but got %+v", resp.Subfolders)
		}
	})

	t.Run("Get Breadcrumb", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/browse/breadcrumb?folderId=%d", folderB.ID), nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var resp []*models.Folder
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if len(resp) != 2 || resp[0].Name != "Folder A" || resp[1].Name != "Folder B" {
			t.Errorf("Breadcrumb was incorrect, got %+v", resp)
		}
	})
}
