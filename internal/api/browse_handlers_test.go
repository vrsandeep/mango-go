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

func TestHandleListAllFolders(t *testing.T) {
	_, router, cookie, _, _, _ := setupTestData(t)

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/folders", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var folders []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
			Path string `json:"path"`
		}
		json.Unmarshal(rr.Body.Bytes(), &folders)

		// Should have at least the folders we created in setupTestData
		if len(folders) < 2 {
			t.Fatalf("Expected at least 2 folders, got %d", len(folders))
		}

		// Check that we have the expected folders
		folderNames := make(map[string]bool)
		for _, folder := range folders {
			folderNames[folder.Name] = true
		}

		if !folderNames["Folder A"] {
			t.Error("Expected to find 'Folder A' in folders list")
		}
		if !folderNames["Folder C"] {
			t.Error("Expected to find 'Folder C' in folders list")
		}

		// Check that each folder has the required fields
		for _, folder := range folders {
			if folder.ID == 0 {
				t.Error("Folder ID should not be zero")
			}
			if folder.Name == "" {
				t.Error("Folder name should not be empty")
			}
			if folder.Path == "" {
				t.Error("Folder path should not be empty")
			}
		}
	})

	t.Run("Unauthorized Access", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/folders", nil)
		// No cookie
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Fatalf("Expected status 401 for unauthorized access, got %d", status)
		}
	})
}

