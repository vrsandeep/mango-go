package api

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Failed logins are limited per client address. These values are fixed in code;
// config.yml does not override them. NewServer passes them to newLoginLimiter.
// The address is r.RemoteAddr (the direct TCP peer), not X-Forwarded-For.
const (
	loginFailureLimit  = 10
	loginFailureWindow = 30 * time.Minute
)

// loginLimiter counts failed logins per client address.
type loginLimiter struct {
	mu      sync.Mutex
	max     int
	window  time.Duration
	attempt map[string][]time.Time
}

// newLoginLimiter records failed logins per address and blocks once max
// failures fall inside window.
func newLoginLimiter(max int, window time.Duration) *loginLimiter {
	return &loginLimiter{
		max:     max,
		window:  window,
		attempt: make(map[string][]time.Time),
	}
}

// allow reports whether ip has fewer than max failures inside the window.
// It does not record the current attempt.
func (l *loginLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.prune(ip)) < l.max
}

// record stores a failed login for ip at the current time.
func (l *loginLimiter) record(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.attempt[ip] = append(l.prune(ip), time.Now())
}

// prune drops failure timestamps for ip that are older than the window and
// returns the ones that remain. Callers must hold l.mu.
func (l *loginLimiter) prune(ip string) []time.Time {
	cutoff := time.Now().Add(-l.window)
	existing := l.attempt[ip]
	recent := make([]time.Time, 0, len(existing))
	for _, ts := range existing {
		if ts.After(cutoff) {
			recent = append(recent, ts)
		}
	}
	if len(recent) == 0 {
		delete(l.attempt, ip)
		return nil
	}
	l.attempt[ip] = recent
	return recent
}

// clientIP returns the host part of RemoteAddr, the direct TCP peer.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || host == "" {
		if r.RemoteAddr == "" {
			return "unknown"
		}
		return r.RemoteAddr
	}
	return host
}

// requestIsHTTPS reports whether the client connection is TLS, including a
// reverse proxy that sets X-Forwarded-Proto. The last value wins so a proxy
// that appends its own scheme overrides a client-supplied one.
func requestIsHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		return false
	}
	parts := strings.Split(proto, ",")
	last := strings.TrimSpace(parts[len(parts)-1])
	return strings.EqualFold(last, "https")
}
