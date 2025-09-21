package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
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
		req.AddCookie(testutil.CookieForUser(t, server, "testuser-sub", "password", "user"))
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
		req.AddCookie(testutil.CookieForUser(t, server, "testuser-sub2", "password", "user"))
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
		req.AddCookie(testutil.CookieForUser(t, server, "testuser-sub3", "password", "user"))
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

	t.Run("Invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/subscriptions", bytes.NewBufferString("invalid json"))
		req.AddCookie(testutil.CookieForUser(t, server, "testuser-invalid", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("Unauthorized", func(t *testing.T) {
		payload := map[string]string{
			"series_title":      "Unauthorized Test",
			"series_identifier": "unauth-test",
			"provider_id":       "mockadex",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/subscriptions", bytes.NewBuffer(body))
		// No cookie
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
	})
}

func TestHandleListSubscriptions(t *testing.T) {
	server, _ := SetupTestServerWithProviders(t)
	router := server.Router()

	// Create a subscription first
	payload := map[string]string{
		"series_title":      "List Test",
		"series_identifier": "list-test-1",
		"provider_id":       "mockadex",
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/subscriptions", bytes.NewBuffer(body))
	req.AddCookie(testutil.CookieForUser(t, server, "testuser-list", "password", "user"))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Failed to create subscription: %v", rr.Code)
	}

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/subscriptions", nil)
		req.AddCookie(testutil.CookieForUser(t, server, "testuser-list2", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var subs []map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &subs)
		if len(subs) == 0 {
			t.Error("Expected at least one subscription")
		}
	})

	t.Run("With provider filter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/subscriptions?provider_id=mockadex", nil)
		req.AddCookie(testutil.CookieForUser(t, server, "testuser-list3", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/subscriptions", nil)
		// No cookie
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
	})
}

func TestHandleDeleteSubscription(t *testing.T) {
	server, _ := SetupTestServerWithProviders(t)
	router := server.Router()

	t.Run("Success", func(t *testing.T) {
		// Create user and subscription
		userCookie := testutil.CookieForUser(t, server, "testuser-delete", "password", "user")

		payload := map[string]string{
			"series_title":      "Delete Test",
			"series_identifier": "delete-test-1",
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

		// Get the subscription ID from the response
		var response map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &response)
		subIDFloat, ok := response["id"].(float64)
		if !ok {
			t.Fatalf("Failed to get subscription ID from response")
		}
		subID := int64(subIDFloat)

		req, _ = http.NewRequest("DELETE", fmt.Sprintf("/api/subscriptions/%d", subID), nil)
		req.AddCookie(userCookie)
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNoContent {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
		}

		// Verify deletion by checking that the subscription is no longer in the list
		req, _ = http.NewRequest("GET", "/api/subscriptions", nil)
		req.AddCookie(userCookie)
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Failed to get subscriptions list: %v", status)
		}

		var subscriptions []map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &subscriptions)

		// Check that our deleted subscription is not in the list
		for _, sub := range subscriptions {
			if sub["id"].(float64) == float64(subID) {
				t.Errorf("Expected subscription %d to be deleted, but it's still in the list", subID)
			}
		}
	})

	t.Run("Non-existent subscription", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/subscriptions/99999", nil)
		req.AddCookie(testutil.CookieForUser(t, server, "testuser-delete3", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNoContent { // DELETE operations return 204 even for non-existent resources
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
		}
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/subscriptions/1", nil)
		// No cookie
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
	})
}

