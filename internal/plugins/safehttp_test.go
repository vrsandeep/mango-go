package plugins

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPluginIPAllowed(t *testing.T) {
	SetAllowLoopbackHTTPForTests(false)
	t.Cleanup(func() { SetAllowLoopbackHTTPForTests(false) })

	blocked := []string{
		"127.0.0.1",
		"::1",
		"10.1.2.3",
		"192.168.1.1",
		"172.16.0.1",
		"169.254.169.254",
		"100.64.0.1",
		"0.0.0.0",
		"0.1.2.3",
		"::",
	}
	for _, raw := range blocked {
		if pluginIPAllowed(net.ParseIP(raw)) {
			t.Errorf("expected %s to be blocked", raw)
		}
	}
	if !pluginIPAllowed(net.ParseIP("1.1.1.1")) {
		t.Error("expected a public address to be allowed")
	}
	if !pluginIPAllowed(net.ParseIP("8.8.8.8")) {
		t.Error("expected a public address to be allowed")
	}
	if pluginIPAllowed(nil) {
		t.Error("nil IP must be rejected")
	}
	if pluginIPAllowed(net.ParseIP("224.0.0.1")) {
		t.Error("multicast addresses must be rejected")
	}
	if pluginIPAllowed(net.ParseIP("198.18.0.1")) {
		t.Error("benchmarking range must be rejected")
	}
	if pluginIPAllowed(net.ParseIP("::ffff:127.0.0.1")) {
		t.Error("IPv4-mapped loopback must be rejected")
	}

	SetAllowLoopbackHTTPForTests(true)
	if !pluginIPAllowed(net.ParseIP("127.0.0.1")) {
		t.Error("expected loopback to be allowed when tests opt in")
	}
	if pluginIPAllowed(net.ParseIP("10.1.2.3")) {
		t.Error("opting into loopback must not allow private addresses")
	}
}

func TestPluginDialBlocksLoopback(t *testing.T) {
	SetAllowLoopbackHTTPForTests(false)
	t.Cleanup(func() { SetAllowLoopbackHTTPForTests(false) })

	_, err := pluginDial(context.Background(), "tcp", "127.0.0.1:1")
	if err == nil {
		t.Fatal("expected loopback dial to be rejected")
	}
}

func TestValidatePluginRequestURL(t *testing.T) {
	SetAllowLoopbackHTTPForTests(false)
	t.Cleanup(func() { SetAllowLoopbackHTTPForTests(false) })

	if err := validatePluginRequestURL("https://example.com/page"); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"file:///etc/passwd", "http://127.0.0.1/x", "http://[::1]/x", "http://user:pass@example.com/", "ftp://example.com/", "http://10.1.2.3/", "http://169.254.169.254/latest"} {
		if err := validatePluginRequestURL(raw); err == nil {
			t.Errorf("expected %s to be rejected", raw)
		}
	}
}

func TestReadPluginHTTPBodyRejectsOversize(t *testing.T) {
	exact := bytes.Repeat([]byte("a"), maxPluginHTTPBody)
	data, err := readPluginHTTPBody(bytes.NewReader(exact))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != maxPluginHTTPBody {
		t.Fatalf("len = %d, want %d", len(data), maxPluginHTTPBody)
	}

	over := bytes.Repeat([]byte("b"), maxPluginHTTPBody+1)
	if _, err := readPluginHTTPBody(bytes.NewReader(over)); err == nil {
		t.Fatal("expected a body past the limit to be rejected, not truncated")
	}
}

func TestPluginClientRejectsPrivateRedirect(t *testing.T) {
	SetAllowLoopbackHTTPForTests(true)
	t.Cleanup(func() { SetAllowLoopbackHTTPForTests(false) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://10.1.2.3/secret", http.StatusFound)
	}))
	t.Cleanup(srv.Close)

	resp, err := newPluginHTTPClient(2 * time.Second).Get(srv.URL)
	if resp != nil {
		resp.Body.Close()
	}
	if err == nil {
		t.Fatal("expected a redirect to a private address to be rejected")
	}
}

func TestPluginClientRejectsFileRedirect(t *testing.T) {
	SetAllowLoopbackHTTPForTests(true)
	t.Cleanup(func() { SetAllowLoopbackHTTPForTests(false) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "file:///etc/passwd", http.StatusFound)
	}))
	t.Cleanup(srv.Close)

	resp, err := newPluginHTTPClient(2 * time.Second).Get(srv.URL)
	if resp != nil {
		resp.Body.Close()
	}
	if err == nil {
		t.Fatal("expected a redirect to a file URL to be rejected")
	}
}

func TestPluginClientStopsRedirectLoop(t *testing.T) {
	SetAllowLoopbackHTTPForTests(true)
	t.Cleanup(func() { SetAllowLoopbackHTTPForTests(false) })

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, srv.URL+"/again", http.StatusFound)
	}))
	t.Cleanup(srv.Close)

	resp, err := newPluginHTTPClient(2 * time.Second).Get(srv.URL)
	if resp != nil {
		resp.Body.Close()
	}
	if err == nil {
		t.Fatal("expected the redirect limit to stop a loop")
	}
}

func TestPluginClientIgnoresHTTPProxy(t *testing.T) {
	proxyHit := false
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyHit = true
		http.Error(w, "proxied", http.StatusBadGateway)
	}))
	t.Cleanup(proxy.Close)
	t.Setenv("HTTP_PROXY", proxy.URL)
	t.Setenv("http_proxy", proxy.URL)
	t.Setenv("HTTPS_PROXY", proxy.URL)
	t.Setenv("https_proxy", proxy.URL)
	t.Setenv("NO_PROXY", "")

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("direct"))
	}))
	t.Cleanup(target.Close)

	SetAllowLoopbackHTTPForTests(true)
	t.Cleanup(func() { SetAllowLoopbackHTTPForTests(false) })

	resp, err := newPluginHTTPClient(2 * time.Second).Get(target.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	got, err := readPluginHTTPBody(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "direct" {
		t.Fatalf("body = %q, want direct response", got)
	}
	if proxyHit {
		t.Fatal("plugin HTTP followed HTTP_PROXY")
	}
}
