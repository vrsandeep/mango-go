package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrsandeep/mango-go/internal/api"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func setupTestData(t *testing.T) (*api.Server, http.Handler, *http.Cookie, *models.Folder, *models.Folder, *models.Chapter) {
	t.Helper()
	server, _, _ := testutil.SetupTestServer(t)
	router := server.Router()
	cookie := testutil.GetAuthCookie(t, server, "user", "pw", "user")

	// Create folder structure:
	// /Folder A
	//   - /Subfolder B
	//     - chapter-b1.cbz
	//   - chapter-a1.cbz
	// /Folder C
	folderA, _ := server.Store().CreateFolder("/Folder A", "Folder A", nil)
	folderB, _ := server.Store().CreateFolder("/Folder A/Subfolder B", "Subfolder B", &folderA.ID)
	server.Store().CreateFolder("/Folder C", "Folder C", nil)
	chapterA1, _ := server.Store().CreateChapter(folderA.ID, "/Folder A/chapter-a1.cbz", "hashA1", 10, "")
	server.Store().CreateChapter(folderB.ID, "/Folder A/Subfolder B/chapter-b1.cbz", "hashB1", 10, "")
	// Assuming user id is 1 for testing
	server.Store().UpdateChapterProgress(chapterA1.ID, 1, 50, false)

	return server, router, cookie, folderA, folderB, chapterA1
}

func TestAddTagToFolder(t *testing.T) {
	server, router, cookie, folderA, _, _ := setupTestData(t)
	// 1. Add Tag
	tagPayload := `{"name": "shonen"}`
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/tags", folderA.ID), bytes.NewBufferString(tagPayload))
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	responseBody := rr.Body.String()
	t.Logf("Response body: %s", responseBody)

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
}

