package websocket

import (
	"net/http"
	"testing"
)

func TestOriginAllowed(t *testing.T) {
	same := httptestRequest("localhost:8080", "")
	if !originAllowed(same) {
		t.Fatal("missing Origin should be allowed for non-browser clients")
	}

	same.Header.Set("Origin", "http://localhost:8080")
	if !originAllowed(same) {
		t.Fatal("same-host Origin should be allowed")
	}

	cross := httptestRequest("localhost:8080", "http://evil.example")
	if originAllowed(cross) {
		t.Fatal("cross-origin websocket should be rejected")
	}

	if originAllowed(httptestRequest("localhost:8080", "http://localhost:9090")) {
		t.Fatal("a different port must be rejected")
	}
	if originAllowed(httptestRequest("localhost:8080", "http://localhost:8080.evil.example")) {
		t.Fatal("a host that only prefixes the real host must be rejected")
	}
	if originAllowed(httptestRequest("localhost:8080", "null")) {
		t.Fatal("a null origin must be rejected")
	}
	if !originAllowed(httptestRequest("localhost:8080", "https://localhost:8080")) {
		t.Fatal("the same host over https should be allowed")
	}
}

func httptestRequest(host, origin string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, "/ws/admin/progress", nil)
	req.Host = host
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	return req
}
