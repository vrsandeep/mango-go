package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestTagHandlers(t *testing.T) {
	server, _, _ := testutil.SetupTestServer(t)
	router := server.Router()

	// Add a dummy series and tag for testing
	folderA, err := server.Store().CreateFolder("/test", "Test", nil)
	if err != nil {
		t.Fatalf("Failed to create folder: %v", err)
	}
	_, err = server.Store().AddTagToFolder(folderA.ID, "action")
	if err != nil {
		t.Fatalf("Failed to add tag to folder: %v", err)
	}

	t.Run("List Tags", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/tags", nil)
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("ListTags handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var tags []*models.Tag
		json.Unmarshal(rr.Body.Bytes(), &tags)
		if len(tags) != 1 {
			t.Fatalf("Expected 1 tag, got %d", len(tags))
		}
		if tags[0].Name != "action" {
			t.Errorf("Expected tag name 'action', got %s", tags[0].Name)
		}
		if tags[0].FolderCount != 1 {
			t.Errorf("Expected series count 1, got %d", tags[0].FolderCount)
		}
	})

	t.Run("List Folders By Tag", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/tags/1/folders", nil)
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("ListFoldersByTag handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var folders []*models.Folder
		json.Unmarshal(rr.Body.Bytes(), &folders)
		// Note: This relies on the placeholder logic in the handler. A full test
		// would require the dynamic SQL to be implemented.
		if len(folders) < 1 {
			t.Fatalf("Expected at least 1 folder for the tag, got %d", len(folders))
		}
	})
}
