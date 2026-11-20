package downloader

import (
	"net/url"
	"strings"
	"testing"
)

func TestUnwrapProxyURL(t *testing.T) {
	target := "https://cdn.example/page.jpg"
	raw := "http://localhost:8080/api/proxy/resource?url=" + url.QueryEscape(target) +
		"&referer=" + url.QueryEscape("https://example.com/") +
		"&cookie=" + url.QueryEscape("session=stolen")
	got, headers := unwrapProxyURL(raw)
	if got != target {
		t.Fatalf("target = %s", got)
	}
	if headers.Get("Referer") != "https://example.com/" {
		t.Fatalf("referer = %q", headers.Get("Referer"))
	}
	if headers.Get("Cookie") != "" {
		t.Fatal("proxy cookie query must not become a request header")
	}

	direct, extra := unwrapProxyURL("https://cdn.example/page.jpg")
	if direct != "https://cdn.example/page.jpg" || extra != nil {
		t.Fatalf("direct url changed: %s %v", direct, extra)
	}
}

func TestFetchPageRejectsPrivateTargets(t *testing.T) {
	for _, raw := range []string{
		"http://127.0.0.1/secret",
		"http://10.1.2.3/x",
		"http://169.254.169.254/latest",
		"http://localhost:8080/api/proxy/resource?url=" + url.QueryEscape("http://169.254.169.254/latest"),
	} {
		_, data, resp, err := fetchPage(raw)
		if resp != nil {
			resp.Body.Close()
		}
		if err == nil {
			t.Fatalf("%s was fetched (%d bytes)", raw, len(data))
		}
		if !strings.Contains(err.Error(), "not allowed") {
			t.Fatalf("%s error = %v", raw, err)
		}
	}
}
