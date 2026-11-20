package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestProgressWebSocketRequiresAuth(t *testing.T) {
	server, _, _ := testutil.SetupTestServer(t)
	ts := httptest.NewServer(server.Router())
	t.Cleanup(ts.Close)

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/admin/progress"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected unauthenticated websocket to be rejected")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("status = %d, want %d (%v)", status, http.StatusUnauthorized, err)
	}

	userCookie := testutil.CookieForUser(t, server, "ws-user", "password", "user")
	header := http.Header{}
	header.Add("Cookie", userCookie.Name+"="+userCookie.Value)
	_, resp, err = websocket.DefaultDialer.Dial(wsURL, header)
	if err == nil {
		t.Fatal("expected non-admin websocket to be rejected")
	}
	if resp == nil || resp.StatusCode != http.StatusForbidden {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("user status = %d, want %d (%v)", status, http.StatusForbidden, err)
	}

	adminCookie := testutil.CookieForUser(t, server, "ws-admin", "password", "admin")
	header = http.Header{}
	header.Add("Cookie", adminCookie.Name+"="+adminCookie.Value)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("admin websocket failed: %v", err)
	}
	conn.Close()

	header.Set("Origin", "http://evil.example")
	_, resp, err = websocket.DefaultDialer.Dial(wsURL, header)
	if err == nil {
		t.Fatal("expected cross-origin websocket to be rejected")
	}
	if resp == nil || resp.StatusCode != http.StatusForbidden {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("cross-origin status = %d, want %d (%v)", status, http.StatusForbidden, err)
	}
}
