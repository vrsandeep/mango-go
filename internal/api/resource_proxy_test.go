package api_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrsandeep/mango-go/internal/testutil"
)

// setupMockResourceServer creates a mock HTTP server that simulates external resources
func setupMockResourceServer() *httptest.Server {
	mux := http.NewServeMux()

	// Mock image endpoint that checks for Referer header
	mux.HandleFunc("/image.jpg", func(w http.ResponseWriter, r *http.Request) {
		referer := r.Header.Get("Referer")
		if referer == "" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Missing Referer header"))
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write([]byte("fake-image-data"))
	})

	// Mock JSON endpoint
	mux.HandleFunc("/api/data.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","data":[1,2,3]}`))
	})

	// Mock HTML endpoint
	mux.HandleFunc("/page.html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>Test Page</body></html>`))
	})

	// Mock endpoint that requires custom headers
	mux.HandleFunc("/protected", func(w http.ResponseWriter, r *http.Request) {
		customHeader := r.Header.Get("X-Custom-Header")
		if customHeader != "secret-value" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Missing required header"))
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Protected content"))
	})

	// Mock endpoint that returns error status
	mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not found"))
	})

	return httptest.NewServer(mux)
}

func TestHandleProxyResource(t *testing.T) {
	server, _, _ := testutil.SetupTestServer(t)
	router := server.Router()
	cookie := testutil.CookieForUser(t, server, "testuser", "password", "user")

	mockResourceServer := setupMockResourceServer()
	defer mockResourceServer.Close()

	t.Run("Success - Image with Referer", func(t *testing.T) {
		imageURL := mockResourceServer.URL + "/image.jpg"
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/proxy/resource?url=%s&referer=https://example.com/", imageURL), nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		if contentType := rr.Header().Get("Content-Type"); contentType != "image/jpeg" {
			t.Errorf("handler returned wrong content type: got %v want %v", contentType, "image/jpeg")
		}
		if cacheControl := rr.Header().Get("Cache-Control"); !strings.Contains(cacheControl, "max-age=86400") {
			t.Errorf("handler should set 1-day cache for images, got: %v", cacheControl)
		}
		if body := rr.Body.String(); body != "fake-image-data" {
			t.Errorf("handler returned wrong body: got %v want %v", body, "fake-image-data")
		}
	})

	t.Run("Success - JSON Resource", func(t *testing.T) {
		jsonURL := mockResourceServer.URL + "/api/data.json"
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/proxy/resource?url=%s", jsonURL), nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
			t.Errorf("handler returned wrong content type: got %v want %v", contentType, "application/json")
		}
		if cacheControl := rr.Header().Get("Cache-Control"); !strings.Contains(cacheControl, "max-age=3600") {
			t.Errorf("handler should set 1-hour cache for JSON, got: %v", cacheControl)
		}
	})

	t.Run("Success - HTML Resource", func(t *testing.T) {
		htmlURL := mockResourceServer.URL + "/page.html"
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/proxy/resource?url=%s", htmlURL), nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		if contentType := rr.Header().Get("Content-Type"); contentType != "text/html" {
			t.Errorf("handler returned wrong content type: got %v want %v", contentType, "text/html")
		}
	})

	t.Run("Success - Custom Headers via JSON", func(t *testing.T) {
		protectedURL := mockResourceServer.URL + "/protected"
		headersJSON := `{"X-Custom-Header":"secret-value"}`
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/proxy/resource?url=%s&headers=%s", protectedURL, headersJSON), nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		if body := rr.Body.String(); body != "Protected content" {
			t.Errorf("handler returned wrong body: got %v want %v", body, "Protected content")
		}
	})

	t.Run("Success - Multiple Headers", func(t *testing.T) {
		imageURL := mockResourceServer.URL + "/image.jpg"
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/proxy/resource?url=%s&referer=https://example.com/&user-agent=TestBot/1.0&origin=https://example.com", imageURL), nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})

	t.Run("Unauthorized - No Auth Cookie", func(t *testing.T) {
		imageURL := mockResourceServer.URL + "/image.jpg"
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/proxy/resource?url=%s", imageURL), nil)
		// No cookie added
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
	})

	t.Run("Bad Request - Missing URL", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/proxy/resource", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("Bad Request - Invalid URL", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/proxy/resource?url=not-a-valid-url", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("Bad Request - Non-HTTP URL", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/proxy/resource?url=file:///etc/passwd", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("Bad Gateway - Resource Returns Error", func(t *testing.T) {
		errorURL := mockResourceServer.URL + "/error"
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/proxy/resource?url=%s", errorURL), nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadGateway {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadGateway)
		}
	})

	t.Run("Content Type Inference", func(t *testing.T) {
		testCases := []struct {
			url          string
			expectedType string
			name         string
		}{
			{"/image.png", "image/png", "PNG"},
			{"/data.json", "application/json", "JSON"},
			{"/style.css", "text/css", "CSS"},
			{"/script.js", "application/javascript", "JavaScript"},
			{"/unknown.xyz", "application/octet-stream", "Unknown"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Create a mock endpoint that explicitly doesn't set Content-Type
				mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Set Content-Type to empty to force inference
					w.Header().Set("Content-Type", "")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("test"))
				}))
				defer mockServer.Close()

				reqURL := mockServer.URL + tc.url
				req, _ := http.NewRequest("GET", fmt.Sprintf("/api/proxy/resource?url=%s", reqURL), nil)
				req.AddCookie(cookie)
				rr := httptest.NewRecorder()
				router.ServeHTTP(rr, req)

				if status := rr.Code; status != http.StatusOK {
					t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
				}
				// Get content type and strip charset if present
				contentType := rr.Header().Get("Content-Type")
				if strings.Contains(contentType, ";") {
					contentType = strings.Split(contentType, ";")[0]
					contentType = strings.TrimSpace(contentType)
				}
				if contentType != tc.expectedType {
					t.Errorf("handler inferred wrong content type for %s: got %v want %v", tc.url, contentType, tc.expectedType)
				}
			})
		}
	})

	t.Run("Headers Passed Correctly", func(t *testing.T) {
		// Create a mock server that checks headers
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			referer := r.Header.Get("Referer")
			userAgent := r.Header.Get("User-Agent")
			origin := r.Header.Get("Origin")

			if referer != "https://test.com/" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Wrong Referer"))
				return
			}
			if userAgent != "TestBot/1.0" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Wrong User-Agent"))
				return
			}
			if origin != "https://test.com" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Wrong Origin"))
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Success"))
		}))
		defer mockServer.Close()

		imageURL := mockServer.URL + "/test"
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/proxy/resource?url=%s&referer=https://test.com/&user-agent=TestBot/1.0&origin=https://test.com", imageURL), nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		if body := rr.Body.String(); body != "Success" {
			t.Errorf("handler returned wrong body: got %v want %v", body, "Success")
		}
	})
}