func TestUpdateFolderRating(t *testing.T) {
	server, router, cookie, folderA, _, _ := setupTestData(t)

	t.Run("Set rating 4", func(t *testing.T) {
		body := `{"rating": 4}`
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/folders/%d/rating", folderA.ID), bytes.NewBufferString(body))
		req.AddCookie(cookie)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var resp struct {
			Rating *int `json:"rating"`
		}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp.Rating == nil || *resp.Rating != 4 {
			t.Errorf("expected rating 4, got %v", resp.Rating)
		}

		f, _ := server.Store().GetFolder(folderA.ID)
		if f.Rating == nil || *f.Rating != 4 {
			t.Errorf("expected folder rating 4 in DB, got %v", f.Rating)
		}
	})

	t.Run("Update rating to 3", func(t *testing.T) {
		body := `{"rating": 3}`
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/folders/%d/rating", folderA.ID), bytes.NewBufferString(body))
		req.AddCookie(cookie)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		f, _ := server.Store().GetFolder(folderA.ID)
		if f.Rating == nil || *f.Rating != 3 {
			t.Errorf("expected rating 3, got %v", f.Rating)
		}
	})

	t.Run("Clear rating with 0", func(t *testing.T) {
		body := `{"rating": 0}`
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/folders/%d/rating", folderA.ID), bytes.NewBufferString(body))
		req.AddCookie(cookie)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		f, _ := server.Store().GetFolder(folderA.ID)
		if f.Rating != nil {
			t.Errorf("expected nil rating after clearing, got %v", *f.Rating)
		}
	})

	t.Run("Rating returned in browse response", func(t *testing.T) {
		body := `{"rating": 5}`
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/folders/%d/rating", folderA.ID), bytes.NewBufferString(body))
		req.AddCookie(cookie)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("set rating: expected 200, got %d", rr.Code)
		}

		req, _ = http.NewRequest("GET", "/api/browse", nil)
		req.AddCookie(cookie)
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		var resp struct {
			Subfolders []*models.Folder `json:"subfolders"`
		}
		json.Unmarshal(rr.Body.Bytes(), &resp)

		var found bool
		for _, f := range resp.Subfolders {
			if f.ID == folderA.ID {
				found = true
				if f.Rating == nil || *f.Rating != 5 {
					t.Errorf("expected rating 5 in browse response, got %v", f.Rating)
				}
			}
		}
		if !found {
			t.Errorf("Folder A not found in browse response")
		}
	})

	t.Run("Non-existent folder returns 404", func(t *testing.T) {
		body := `{"rating": 5}`
		req, _ := http.NewRequest("PUT", "/api/folders/99999/rating", bytes.NewBufferString(body))
		req.AddCookie(cookie)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rr.Code)
		}
	})

	t.Run("Invalid JSON returns 400", func(t *testing.T) {
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/folders/%d/rating", folderA.ID), bytes.NewBufferString(`{bad json`))
		req.AddCookie(cookie)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("Unauthorized returns 401", func(t *testing.T) {
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/folders/%d/rating", folderA.ID), bytes.NewBufferString(`{"rating": 5}`))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})
}

func setupUnreadTestData(t *testing.T) (http.Handler, *http.Cookie, int64, int64, int64) {
	t.Helper()
	server, _, _ := testutil.SetupTestServer(t)
	router := server.Router()
	cookie := testutil.GetAuthCookie(t, server, "filtuser", "pw", "user")

	folderX, _ := server.Store().CreateFolder("/Folder X", "Folder X", nil)
	chX1, _ := server.Store().CreateChapter(folderX.ID, "/Folder X/ch1.cbz", "hx1", 5, "")
	server.Store().CreateChapter(folderX.ID, "/Folder X/ch2.cbz", "hx2", 5, "")
	server.Store().UpdateChapterProgress(chX1.ID, 1, 100, true)

	folderY, _ := server.Store().CreateFolder("/Folder Y", "Folder Y", nil)
	chY1, _ := server.Store().CreateChapter(folderY.ID, "/Folder Y/ch1.cbz", "hy1", 5, "")
	server.Store().UpdateChapterProgress(chY1.ID, 1, 100, true)

	folderZ, _ := server.Store().CreateFolder("/Folder Z", "Folder Z", nil)

	return router, cookie, folderX.ID, folderY.ID, folderZ.ID
}

func TestBrowseUnreadFilter(t *testing.T) {
	router, cookie, folderXID, folderYID, folderZID := setupUnreadTestData(t)

	t.Run("Without filter returns all folders", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/browse", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		var resp struct {
			Subfolders []*models.Folder `json:"subfolders"`
		}
		json.Unmarshal(rr.Body.Bytes(), &resp)

		ids := make(map[int64]bool)
		for _, f := range resp.Subfolders {
			ids[f.ID] = true
		}

		if !ids[folderXID] || !ids[folderYID] || !ids[folderZID] {
			t.Errorf("expected all 3 folders without filter, got IDs: %v", ids)
		}
	})

	t.Run("With unread_only=true returns only folders with unread chapters", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/browse?unread_only=true", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		var resp struct {
			Subfolders []*models.Folder `json:"subfolders"`
		}
		json.Unmarshal(rr.Body.Bytes(), &resp)

		ids := make(map[int64]bool)
		for _, f := range resp.Subfolders {
			ids[f.ID] = true
		}

		if !ids[folderXID] {
			t.Errorf("Folder X (partial read) should appear, got IDs: %v", ids)
		}
		if ids[folderYID] {
			t.Errorf("Folder Y (fully read) should not appear, got IDs: %v", ids)
		}
		if ids[folderZID] {
			t.Errorf("Folder Z (no chapters) should not appear, got IDs: %v", ids)
		}
		if len(resp.Subfolders) != 1 {
			t.Errorf("expected 1 folder with unread filter, got %d", len(resp.Subfolders))
		}
	})
}

func TestBrowseUnreadFilterInsideFolder(t *testing.T) {
	server, _, _ := testutil.SetupTestServer(t)
	router := server.Router()
	cookie := testutil.GetAuthCookie(t, server, "chfiltuser", "pw", "user")

	folder, _ := server.Store().CreateFolder("/FX", "FX", nil)
	chRead, _ := server.Store().CreateChapter(folder.ID, "/FX/ch1.cbz", "hfxa", 5, "")
	chUnread, _ := server.Store().CreateChapter(folder.ID, "/FX/ch2.cbz", "hfxb", 5, "")
	server.Store().UpdateChapterProgress(chRead.ID, 1, 100, true)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/browse?folderId=%d&unread_only=true", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Chapters []*models.Chapter `json:"chapters"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if len(resp.Chapters) != 1 || resp.Chapters[0].ID != chUnread.ID {
		t.Errorf("expected only unread chapter (id=%d), got %+v", chUnread.ID, resp.Chapters)
	}
}

