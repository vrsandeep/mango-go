package api_test

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestHandleUploadFolderCover(t *testing.T) {
	server, db, _ := testutil.SetupTestServer(t)
	router := server.Router()
	cookie := testutil.CookieForUser(t, server, "testuser", "password", "user")
	s := store.New(db)

	// Helper to create a multipart form body with a dummy image
	createMultipartBody := func() (bytes.Buffer, string) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("cover_file", "test.jpg")
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}
		// A simple 1x1 red pixel PNG in bytes
		dummyImageData := []byte{
			0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
			0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
			0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
			0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xff, 0xff, 0x3f,
			0x00, 0x05, 0xfe, 0x02, 0xfe, 0xdc, 0xcc, 0x59, 0xe7, 0x00, 0x00, 0x00,
			0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
		}
		part.Write(dummyImageData)
		writer.Close()
		return body, writer.FormDataContentType()
	}

	t.Run("Success", func(t *testing.T) {
		// 1. Setup a folder in the database to upload a cover for.
		folder, err := s.CreateFolder("/upload_test", "Upload Test", nil)
		if err != nil {
			t.Fatalf("Failed to create test folder: %v", err)
		}

		// 2. Create a multipart form request body with a dummy image file.
		body, contentType := createMultipartBody()

		// 3. Create and execute the HTTP request.
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/cover", folder.ID), &body)
		req.AddCookie(cookie)
		req.Header.Set("Content-Type", contentType)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// 4. Assert the response.
		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// 5. Verify the change in the database.
		updatedFolder, err := s.GetFolder(folder.ID)
		if err != nil {
			t.Fatalf("Failed to fetch folder after update: %v", err)
		}

		if updatedFolder.Thumbnail == "" {
			t.Error("Expected folder thumbnail to be updated, but it was empty.")
		}
		if !strings.HasPrefix(updatedFolder.Thumbnail, "data:image/jpeg;base64,") {
			t.Errorf("Expected thumbnail to be a JPEG data URI, but it was: %s", updatedFolder.Thumbnail)
		}
	})

	t.Run("NonExistentFolder", func(t *testing.T) {
		nonExistentID := int64(99999)
		body, contentType := createMultipartBody()

		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/cover", nonExistentID), &body)
		req.AddCookie(cookie)
		req.Header.Set("Content-Type", contentType)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Expecting 404 Not Found or 400 Bad Request depending on implementation
		if rr.Code != http.StatusNotFound && rr.Code != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code for non-existent folder: got %v want %v or %v", rr.Code, http.StatusNotFound, http.StatusBadRequest)
		}
	})

	t.Run("InvalidFolderID", func(t *testing.T) {
		body, contentType := createMultipartBody()
		req, _ := http.NewRequest("POST", "/api/folders/invalid/cover", &body)
		req.AddCookie(cookie)
		req.Header.Set("Content-Type", contentType)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code for invalid folder ID: got %v want %v", rr.Code, http.StatusBadRequest)
		}
	})
}
