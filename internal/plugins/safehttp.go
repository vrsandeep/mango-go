package plugins

import (
	"context"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/vrsandeep/mango-go/internal/httpguard"
)

// maxPluginHTTPBody is the largest response body mango.http will accept (16 MiB).
// readPluginHTTPBody reads one extra byte so an oversized body is rejected
// instead of silently truncated.
const maxPluginHTTPBody = 16 << 20

// allowLoopbackHTTP is set by tests that talk to httptest servers.
var allowLoopbackHTTP atomic.Bool

// SetAllowLoopbackHTTPForTests permits plugin HTTP to loopback addresses.
// Production calls leave this false.
func SetAllowLoopbackHTTPForTests(allow bool) {
	allowLoopbackHTTP.Store(allow)
}

// readPluginHTTPBody reads a plugin HTTP response up to maxPluginHTTPBody.
func readPluginHTTPBody(body io.Reader) ([]byte, error) {
	return httpguard.ReadBody(body, maxPluginHTTPBody)
}

// newPluginHTTPClient is the HTTP client used by mango.http.
// It ignores HTTP_PROXY, follows at most five redirects, and revalidates each one.
func newPluginHTTPClient(timeout time.Duration) *http.Client {
	return httpguard.NewClient(timeout, allowLoopbackHTTP.Load())
}

// pluginDial resolves addr and connects only to addresses pluginIPAllowed accepts.
func pluginDial(ctx context.Context, network, addr string) (net.Conn, error) {
	return httpguard.Dial(ctx, network, addr, allowLoopbackHTTP.Load())
}

// validatePluginRequestURL allows http and https URLs with a host and no userinfo.
func validatePluginRequestURL(raw string) error {
	return httpguard.ValidateURL(raw, allowLoopbackHTTP.Load())
}

// pluginIPAllowed reports whether a plugin may open a connection to ip.
func pluginIPAllowed(ip net.IP) bool {
	return httpguard.IPAllowed(ip, allowLoopbackHTTP.Load())
}
