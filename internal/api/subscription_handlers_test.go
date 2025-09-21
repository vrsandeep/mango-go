package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestHandleSubscribeToSeries(t *testing.T) {
	server, db := SetupTestServerWithProviders(t)
	router := server.Router()

	t.Run("Success", func(t *testing.T) {
		payload := map[string]string{
			"series_title":      "Subscribe Test",
			"series_identifier": "sub-test-1",
			"provider_id":       "mockadex",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/subscriptions", bytes.NewBuffer(body))
		req.AddCookie(testutil.CookieForUser(t, server, "testuser-update", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
		}

		var count int
		db.QueryRow("SELECT COUNT(*) FROM subscriptions WHERE series_identifier = 'sub-test-1'").Scan(&count)
		if count != 1 {
			t.Error("Expected subscription to be created, but it was not found in DB")
		}
	})

	t.Run("Success with folder path", func(t *testing.T) {
		payload := map[string]interface{}{
			"series_title":      "Subscribe Test with Folder",
			"series_identifier": "sub-test-2",
			"provider_id":       "mockadex",
			"folder_path":       "custom/path",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/subscriptions", bytes.NewBuffer(body))
		req.AddCookie(testutil.CookieForUser(t, server, "testuser-update", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
		}

		var folderPath string
		var count int
		db.QueryRow("SELECT folder_path FROM subscriptions WHERE series_identifier = 'sub-test-2'").Scan(&folderPath)
		db.QueryRow("SELECT COUNT(*) FROM subscriptions WHERE series_identifier = 'sub-test-2'").Scan(&count)

		if count != 1 {
			t.Error("Expected subscription to be created, but it was not found in DB")
		}
		if folderPath != "custom/path" {
			t.Errorf("Expected folder_path to be 'custom/path', got '%s'", folderPath)
		}
	})

	t.Run("Success with null folder path", func(t *testing.T) {
		payload := map[string]interface{}{
			"series_title":      "Subscribe Test with Null Folder",
			"series_identifier": "sub-test-3",
			"provider_id":       "mockadex",
			"folder_path":       nil,
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/subscriptions", bytes.NewBuffer(body))
		req.AddCookie(testutil.CookieForUser(t, server, "testuser-update", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
		}

		var folderPath *string
		var count int
		db.QueryRow("SELECT folder_path FROM subscriptions WHERE series_identifier = 'sub-test-3'").Scan(&folderPath)
		db.QueryRow("SELECT COUNT(*) FROM subscriptions WHERE series_identifier = 'sub-test-3'").Scan(&count)

		if count != 1 {
			t.Error("Expected subscription to be created, but it was not found in DB")
		}
		if folderPath != nil {
			t.Errorf("Expected folder_path to be null, got '%s'", *folderPath)
		}
	})
}

func TestHandleUpdateSubscriptionFolderPath(t *testing.T) {
	server, db := SetupTestServerWithProviders(t)
	router := server.Router()

	// Create user once for all tests
	userCookie := testutil.CookieForUser(t, server, "testuser-update", "password", "user")

	// First create a subscription
	payload := map[string]string{
		"series_title":      "Update Folder Test",
		"series_identifier": "update-test-1",
		"provider_id":       "mockadex",
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/subscriptions", bytes.NewBuffer(body))
	req.AddCookie(userCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Failed to create subscription: %v", rr.Code)
	}

	var subID int64
	db.QueryRow("SELECT id FROM subscriptions WHERE series_identifier = 'update-test-1'").Scan(&subID)

	t.Run("Update folder path", func(t *testing.T) {
		updatePayload := map[string]string{
			"folder_path": "updated/custom/path",
		}
		body, _ := json.Marshal(updatePayload)
		req, _ := http.NewRequest("PUT", "/api/subscriptions/1/folder-path", bytes.NewBuffer(body))
		req.AddCookie(userCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var folderPath string
		db.QueryRow("SELECT folder_path FROM subscriptions WHERE id = ?", subID).Scan(&folderPath)
		if folderPath != "updated/custom/path" {
			t.Errorf("Expected folder_path to be 'updated/custom/path', got '%s'", folderPath)
		}
	})

	t.Run("Update folder path to null", func(t *testing.T) {
		updatePayload := map[string]interface{}{
			"folder_path": nil,
		}
		body, _ := json.Marshal(updatePayload)
		req, _ := http.NewRequest("PUT", "/api/subscriptions/1/folder-path", bytes.NewBuffer(body))
		req.AddCookie(userCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var folderPath *string
		db.QueryRow("SELECT folder_path FROM subscriptions WHERE id = ?", subID).Scan(&folderPath)
		if folderPath != nil {
			t.Errorf("Expected folder_path to be null, got '%s'", *folderPath)
		}
	})

	t.Run("Update non-existent subscription", func(t *testing.T) {
		updatePayload := map[string]string{
			"folder_path": "should/fail",
		}
		body, _ := json.Marshal(updatePayload)
		req, _ := http.NewRequest("PUT", "/api/subscriptions/99999/folder-path", bytes.NewBuffer(body))
		req.AddCookie(userCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})
}

func TestHandleSubscribeToSeriesWithPathValidation(t *testing.T) {
	server, db := SetupTestServerWithProviders(t)
	router := server.Router()

	t.Run("Invalid folder path - directory traversal", func(t *testing.T) {
		payload := map[string]interface{}{
			"series_title":      "Test Series",
			"series_identifier": "test-invalid-path",
			"provider_id":       "mockadex",
			"folder_path":       "../../etc/passwd",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/subscriptions", bytes.NewBuffer(body))
		req.AddCookie(testutil.CookieForUser(t, server, "testuser-path", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}

		var response map[string]string
		json.Unmarshal(rr.Body.Bytes(), &response)
		if !strings.Contains(response["error"], "Invalid folder path") {
			t.Errorf("Expected error about invalid folder path, got: %s", response["error"])
		}
	})

	t.Run("Valid folder path", func(t *testing.T) {
		payload := map[string]interface{}{
			"series_title":      "Test Series Valid",
			"series_identifier": "test-valid-path",
			"provider_id":       "mockadex",
			"folder_path":       "valid/custom/path",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/subscriptions", bytes.NewBuffer(body))
		req.AddCookie(testutil.CookieForUser(t, server, "testuser-valid", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
		}

		// Verify the subscription was created with the sanitized path
		var folderPath string
		db.QueryRow("SELECT folder_path FROM subscriptions WHERE series_identifier = 'test-valid-path'").Scan(&folderPath)
		if folderPath != "valid/custom/path" {
			t.Errorf("Expected folder path 'valid/custom/path', got '%s'", folderPath)
		}
	})

	t.Run("Empty folder path should be allowed", func(t *testing.T) {
		payload := map[string]interface{}{
			"series_title":      "Test Series Empty",
			"series_identifier": "test-empty-path",
			"provider_id":       "mockadex",
			"folder_path":       "",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/subscriptions", bytes.NewBuffer(body))
		req.AddCookie(testutil.CookieForUser(t, server, "testuser-empty", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
		}
	})
}

func TestHandleUpdateSubscriptionFolderPathWithValidation(t *testing.T) {
	server, db := SetupTestServerWithProviders(t)
	router := server.Router()

	// Create user and subscription
	userCookie := testutil.CookieForUser(t, server, "testuser-update-validation", "password", "user")

	payload := map[string]string{
		"series_title":      "Update Validation Test",
		"series_identifier": "update-validation-test",
		"provider_id":       "mockadex",
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/subscriptions", bytes.NewBuffer(body))
	req.AddCookie(userCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Failed to create subscription: %v", rr.Code)
	}

	var subID int64
	db.QueryRow("SELECT id FROM subscriptions WHERE series_identifier = 'update-validation-test'").Scan(&subID)

	t.Run("Invalid folder path update", func(t *testing.T) {
		updatePayload := map[string]string{
			"folder_path": "../../invalid",
		}
		body, _ := json.Marshal(updatePayload)
		req, _ := http.NewRequest("PUT", "/api/subscriptions/1/folder-path", bytes.NewBuffer(body))
		req.AddCookie(userCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}

		var response map[string]string
		json.Unmarshal(rr.Body.Bytes(), &response)
		if !strings.Contains(response["error"], "Invalid folder path") {
			t.Errorf("Expected error about invalid folder path, got: %s", response["error"])
		}
	})

	t.Run("Valid folder path update", func(t *testing.T) {
		updatePayload := map[string]string{
			"folder_path": "updated/valid/path",
		}
		body, _ := json.Marshal(updatePayload)
		req, _ := http.NewRequest("PUT", "/api/subscriptions/1/folder-path", bytes.NewBuffer(body))
		req.AddCookie(userCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Verify the update
		var folderPath string
		db.QueryRow("SELECT folder_path FROM subscriptions WHERE id = ?", subID).Scan(&folderPath)
		if folderPath != "updated/valid/path" {
			t.Errorf("Expected folder path 'updated/valid/path', got '%s'", folderPath)
		}
	})
}
