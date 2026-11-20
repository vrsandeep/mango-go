package api_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/vrsandeep/mango-go/internal/api"
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
	api.SetProxyLoopbackForTests(true)
	t.Cleanup(func() { api.SetProxyLoopbackForTests(false) })

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

	t.Run("Rejects JSON Resource", func(t *testing.T) {
		jsonURL := mockResourceServer.URL + "/api/data.json"
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/proxy/resource?url=%s", jsonURL), nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnsupportedMediaType {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnsupportedMediaType)
		}
	})

	t.Run("Rejects HTML Resource", func(t *testing.T) {
		htmlURL := mockResourceServer.URL + "/page.html"
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/proxy/resource?url=%s", htmlURL), nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnsupportedMediaType {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnsupportedMediaType)
		}
	})

	t.Run("Ignores Caller Supplied Cookie And Custom Headers", func(t *testing.T) {
		protectedURL := mockResourceServer.URL + "/protected"
		headersJSON := `{"X-Custom-Header":"secret-value","Cookie":"session=stolen"}`
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/proxy/resource?url=%s&headers=%s", protectedURL, headersJSON), nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadGateway {
			t.Errorf("handler returned wrong status code: got %v want %v body %s", status, http.StatusBadGateway, rr.Body.String())
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

	t.Run("Rejects Missing Session", func(t *testing.T) {
		imageURL := mockResourceServer.URL + "/image.jpg"
		for _, addr := range []string{"192.168.1.100:54321", "", "127.0.0.1:54321", "[::1]:54321"} {
			req, _ := http.NewRequest("GET", fmt.Sprintf("/api/proxy/resource?url=%s&referer=https://example.com/", imageURL), nil)
			req.RemoteAddr = addr
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if status := rr.Code; status != http.StatusUnauthorized {
				t.Errorf("addr %q status = %d, want %d", addr, status, http.StatusUnauthorized)
			}
		}
	})

	t.Run("Rejects Private And Link Local Targets", func(t *testing.T) {
		api.SetProxyLoopbackForTests(false)
		t.Cleanup(func() { api.SetProxyLoopbackForTests(true) })
		for _, raw := range []string{"http://127.0.0.1/secret", "http://10.1.2.3/x", "http://169.254.169.254/latest", "http://[::1]/x"} {
			req, _ := http.NewRequest("GET", "/api/proxy/resource?url="+url.QueryEscape(raw), nil)
			req.AddCookie(cookie)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if status := rr.Code; status != http.StatusBadRequest {
				t.Errorf("%s status = %d, want %d", raw, status, http.StatusBadRequest)
			}
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
			ok           bool
			name         string
		}{
			{"/image.png", "image/png", true, "PNG"},
			{"/data.json", "", false, "JSON"},
			{"/style.css", "", false, "CSS"},
			{"/script.js", "", false, "JavaScript"},
			{"/unknown.xyz", "", false, "Unknown"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

				if !tc.ok {
					if status := rr.Code; status != http.StatusUnsupportedMediaType {
						t.Errorf("status = %d, want %d", status, http.StatusUnsupportedMediaType)
					}
					return
				}
				if status := rr.Code; status != http.StatusOK {
					t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
				}
				if contentType := rr.Header().Get("Content-Type"); contentType != tc.expectedType {
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

			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Success"))
		}))
		defer mockServer.Close()

		imageURL := mockServer.URL + "/test.jpg"
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/proxy/resource?url=%s&referer=https://test.com/&user-agent=TestBot/1.0&origin=https://test.com", imageURL), nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v body %s", status, http.StatusOK, rr.Body.String())
		}
		if body := rr.Body.String(); body != "Success" {
			t.Errorf("handler returned wrong body: got %v want %v", body, "Success")
		}
	})

	t.Run("Rejects Redirects", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "http://10.1.2.3/secret", http.StatusFound)
		}))
		defer mockServer.Close()

		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/proxy/resource?url=%s/image.jpg", mockServer.URL), nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusBadGateway {
			t.Errorf("status = %d, want %d", status, http.StatusBadGateway)
		}
	})

	t.Run("Rejects Oversized Body", func(t *testing.T) {
		previous := api.MaxProxyBodyForTests()
		api.SetMaxProxyBodyForTests(4)
		t.Cleanup(func() { api.SetMaxProxyBodyForTests(previous) })

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			w.Write([]byte("12345"))
		}))
		defer mockServer.Close()

		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/proxy/resource?url=%s/x.png", mockServer.URL), nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusBadGateway {
			t.Errorf("status = %d, want %d", status, http.StatusBadGateway)
		}
	})
}

func TestInferContentType(t *testing.T) {
	api.SetProxyLoopbackForTests(true)
	t.Cleanup(func() { api.SetProxyLoopbackForTests(false) })

	testCases := []struct {
		urlSuffix    string
		expectedType string
		ok           bool
		name         string
	}{
		{".jpg", "image/jpeg", true, "JPEG"},
		{".jpeg", "image/jpeg", true, "JPEG alt"},
		{".png", "image/png", true, "PNG"},
		{".gif", "image/gif", true, "GIF"},
		{".webp", "image/webp", true, "WebP"},
		{".svg", "", false, "SVG"},
		{".json", "", false, "JSON"},
		{".html", "", false, "HTML"},
		{".js", "", false, "JavaScript"},
		{".xyz", "", false, "Unknown"},
	}

	server, _, _ := testutil.SetupTestServer(t)
	router := server.Router()
	cookie := testutil.CookieForUser(t, server, "testuser", "password", "user")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

			if !tc.ok {
				if status := rr.Code; status != http.StatusUnsupportedMediaType {
					t.Errorf("%s status = %d, want %d", tc.urlSuffix, status, http.StatusUnsupportedMediaType)
				}
				return
			}
			if status := rr.Code; status != http.StatusOK {
				t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
			}
			if contentType := rr.Header().Get("Content-Type"); contentType != tc.expectedType {
				t.Errorf("handler inferred wrong content type for %s: got %v want %v", tc.urlSuffix, contentType, tc.expectedType)
			}
		})
	}
}
