package plugins

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
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
	data, err := io.ReadAll(io.LimitReader(body, maxPluginHTTPBody+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxPluginHTTPBody {
		return nil, fmt.Errorf("response exceeds %d bytes", maxPluginHTTPBody)
	}
	return data, nil
}

// newPluginHTTPClient is the HTTP client used by mango.http.
// It ignores HTTP_PROXY, follows at most five redirects, and revalidates each
// redirect with validatePluginRequestURL. Connections are made by pluginDial.
func newPluginHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: pluginTransport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("stopped after 5 redirects")
			}
			return validatePluginRequestURL(req.URL.String())
		},
	}
}

var pluginTransport = &http.Transport{
	Proxy:                 nil, // ignore HTTP_PROXY so it cannot retarget plugin requests
	DialContext:           pluginDial,
	TLSHandshakeTimeout:   10 * time.Second,
	ResponseHeaderTimeout: 30 * time.Second,
	IdleConnTimeout:       30 * time.Second,
	MaxIdleConns:          10,
}

// pluginDial resolves addr and connects only to addresses pluginIPAllowed accepts.
// It dials the resolved IP so a later DNS answer cannot retarget the connection.
func pluginDial(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ips, err := lookupPluginIPs(ctx, host)
	if err != nil {
		return nil, err
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	var lastErr error
	for _, ip := range ips {
		if !pluginIPAllowed(ip) {
			lastErr = fmt.Errorf("connection to a local or private address is not allowed")
			continue
		}
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no addresses for host")
	}
	return nil, lastErr
}

// lookupPluginIPs returns host as an IP when it already is one, otherwise the DNS answers.
func lookupPluginIPs(ctx context.Context, host string) ([]net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		return []net.IP{ip}, nil
	}
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	ips := make([]net.IP, 0, len(addrs))
	for _, addr := range addrs {
		ips = append(ips, addr.IP)
	}
	return ips, nil
}

// validatePluginRequestURL allows http and https URLs with a host and no userinfo.
// Literal IP hosts are checked here. Hostnames are checked in pluginDial after DNS.
func validatePluginRequestURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return fmt.Errorf("invalid URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("only http and https URLs are allowed")
	}
	if parsed.User != nil {
		return fmt.Errorf("URLs with credentials are not allowed")
	}
	if ip := net.ParseIP(parsed.Hostname()); ip != nil && !pluginIPAllowed(ip) {
		return fmt.Errorf("connection to a local or private address is not allowed")
	}
	return nil
}

var (
	cgnatNet = mustCIDR("100.64.0.0/10")
	benchNet = mustCIDR("198.18.0.0/15")
)

// mustCIDR parses cidr for the package-level block lists. A bad cidr panics at startup.
func mustCIDR(cidr string) *net.IPNet {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	return network
}

// pluginIPAllowed reports whether a plugin may open a connection to ip.
// Loopback is rejected unless SetAllowLoopbackHTTPForTests enabled it.
// Private, link-local, multicast, unspecified, 0.0.0.0/8, carrier-grade NAT,
// and benchmarking ranges are always rejected.
func pluginIPAllowed(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
		if ip[0] == 0 {
			return false
		}
	}
	if allowLoopbackHTTP.Load() && ip.IsLoopback() {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() || ip.Equal(net.IPv4bcast) {
		return false
	}
	if cgnatNet.Contains(ip) || benchNet.Contains(ip) {
		return false
	}
	return true
}
