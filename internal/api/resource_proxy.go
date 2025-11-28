package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// handleProxyResource proxies a resource request with appropriate headers
// This is useful for resources that require specific headers (e.g., Referer for webtoons)
// or to bypass CORS restrictions
//
// Query parameters:
//   - url: (required) The resource URL to proxy
//   - referer: (optional) Referer header value
//   - user-agent: (optional) User-Agent header value
//   - origin: (optional) Origin header value
//   - headers: (optional) JSON object with additional custom headers
func (s *Server) handleProxyResource(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	// Get the resource URL from query parameter
	resourceURL := query.Get("url")
	if resourceURL == "" {
		RespondWithError(w, http.StatusBadRequest, "Missing 'url' parameter")
		return
	}

	// Validate URL
	parsedURL, err := url.Parse(resourceURL)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid URL")
		return
	}

	// Security: Only allow http/https URLs
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		RespondWithError(w, http.StatusBadRequest, "Only http and https URLs are allowed")
		return
	}

	// Security: Optional - restrict to allowed domains (can be configured)
	// For now, we allow all domains but could add a whitelist

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create request
	req, err := http.NewRequest("GET", resourceURL, nil)
	if err != nil {
		log.Printf("Error creating proxy request: %v", err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to create request")
		return
	}

	// Set headers from query parameters
	// Common headers as individual query params
	if referer := query.Get("referer"); referer != "" {
		req.Header.Set("Referer", referer)
	}
	if userAgent := query.Get("user-agent"); userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	if origin := query.Get("origin"); origin != "" {
		req.Header.Set("Origin", origin)
	}

	// Parse additional headers from JSON query parameter
	if headersJSON := query.Get("headers"); headersJSON != "" {
		var customHeaders map[string]string
		if err := json.Unmarshal([]byte(headersJSON), &customHeaders); err == nil {
			for key, value := range customHeaders {
				req.Header.Set(key, value)
			}
		} else {
			log.Printf("Warning: Failed to parse headers JSON: %v", err)
		}
	}

	// Execute the request
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error fetching proxied resource: %v", err)
		RespondWithError(w, http.StatusBadGateway, "Failed to fetch resource")
		return
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		log.Printf("Proxied resource returned status %d for URL: %s", resp.StatusCode, resourceURL)
		RespondWithError(w, http.StatusBadGateway, "Resource server returned error")
		return
	}

	// Copy content type from the original response
	contentType := resp.Header.Get("Content-Type")
	// Trim any charset or other parameters
	if strings.Contains(contentType, ";") {
		contentType = strings.Split(contentType, ";")[0]
		contentType = strings.TrimSpace(contentType)
	}
	if contentType == "" {
		// Try to infer from extension
		contentType = inferContentType(resourceURL)
	}

	// Copy relevant headers from the response
	w.Header().Set("Content-Type", contentType)

	// Set cache headers based on content type
	if strings.HasPrefix(contentType, "image/") {
		// Images can be cached longer
		w.Header().Set("Cache-Control", "public, max-age=86400") // 1 day
	} else if strings.HasPrefix(contentType, "application/json") {
		// JSON responses should have shorter cache
		w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hour
	} else {
		// Default cache for other content types
		w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hour
	}

	// Copy the resource data to the response
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Error copying proxied resource data: %v", err)
		// Response already started, can't send error
		return
	}
}

// inferContentType tries to infer content type from URL extension
func inferContentType(url string) string {
	lowerURL := strings.ToLower(url)
	switch {
	case strings.HasSuffix(lowerURL, ".jpg") || strings.HasSuffix(lowerURL, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lowerURL, ".png"):
		return "image/png"
	case strings.HasSuffix(lowerURL, ".gif"):
		return "image/gif"
	case strings.HasSuffix(lowerURL, ".webp"):
		return "image/webp"
	case strings.HasSuffix(lowerURL, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(lowerURL, ".json"):
		return "application/json"
	case strings.HasSuffix(lowerURL, ".xml"):
		return "application/xml"
	case strings.HasSuffix(lowerURL, ".html") || strings.HasSuffix(lowerURL, ".htm"):
		return "text/html"
	case strings.HasSuffix(lowerURL, ".css"):
		return "text/css"
	case strings.HasSuffix(lowerURL, ".js"):
		return "application/javascript"
	default:
		return "application/octet-stream"
	}
}

