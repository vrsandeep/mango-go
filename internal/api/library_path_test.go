package api

import (
	"net/http/httptest"
	"testing"
)

func TestRelativeToLibrary(t *testing.T) {
	if got := relativeToLibrary("/manga", "/manga/Series/ch.cbz"); got != "Series/ch.cbz" {
		t.Fatalf("inside library: %q", got)
	}
	if got := relativeToLibrary("/manga", "/manga"); got != "" {
		t.Fatalf("library root: %q", got)
	}
	if got := relativeToLibrary("/manga", "Series/ch.cbz"); got != "Series/ch.cbz" {
		t.Fatalf("already relative: %q", got)
	}

	// A prefix match would turn /manga2 into "2/secret". The path must not escape.
	if got := relativeToLibrary("/manga", "/manga2/secret.cbz"); got != "secret.cbz" {
		t.Fatalf("sibling directory: %q", got)
	}
	if got := relativeToLibrary("/manga", "/etc/passwd"); got != "passwd" {
		t.Fatalf("outside library: %q", got)
	}
}

func TestGetListParamsCapsPerPage(t *testing.T) {
	req := httptest.NewRequest("GET", "/?per_page=10000", nil)
	_, perPage, _, _, _ := getListParams(req)
	if perPage != 200 {
		t.Fatalf("oversized per_page = %d", perPage)
	}

	req = httptest.NewRequest("GET", "/?per_page=0", nil)
	_, perPage, _, _, _ = getListParams(req)
	if perPage != 100 {
		t.Fatalf("missing per_page = %d", perPage)
	}

	req = httptest.NewRequest("GET", "/?per_page=25", nil)
	_, perPage, _, _, _ = getListParams(req)
	if perPage != 25 {
		t.Fatalf("normal per_page = %d", perPage)
	}
}
