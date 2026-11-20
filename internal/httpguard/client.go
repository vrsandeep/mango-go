// Package httpguard is the HTTP client for server-side fetches.
// It refuses private and link-local addresses, ignores HTTP_PROXY, and caps response bodies.
package httpguard

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

// NewClient returns a client that dials only addresses IPAllowed accepts.
// allowLoopback is for tests that talk to httptest servers. Production passes false.
// Redirects are followed at most five times and each target is checked again.
func NewClient(timeout time.Duration, allowLoopback bool) *http.Client {
	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return Dial(ctx, network, addr, allowLoopback)
		},
		// A custom dialer disables HTTP/2 unless this is set. Cloudflare APIs such as
		// MangaDex stall or fall behind on HTTP/1.1.
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       30 * time.Second,
		MaxIdleConns:          10,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("stopped after 5 redirects")
			}
			return ValidateURL(req.URL.String(), allowLoopback)
		},
	}
}

// ipv6FallbackDelay matches the standard dual-stack dial: IPv4 starts immediately,
// and IPv6 starts if IPv4 has not connected. A missing IPv6 route must not stall the request.
const ipv6FallbackDelay = 300 * time.Millisecond

// Dial resolves addr and connects only to an allowed IP.
// It dials that IP so a later DNS answer cannot retarget the connection.
func Dial(ctx context.Context, network, addr string, allowLoopback bool) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ips, err := lookupIPs(ctx, host)
	if err != nil {
		return nil, err
	}
	var allowed []net.IP
	var lastErr error
	for _, ip := range ips {
		if !IPAllowed(ip, allowLoopback) {
			lastErr = fmt.Errorf("connection to a local or private address is not allowed")
			continue
		}
		allowed = append(allowed, ip)
	}
	if len(allowed) == 0 {
		if lastErr == nil {
			lastErr = fmt.Errorf("no addresses for host")
		}
		return nil, lastErr
	}
	return dialAllowed(ctx, network, allowed, port)
}

// dialAllowed connects to the first address that answers.
// IPv4 is attempted immediately. IPv6 starts shortly after, and the first success is kept.
func dialAllowed(ctx context.Context, network string, ips []net.IP, port string) (net.Conn, error) {
	var v4, v6 []net.IP
	for _, ip := range ips {
		if ip.To4() != nil {
			v4 = append(v4, ip)
		} else {
			v6 = append(v6, ip)
		}
	}
	primary, secondary := v4, v6
	if len(primary) == 0 {
		primary = v6
		secondary = nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type dialResult struct {
		conn net.Conn
		err  error
	}
	results := make(chan dialResult, 2)
	go func() {
		conn, err := dialSequential(ctx, network, primary, port)
		results <- dialResult{conn, err}
	}()
	waitFor := 1
	if len(secondary) > 0 {
		waitFor = 2
		go func() {
			timer := time.NewTimer(ipv6FallbackDelay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				results <- dialResult{err: ctx.Err()}
				return
			case <-timer.C:
			}
			conn, err := dialSequential(ctx, network, secondary, port)
			results <- dialResult{conn, err}
		}()
	}

	var conn net.Conn
	var lastErr error
	for i := 0; i < waitFor; i++ {
		res := <-results
		if res.conn != nil {
			if conn == nil {
				conn = res.conn
				cancel()
			} else {
				res.conn.Close()
			}
			continue
		}
		if lastErr == nil {
			lastErr = res.err
		}
	}
	if conn != nil {
		return conn, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no addresses for host")
	}
	return nil, lastErr
}

func dialSequential(ctx context.Context, network string, ips []net.IP, port string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	var lastErr error
	for _, ip := range ips {
		if ctx.Err() != nil {
			return nil, ctx.Err()
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

func lookupIPs(ctx context.Context, host string) ([]net.IP, error) {
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

// ValidateURL allows http and https URLs with a host and no userinfo.
// Literal IP hosts are checked here. Hostnames are checked again in Dial.
func ValidateURL(raw string, allowLoopback bool) error {
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
	if ip := net.ParseIP(parsed.Hostname()); ip != nil && !IPAllowed(ip, allowLoopback) {
		return fmt.Errorf("connection to a local or private address is not allowed")
	}
	return nil
}

// ReadBody reads at most limit bytes. A larger body is an error, not a truncated result.
func ReadBody(body io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("response exceeds %d bytes", limit)
	}
	return data, nil
}

var (
	cgnatNet = mustCIDR("100.64.0.0/10")
	benchNet = mustCIDR("198.18.0.0/15")
)

func mustCIDR(cidr string) *net.IPNet {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	return network
}

// IPAllowed reports whether a server-side fetch may connect to ip.
// Loopback is allowed only when allowLoopback is true.
// Private, link-local, multicast, unspecified, 0.0.0.0/8, carrier-grade NAT,
// and benchmarking ranges are always rejected.
func IPAllowed(ip net.IP, allowLoopback bool) bool {
	if ip == nil {
		return false
	}
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
		if ip[0] == 0 {
			return false
		}
	}
	if allowLoopback && ip.IsLoopback() {
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