func TestHandleUpdateSubscriptionFolderPath(t *testing.T) {
	server, _ := SetupTestServerWithProviders(t)
	router := server.Router()

	t.Run("Update folder path", func(t *testing.T) {
		// Create user and subscription
		userCookie := testutil.CookieForUser(t, server, "testuser-update", "password", "user")

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

		// Get the subscription ID from the response
		var response map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &response)
		subIDFloat, ok := response["id"].(float64)
		if !ok {
			t.Fatalf("Failed to get subscription ID from response")
		}
		subID := int64(subIDFloat)

		updatePayload := map[string]string{
			"folder_path": "updated/custom/path",
		}
		body, _ = json.Marshal(updatePayload)
		req, _ = http.NewRequest("PUT", fmt.Sprintf("/api/subscriptions/%d/folder-path", subID), bytes.NewBuffer(body))
		req.AddCookie(userCookie)
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Verify the update by checking the response message
		var response2 map[string]string
		json.Unmarshal(rr.Body.Bytes(), &response2)
		if response2["message"] != "Subscription folder path updated successfully." {
			t.Errorf("Expected success message, got: %s", response2["message"])
		}
	})

	t.Run("Update folder path to null", func(t *testing.T) {
		// Create user and subscription
		userCookie := testutil.CookieForUser(t, server, "testuser-update2", "password", "user")

		payload := map[string]string{
			"series_title":      "Update Folder Test 2",
			"series_identifier": "update-test-2",
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

		// Get the subscription ID from the response
		var response map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &response)
		subIDFloat, ok := response["id"].(float64)
		if !ok {
			t.Fatalf("Failed to get subscription ID from response")
		}
		subID := int64(subIDFloat)

		updatePayload := map[string]interface{}{
			"folder_path": nil,
		}
		body, _ = json.Marshal(updatePayload)
		req, _ = http.NewRequest("PUT", fmt.Sprintf("/api/subscriptions/%d/folder-path", subID), bytes.NewBuffer(body))
		req.AddCookie(userCookie)
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Verify the update by checking the response message
		var response2 map[string]string
		json.Unmarshal(rr.Body.Bytes(), &response2)
		if response2["message"] != "Subscription folder path updated successfully." {
			t.Errorf("Expected success message, got: %s", response2["message"])
		}
	})

	t.Run("Update non-existent subscription", func(t *testing.T) {
		updatePayload := map[string]string{
			"folder_path": "should/fail",
		}
		body, _ := json.Marshal(updatePayload)
		req, _ := http.NewRequest("PUT", "/api/subscriptions/99999/folder-path", bytes.NewBuffer(body))
		req.AddCookie(testutil.CookieForUser(t, server, "testuser-update3", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest("PUT", "/api/subscriptions/1/folder-path", bytes.NewBufferString("invalid json"))
		req.AddCookie(testutil.CookieForUser(t, server, "testuser-update4", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("Unauthorized", func(t *testing.T) {
		updatePayload := map[string]string{
			"folder_path": "unauthorized",
		}
		body, _ := json.Marshal(updatePayload)
		req, _ := http.NewRequest("PUT", "/api/subscriptions/1/folder-path", bytes.NewBuffer(body))
		// No cookie
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
	})
}

func TestHandleRecheckSubscription(t *testing.T) {
	server, _ := SetupTestServerWithProviders(t)
	router := server.Router()

	// Create a user once for all sub-tests to ensure database is properly initialized
	userCookie := testutil.CookieForUser(t, server, "testuser-recheck", "password", "user")

	t.Run("Success", func(t *testing.T) {
		// Create a subscription first
		payload := map[string]string{
			"series_title":      "Recheck Test",
			"series_identifier": "recheck-test-1",
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

		// Get the subscription ID from the response
		var response map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &response)
		subIDFloat, ok := response["id"].(float64)
		if !ok {
			t.Fatalf("Failed to get subscription ID from response")
		}
		subID := int64(subIDFloat)

		req, _ = http.NewRequest("POST", fmt.Sprintf("/api/subscriptions/%d/recheck", subID), nil)
		req.AddCookie(userCookie)
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusAccepted {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusAccepted)
		}
	})

	t.Run("Non-existent subscription", func(t *testing.T) {
		// Create a fresh user for this test to avoid any session issues
		freshUserCookie := testutil.CookieForUser(t, server, "testuser-recheck3", "password", "user")

		req, _ := http.NewRequest("POST", "/api/subscriptions/99999/recheck", nil)
		req.AddCookie(freshUserCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusAccepted {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusAccepted)
		}
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/subscriptions/1/recheck", nil)
		// No cookie
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
	})
}

func TestPathValidation(t *testing.T) {
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

	t.Run("Invalid folder path update", func(t *testing.T) {
		// Create user and subscription first
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

		updatePayload := map[string]string{
			"folder_path": "../../invalid",
		}
		body, _ = json.Marshal(updatePayload)
		req, _ = http.NewRequest("PUT", "/api/subscriptions/1/folder-path", bytes.NewBuffer(body))
		req.AddCookie(userCookie)
		rr = httptest.NewRecorder()
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
}