func TestBrowseRoot(t *testing.T) {
	_, router, cookie, _, _, _ := setupTestData(t)

	req, _ := http.NewRequest("GET", "/api/browse", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		return
	}

	var resp struct {
		CurrentFolder *models.Folder    `json:"current_folder"`
		Subfolders    []*models.Folder  `json:"subfolders"`
		Chapters      []*models.Chapter `json:"chapters"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if len(resp.Subfolders) != 2 {
		t.Fatalf("Expected to find 2 root folders, but got %d", len(resp.Subfolders))
	}
}

func TestBrowseFolderWithMixedContent(t *testing.T) {
	server, router, cookie, _, _, _ := setupTestData(t)
	folderA, err := server.Store().GetFolderByPath("/Folder A")
	if err != nil || folderA == nil {
		t.Fatalf("Failed to get Folder A by path: %v", err)
	}
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/browse?folderId=%d", folderA.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var resp struct {
		CurrentFolder *models.Folder    `json:"current_folder"`
		Subfolders    []*models.Folder  `json:"subfolders"`
		Chapters      []*models.Chapter `json:"chapters"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if len(resp.Subfolders) != 1 || resp.Subfolders[0].Name != "Subfolder B" {
		t.Errorf("Expected to find 'Subfolder B', but got %+v", resp.Subfolders)
	}
	if len(resp.Chapters) != 1 || resp.Chapters[0].Path != "/Folder A/chapter-a1.cbz" {
		t.Errorf("Expected to find 'chapter-a1.cbz', but got %+v", resp.Chapters)
	}
}

func TestBrowseFolderA(t *testing.T) {
	_, router, cookie, folderA, _, _ := setupTestData(t)
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/browse?folderId=%d", folderA.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var resp struct {
		CurrentFolder *models.Folder    `json:"current_folder"`
		Subfolders    []*models.Folder  `json:"subfolders"`
		Chapters      []*models.Chapter `json:"chapters"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if len(resp.Subfolders) != 1 || resp.Subfolders[0].Name != "Subfolder B" {
		t.Errorf("Expected to find 'Subfolder B', but got %+v", resp.Subfolders)
	}
	if len(resp.Chapters) != 1 || resp.Chapters[0].Path != "/Folder A/chapter-a1.cbz" {
		t.Errorf("Expected to find 'chapter-a1.cbz', but got %+v", resp.Chapters)
	}
	if resp.CurrentFolder.ID != folderA.ID {
		t.Errorf("Expected current folder to be %d, but got %d", folderA.ID, resp.CurrentFolder.ID)
	}
}

func TestGetBreadcrumb(t *testing.T) {
	_, router, cookie, _, folderB, _ := setupTestData(t)
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/browse/breadcrumb?folderId=%d", folderB.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var resp []*models.Folder
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if len(resp) != 2 || resp[0].Name != "Folder A" || resp[1].Name != "Subfolder B" {
		t.Errorf("Breadcrumb was incorrect, got %+v", resp)
	}
}

func TestGetFolderSettings(t *testing.T) {
	_, router, cookie, folderA, _, _ := setupTestData(t)

	// Test 1: Get default settings for a folder that has no saved settings
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/folders/%d/settings", folderA.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var settings models.FolderSettings
	json.Unmarshal(rr.Body.Bytes(), &settings)
	if settings.SortBy != "auto" {
		t.Errorf("Expected default sort_by 'auto', got '%s'", settings.SortBy)
	}
	if settings.SortDir != "asc" {
		t.Errorf("Expected default sort_dir 'asc', got '%s'", settings.SortDir)
	}

	// Test 2: Update settings and then retrieve them
	updatePayload := `{"sort_by": "name", "sort_dir": "desc"}`
	req, _ = http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/settings", folderA.ID), bytes.NewBufferString(updatePayload))
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("Update settings: expected status 200, got %d", status)
	}

	// Test 3: Retrieve the updated settings
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/folders/%d/settings", folderA.ID), nil)
	req.AddCookie(cookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("Get updated settings: expected status 200, got %d", status)
	}

	json.Unmarshal(rr.Body.Bytes(), &settings)
	if settings.SortBy != "name" {
		t.Errorf("Expected sort_by 'name', got '%s'", settings.SortBy)
	}
	if settings.SortDir != "desc" {
		t.Errorf("Expected sort_dir 'desc', got '%s'", settings.SortDir)
	}

	// Test 4: Test unauthorized access (no cookie)
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/folders/%d/settings", folderA.ID), nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Fatalf("Unauthorized access: expected status 401, got %d", status)
	}

	// Test 5: Test with non-existent folder ID (should return default settings)
	req, _ = http.NewRequest("GET", "/api/folders/99999/settings", nil)
	req.AddCookie(cookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("Non-existent folder: expected status 200, got %d", status)
	}

	// Should return default settings for non-existent folder
	json.Unmarshal(rr.Body.Bytes(), &settings)
	if settings.SortBy != "auto" {
		t.Errorf("Expected default sort_by 'auto' for non-existent folder, got '%s'", settings.SortBy)
	}
	if settings.SortDir != "asc" {
		t.Errorf("Expected default sort_dir 'asc' for non-existent folder, got '%s'", settings.SortDir)
	}
}

func TestUpdateFolderSettings(t *testing.T) {
	server, router, cookie, folderA, _, _ := setupTestData(t)

	// Test 1: Update settings with valid payload
	updatePayload := `{"sort_by": "created_at", "sort_dir": "asc"}`
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/settings", folderA.ID), bytes.NewBufferString(updatePayload))
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("Update settings: expected status 200, got %d", status)
	}

	// Verify settings were saved
	settings, err := server.Store().GetFolderSettings(folderA.ID, 1) // Assuming user ID is 1
	if err != nil {
		t.Fatalf("Failed to retrieve settings: %v", err)
	}
	if settings.SortBy != "created_at" {
		t.Errorf("Expected sort_by 'created_at', got '%s'", settings.SortBy)
	}
	if settings.SortDir != "asc" {
		t.Errorf("Expected sort_dir 'asc', got '%s'", settings.SortDir)
	}

	// Test 2: Update settings again (should update existing record)
	updatePayload2 := `{"sort_by": "progress", "sort_dir": "desc"}`
	req, _ = http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/settings", folderA.ID), bytes.NewBufferString(updatePayload2))
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("Update settings again: expected status 200, got %d", status)
	}

	// Verify settings were updated
	settings, err = server.Store().GetFolderSettings(folderA.ID, 1)
	if err != nil {
		t.Fatalf("Failed to retrieve updated settings: %v", err)
	}
	if settings.SortBy != "progress" {
		t.Errorf("Expected sort_by 'progress', got '%s'", settings.SortBy)
	}
	if settings.SortDir != "desc" {
		t.Errorf("Expected sort_dir 'desc', got '%s'", settings.SortDir)
	}

	// Test 3: Test unauthorized access (no cookie)
	req, _ = http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/settings", folderA.ID), bytes.NewBufferString(updatePayload))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Fatalf("Unauthorized access: expected status 401, got %d", status)
	}

	// Test 4: Test with invalid JSON payload
	req, _ = http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/settings", folderA.ID), bytes.NewBufferString(`{"invalid": json`))
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Fatalf("Invalid JSON: expected status 400, got %d", status)
	}

	// Test 5: Test with non-existent folder ID
	req, _ = http.NewRequest("POST", "/api/folders/99999/settings", bytes.NewBufferString(updatePayload))
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Fatalf("Non-existent folder: expected status 500, got %d", status)
	}
}
