package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestBrowseHandlers(t *testing.T) {
	server, _ := testutil.SetupTestServer(t)
	router := server.Router()
	cookie := testutil.GetAuthCookie(t, server, "user", "pw", "user")

	// Setup: Create a folder hierarchy in the test DB
	// var folderA, folderB models.Folder
	// db.QueryRow("INSERT INTO folders (path, name) VALUES ('/A', 'Folder A') RETURNING id, path, name").Scan(&folderA.ID, &folderA.Path, &folderA.Name)
	// db.QueryRow("INSERT INTO folders (path, name, parent_id) VALUES ('/A/B', 'Folder B', ?) RETURNING id, path, name", folderA.ID).Scan(&folderB.ID, &folderB.Path, &folderB.Name)
	// db.Exec("INSERT INTO chapters (folder_id, path, content_hash) VALUES (?, '/A/B/ch1.cbz', 'hash1')", folderB.ID)

	// Create folder structure:
	// /Folder A
	//   - /Subfolder B
	//     - chapter-b1.cbz
	//   - chapter-a1.cbz
	// /Folder C
	folderA, _ := server.Store().CreateFolder("/Folder A", "Folder A", nil)
	folderB, _ := server.Store().CreateFolder("/Folder A/Subfolder B", "Subfolder B", &folderA.ID)
	server.Store().CreateFolder("/Folder C", "Folder C", nil)
	server.Store().CreateChapter(folderA.ID, "/Folder A/chapter-a1.cbz", "hashA1", 10, "")
	server.Store().CreateChapter(folderB.ID, "/Folder A/Subfolder B/chapter-b1.cbz", "hashB1", 10, "")
	t.Run("Add and Remove Tag from Folder", func(t *testing.T) {
		// 1. Add Tag
		tagPayload := `{"name": "shonen"}`
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/tags", folderA.ID), bytes.NewBufferString(tagPayload))
		req.AddCookie(cookie)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Fatalf("Add tag: expected status 201, got %d", status)
		}
		var tag models.Tag
		json.Unmarshal(rr.Body.Bytes(), &tag)
		if tag.Name != "shonen" {
			t.Errorf("Expected tag 'shonen', got '%s'", tag.Name)
		}

		// Verify tag is associated
		f, _ := server.Store().GetFolder(folderA.ID)
		if len(f.Tags) != 1 || f.Tags[0].Name != "shonen" {
			t.Fatal("Tag was not correctly associated with folder in DB")
		}

		// 2. Remove Tag
		req, _ = http.NewRequest("DELETE", fmt.Sprintf("/api/folders/%d/tags/%d", folderA.ID, tag.ID), nil)
		req.AddCookie(cookie)
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNoContent {
			t.Fatalf("Remove tag: expected status 204, got %d", status)
		}

		// Verify tag is removed
		f, _ = server.Store().GetFolder(folderA.ID)
		if len(f.Tags) != 0 {
			t.Fatal("Tag was not correctly removed from folder in DB")
		}
	})
	t.Run("Browse Root", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/browse", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var resp struct {
			Subfolders []*models.Folder `json:"subfolders"`
		}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		// if len(resp.Subfolders) != 1 || resp.Subfolders[0].Name != "Folder A" {
		// 	t.Errorf("Expected to browse root and find 'Folder A', but got %+v", resp.Subfolders)
		// }
		if len(resp.Subfolders) != 2 {
			t.Fatalf("Expected to find 2 root folders, but got %d", len(resp.Subfolders))
		}
	})
	t.Run("Browse Folder with Mixed Content", func(t *testing.T) {
		folderA, _ := server.Store().GetFolderByPath("/Folder A")
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/browse?folderId=%d", folderA.ID), nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var resp struct {
			Subfolders []*models.Folder  `json:"subfolders"`
			Chapters   []*models.Chapter `json:"chapters"`
		}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if len(resp.Subfolders) != 1 || resp.Subfolders[0].Name != "Subfolder B" {
			t.Errorf("Expected to find 'Subfolder B', but got %+v", resp.Subfolders)
		}
		if len(resp.Chapters) != 1 || resp.Chapters[0].Path != "/Folder A/chapter-a1.cbz" {
			t.Errorf("Expected to find 'chapter-a1.cbz', but got %+v", resp.Chapters)
		}
	})
	t.Run("Browse Folder A", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/browse?folderId=%d", folderA.ID), nil)
		req.AddCookie(cookie)
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
		req.AddCookie(cookie)
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
