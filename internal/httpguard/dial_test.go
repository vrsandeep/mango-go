package httpguard

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientUsesHTTP2(t *testing.T) {
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, r.Proto)
	}))
	srv.EnableHTTP2 = true
	srv.StartTLS()
	defer srv.Close()

	client := NewClient(5*time.Second, true)
	client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.Proto != "HTTP/2.0" || string(body) != "HTTP/2.0" {
		t.Fatalf("proto response=%s body=%s", resp.Proto, body)
	}
}

func TestDialAllowedPrefersReachableIPv4(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	// An unreachable IPv6 address is listed first. It must not delay the IPv4 connection.
	start := time.Now()
	conn, err := dialAllowed(context.Background(), "tcp", []net.IP{
		net.ParseIP("2001:db8::1"),
		net.ParseIP("127.0.0.1"),
	}, port)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("dial took %s; unreachable IPv6 stalled the connection", elapsed)
	}
}