func TestInferContentType(t *testing.T) {
	// This tests the helper function indirectly through the proxy handler
	testCases := []struct {
		urlSuffix    string
		expectedType string
		name         string
	}{
		{".jpg", "image/jpeg", "JPEG"},
		{".jpeg", "image/jpeg", "JPEG alt"},
		{".png", "image/png", "PNG"},
		{".gif", "image/gif", "GIF"},
		{".webp", "image/webp", "WebP"},
		{".svg", "image/svg+xml", "SVG"},
		{".json", "application/json", "JSON"},
		{".xml", "application/xml", "XML"},
		{".html", "text/html", "HTML"},
		{".htm", "text/html", "HTML alt"},
		{".css", "text/css", "CSS"},
		{".js", "application/javascript", "JavaScript"},
		{".xyz", "application/octet-stream", "Unknown"},
	}

	server, _, _ := testutil.SetupTestServer(t)
	router := server.Router()
	cookie := testutil.CookieForUser(t, server, "testuser", "password", "user")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock server that explicitly doesn't set Content-Type
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Set Content-Type to empty to force inference
				w.Header().Set("Content-Type", "")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("test"))
			}))
			defer mockServer.Close()

			reqURL := mockServer.URL + "/test" + tc.urlSuffix
			req, _ := http.NewRequest("GET", fmt.Sprintf("/api/proxy/resource?url=%s", reqURL), nil)
			req.AddCookie(cookie)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
			}
			// Get content type and strip charset if present
			contentType := rr.Header().Get("Content-Type")
			if strings.Contains(contentType, ";") {
				contentType = strings.Split(contentType, ";")[0]
				contentType = strings.TrimSpace(contentType)
			}
			if contentType != tc.expectedType {
				t.Errorf("handler inferred wrong content type for %s: got %v want %v", tc.urlSuffix, contentType, tc.expectedType)
			}
		})
	}
}

