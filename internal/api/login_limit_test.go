package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLoginLimiter(t *testing.T) {
	limiter := newLoginLimiter(2, time.Hour)
	if !limiter.allow("203.0.113.5") {
		t.Fatal("expected first attempts to be allowed")
	}
	limiter.record("203.0.113.5")
	limiter.record("203.0.113.5")
	if limiter.allow("203.0.113.5") {
		t.Fatal("expected the address to be blocked")
	}
	if !limiter.allow("203.0.113.6") {
		t.Fatal("expected a different address to be allowed")
	}
}

func TestLoginLimiterAllowDoesNotConsumeBudget(t *testing.T) {
	limiter := newLoginLimiter(1, time.Hour)
	for i := 0; i < 5; i++ {
		if !limiter.allow("203.0.113.7") {
			t.Fatal("checking the limit must not count as a failed login")
		}
	}
}

func TestLoginLimiterExpiresOldFailures(t *testing.T) {
	limiter := newLoginLimiter(1, 15*time.Millisecond)
	limiter.record("203.0.113.9")
	if limiter.allow("203.0.113.9") {
		t.Fatal("expected a fresh failure to block")
	}
	time.Sleep(40 * time.Millisecond)
	if !limiter.allow("203.0.113.9") {
		t.Fatal("expected a failure outside the window to stop blocking")
	}
}

func TestLoginRateLimitResponse(t *testing.T) {
	server := &Server{loginLimiter: newLoginLimiter(1, time.Minute)}
	server.loginLimiter.record("192.0.2.8")

	req := httptest.NewRequest(http.MethodPost, "/api/users/login", strings.NewReader(`{"username":"a","password":"b"}`))
	req.RemoteAddr = "192.0.2.8:4000"
	rr := httptest.NewRecorder()
	server.handleLogin(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d, body %s", rr.Code, http.StatusTooManyRequests, rr.Body.String())
	}
}

func TestRequestIsHTTPS(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if requestIsHTTPS(req) {
		t.Fatal("plain HTTP should not be treated as HTTPS")
	}
	req.Header.Set("X-Forwarded-Proto", "http, https")
	if !requestIsHTTPS(req) {
		t.Fatal("expected the proxy scheme to win")
	}
	req.Header.Set("X-Forwarded-Proto", "https, http")
	if requestIsHTTPS(req) {
		t.Fatal("expected a trailing http scheme to stay insecure")
	}
}

func TestSessionCookieIgnoresClientSuppliedProto(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/users/login", nil)
	cookie := sessionCookie(req, "token", time.Now().Add(time.Hour), 0)
	if cookie.Secure {
		t.Fatal("plain HTTP must not set Secure")
	}
	if cookie.MaxAge != 0 {
		t.Fatal("a zero maxAge must be left unset")
	}

	req.Header.Set("X-Forwarded-Proto", "https, http")
	cookie = sessionCookie(req, "token", time.Now().Add(time.Hour), 0)
	if cookie.Secure {
		t.Fatal("a client-supplied https must not win over the proxy's http")
	}

	expired := sessionCookie(req, "", time.Unix(0, 0), -1)
	if expired.MaxAge != -1 {
		t.Fatalf("logout maxAge = %d, want -1", expired.MaxAge)
	}
}
