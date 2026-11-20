package api

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/vrsandeep/mango-go/internal/httpguard"
)

// maxProxyBody is the largest image the resource proxy will return (20 MiB).
var maxProxyBody int64 = 20 << 20

// MaxProxyBodyForTests returns the current proxy body cap.
func MaxProxyBodyForTests() int64 { return maxProxyBody }

// SetMaxProxyBodyForTests replaces the proxy body cap. Tests restore the previous value.
func SetMaxProxyBodyForTests(n int64) { maxProxyBody = n }

// allowProxyLoopback is set by tests that proxy httptest servers.
var allowProxyLoopback atomic.Bool

// SetProxyLoopbackForTests permits the resource proxy to dial loopback.
// Production leaves this false.
func SetProxyLoopbackForTests(allow bool) {
	allowProxyLoopback.Store(allow)
}

// handleProxyResource fetches an image for a logged-in client.
// Plugins use it when a host requires a Referer or User-Agent.
// Query parameters:
//   - url: (required) image URL
//   - referer, user-agent, origin: optional request headers
func (s *Server) handleProxyResource(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	resourceURL := query.Get("url")
	if resourceURL == "" {
		RespondWithError(w, http.StatusBadRequest, "Missing 'url' parameter")
		return
	}
	if err := httpguard.ValidateURL(resourceURL, allowProxyLoopback.Load()); err != nil {
		RespondWithError(w, http.StatusBadRequest, "URL is not allowed")
		return
	}

	// Do not follow redirects. A public image must not pivot to another host.
	client := httpguard.NewClient(30*time.Second, allowProxyLoopback.Load())
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	req, err := http.NewRequest("GET", resourceURL, nil)
	if err != nil {
		log.Printf("Error creating proxy request: %v", err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to create request")
		return
	}
	setProxyRequestHeaders(req, query)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error fetching proxied resource: %v", err)
		RespondWithError(w, http.StatusBadGateway, "Failed to fetch resource")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Proxied resource returned status %d", resp.StatusCode)
		RespondWithError(w, http.StatusBadGateway, "Resource server returned error")
		return
	}

	contentType, ok := allowedImageType(resp.Header.Get("Content-Type"), resourceURL)
	if !ok {
		RespondWithError(w, http.StatusUnsupportedMediaType, "Only image resources can be proxied")
		return
	}

	body, err := httpguard.ReadBody(resp.Body, maxProxyBody)
	if err != nil {
		log.Printf("Error reading proxied resource: %v", err)
		RespondWithError(w, http.StatusBadGateway, "Failed to read resource")
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write(body)
}

// setProxyRequestHeaders copies the three headers plugins need.
// Cookie, Authorization, Host, and any other caller-supplied header are ignored.
func setProxyRequestHeaders(req *http.Request, query url.Values) {
	if referer := query.Get("referer"); referer != "" {
		req.Header.Set("Referer", referer)
	}
	if userAgent := query.Get("user-agent"); userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	if origin := query.Get("origin"); origin != "" {
		req.Header.Set("Origin", origin)
	}
	raw := query.Get("headers")
	if raw == "" {
		return
	}
	var extra map[string]string
	if err := json.Unmarshal([]byte(raw), &extra); err != nil {
		log.Printf("Warning: Failed to parse headers JSON: %v", err)
		return
	}
	for key, value := range extra {
		switch strings.ToLower(key) {
		case "referer", "user-agent", "origin":
			req.Header.Set(key, value)
		}
	}
}

// allowedImageType accepts JPEG, PNG, GIF, and WebP. SVG and HTML are rejected
// because they can carry script when served from this origin.
func allowedImageType(header, rawURL string) (string, bool) {
	contentType := strings.ToLower(strings.TrimSpace(header))
	if i := strings.Index(contentType, ";"); i >= 0 {
		contentType = strings.TrimSpace(contentType[:i])
	}
	switch contentType {
	case "image/jpeg", "image/jpg":
		return "image/jpeg", true
	case "image/png", "image/gif", "image/webp":
		return contentType, true
	}
	if contentType != "" && contentType != "application/octet-stream" {
		return "", false
	}
	switch inferImageType(rawURL) {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return inferImageType(rawURL), true
	default:
		return "", false
	}
}

func inferImageType(rawURL string) string {
	path := rawURL
	if parsed, err := url.Parse(rawURL); err == nil {
		path = parsed.Path
	}
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".gif"):
		return "image/gif"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	default:
		return ""
	}
}