func TestBrowseWithTagFilter(t *testing.T) {
	_, router, cookie, folderA, _, _ := setupTestData(t)

	tagPayload := `{"name": "action"}`
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/tags", folderA.ID), bytes.NewBufferString(tagPayload))
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("add tag: expected 201, got %d", rr.Code)
	}

	var tag models.Tag
	json.Unmarshal(rr.Body.Bytes(), &tag)

	t.Run("Filter by tag returns only tagged folders", func(t *testing.T) {
		req, _ = http.NewRequest("GET", fmt.Sprintf("/api/browse?tagId=%d", tag.ID), nil)
		req.AddCookie(cookie)
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		var resp struct {
			Subfolders []*models.Folder `json:"subfolders"`
		}
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if len(resp.Subfolders) != 1 || resp.Subfolders[0].ID != folderA.ID {
			t.Errorf("expected only Folder A with tag filter, got %+v", resp.Subfolders)
		}
	})

	t.Run("Non-existent tag ID returns empty result", func(t *testing.T) {
		req, _ = http.NewRequest("GET", "/api/browse?tagId=99999", nil)
		req.AddCookie(cookie)
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		var resp struct {
			Subfolders []*models.Folder `json:"subfolders"`
		}
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if len(resp.Subfolders) != 0 {
			t.Errorf("expected empty result for non-existent tag, got %d folders", len(resp.Subfolders))
		}
	})

	t.Run("Invalid tag ID returns 400", func(t *testing.T) {
		req, _ = http.NewRequest("GET", "/api/browse?tagId=notanumber", nil)
		req.AddCookie(cookie)
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for invalid tag ID, got %d", rr.Code)
		}
	})
}

func TestHandleSearchFolders(t *testing.T) {
	_, router, cookie, folderA, _, _ := setupTestData(t)

	t.Run("Search with matching query", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/folders/search?q=Folder", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var results []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
			Path string `json:"path"`
		}
		json.Unmarshal(rr.Body.Bytes(), &results)

		if len(results) == 0 {
			t.Fatal("Expected at least one folder matching 'Folder', got 0")
		}

		// Verify all results contain "Folder" in the name
		for _, result := range results {
			if result.ID == 0 {
				t.Error("Folder ID should not be zero")
			}
			if result.Name == "" {
				t.Error("Folder name should not be empty")
			}
		}
	})

	t.Run("Search with exact match", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/folders/search?q=%s", "Folder A"), nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var results []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
			Path string `json:"path"`
		}
		json.Unmarshal(rr.Body.Bytes(), &results)

		// Should find Folder A
		found := false
		for _, result := range results {
			if result.ID == folderA.ID && result.Name == "Folder A" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find 'Folder A' in search results")
		}
	})

	t.Run("Search with empty query", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/folders/search?q=", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var results []map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &results)

		if len(results) != 0 {
			t.Errorf("Expected empty results for empty query, got %d results", len(results))
		}
	})

	t.Run("Search with no query parameter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/folders/search", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var results []map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &results)

		if len(results) != 0 {
			t.Errorf("Expected empty results for no query parameter, got %d results", len(results))
		}
	})

	t.Run("Search with no matches", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/folders/search?q=NonExistentFolder123", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var results []map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &results)

		if len(results) != 0 {
			t.Errorf("Expected 0 results for non-existent folder, got %d", len(results))
		}
	})

	t.Run("Search results have correct structure", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/folders/search?q=Folder", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var results []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
			Path string `json:"path"`
		}
		json.Unmarshal(rr.Body.Bytes(), &results)

		if len(results) > 0 {
			result := results[0]
			if result.ID == 0 {
				t.Error("Result ID should not be zero")
			}
			if result.Name == "" {
				t.Error("Result name should not be empty")
			}
			// Path can be empty if it's relative to library root
		}
	})

	t.Run("Unauthorized Access", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/folders/search?q=Folder", nil)
		// No cookie
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Fatalf("Expected status 401 for unauthorized access, got %d", status)
		}
	})

	t.Run("Search with URL encoded query", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/folders/search?q=Folder%20A", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var results []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
			Path string `json:"path"`
		}
		json.Unmarshal(rr.Body.Bytes(), &results)

		// Should find Folder A
		found := false
		for _, result := range results {
			if result.ID == folderA.ID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find 'Folder A' in search results with URL encoded query")
		}
	})
}
