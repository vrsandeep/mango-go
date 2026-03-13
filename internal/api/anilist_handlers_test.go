package api_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrsandeep/mango-go/internal/anilist"
	"github.com/vrsandeep/mango-go/internal/api"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

// mockAnilistSearcher returns fixed mock API responses for tests.
type mockAnilistSearcher struct {
	media *anilist.Media
	err   error
}

func (m *mockAnilistSearcher) SearchManga(title string) (*anilist.Media, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.media, nil
}

func setupAnilistTestData(t *testing.T) (*api.Server, http.Handler, *http.Cookie) {
	t.Helper()
	server, _, _ := testutil.SetupTestServer(t)
	router := server.Router()
	cookie := testutil.GetAuthCookie(t, server, "user", "pw", "user")
	return server, router, cookie
}

func TestHandleGetFolderAnilist_InvalidFolderID(t *testing.T) {
	_, router, cookie := setupAnilistTestData(t)

	req, _ := http.NewRequest("GET", "/api/folders/0/anilist", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	req2, _ := http.NewRequest("GET", "/api/folders/invalid/anilist", nil)
	req2.AddCookie(cookie)
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid ID, got %d", rr2.Code)
	}
}

func TestHandleGetFolderAnilist_FolderNotFound(t *testing.T) {
	_, router, cookie := setupAnilistTestData(t)

	req, _ := http.NewRequest("GET", "/api/folders/99999/anilist", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestHandleGetFolderAnilist_CacheMiss(t *testing.T) {
	server, router, cookie := setupAnilistTestData(t)
	folder, _ := server.Store().CreateFolder("/library/No Cache", "No Cache", nil)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/folders/%d/anilist", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404 (cache miss), got %d", rr.Code)
	}
}

func TestHandleGetFolderAnilist_CacheHit(t *testing.T) {
	server, router, cookie := setupAnilistTestData(t)
	folder, _ := server.Store().CreateFolder("/library/Cached", "Cached", nil)
	_ = server.Store().SetFolderAnilist(folder.ID, 12345, "https://anilist.co/manga/12345", "https://cover.example/large.jpg", "Romaji Title", "English Title")

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/folders/%d/anilist", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	var resp struct {
		ID      int64  `json:"id"`
		SiteURL string `json:"siteUrl"`
		Title   *struct {
			Romaji  string `json:"romaji"`
			English string `json:"english"`
		} `json:"title"`
		CoverImage *struct {
			Large string `json:"large"`
		} `json:"coverImage"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ID != 12345 {
		t.Errorf("id: got %d, want 12345", resp.ID)
	}
	if resp.SiteURL != "https://anilist.co/manga/12345" {
		t.Errorf("siteUrl: got %q", resp.SiteURL)
	}
	if resp.Title == nil || resp.Title.Romaji != "Romaji Title" || resp.Title.English != "English Title" {
		t.Errorf("title: got %+v", resp.Title)
	}
	if resp.CoverImage == nil || resp.CoverImage.Large != "https://cover.example/large.jpg" {
		t.Errorf("coverImage: got %+v", resp.CoverImage)
	}
}

func TestHandlePostFolderAnilist_InvalidFolderID(t *testing.T) {
	_, router, cookie := setupAnilistTestData(t)

	req, _ := http.NewRequest("POST", "/api/folders/0/anilist", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandlePostFolderAnilist_FolderNotFound(t *testing.T) {
	_, router, cookie := setupAnilistTestData(t)

	req, _ := http.NewRequest("POST", "/api/folders/99999/anilist", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestHandlePostFolderAnilist_MockReturnsNil_NotFound(t *testing.T) {
	server, router, cookie := setupAnilistTestData(t)
	folder, _ := server.Store().CreateFolder("/library/No Match", "No Match", nil)
	server.SetAnilistSearcher(&mockAnilistSearcher{media: nil, err: nil})

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/anilist", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404 (no manga found), got %d", rr.Code)
	}
}

func TestHandlePostFolderAnilist_MockReturnsError_BadGateway(t *testing.T) {
	server, router, cookie := setupAnilistTestData(t)
	folder, _ := server.Store().CreateFolder("/library/Error", "Error", nil)
	server.SetAnilistSearcher(&mockAnilistSearcher{err: errors.New("network error")})

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/anilist", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", rr.Code)
	}
}

func TestHandlePostFolderAnilist_MockReturnsMedia_Success(t *testing.T) {
	server, router, cookie := setupAnilistTestData(t)
	folder, _ := server.Store().CreateFolder("/library/My Manga", "My Manga", nil)
	mockMedia := &anilist.Media{
		ID:         99999,
		SiteURL:    "https://anilist.co/manga/99999",
		Title:      &anilist.Title{Romaji: "Mock Romaji", English: "Mock English"},
		CoverImage: &anilist.CoverImage{Large: "https://cover.example/mock.jpg"},
	}
	server.SetAnilistSearcher(&mockAnilistSearcher{media: mockMedia})

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/anilist", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		ID      int64  `json:"id"`
		SiteURL string `json:"siteUrl"`
		Title   *struct {
			Romaji  string `json:"romaji"`
			English string `json:"english"`
		} `json:"title"`
		CoverImage *struct {
			Large string `json:"large"`
		} `json:"coverImage"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ID != 99999 {
		t.Errorf("id: got %d, want 99999", resp.ID)
	}
	if resp.SiteURL != "https://anilist.co/manga/99999" {
		t.Errorf("siteUrl: got %q", resp.SiteURL)
	}
	if resp.Title == nil || resp.Title.Romaji != "Mock Romaji" {
		t.Errorf("title: got %+v", resp.Title)
	}
	if resp.CoverImage == nil || resp.CoverImage.Large != "https://cover.example/mock.jpg" {
		t.Errorf("coverImage: got %+v", resp.CoverImage)
	}

	// Verify cache was written: GET should now return the same data
	reqGet, _ := http.NewRequest("GET", fmt.Sprintf("/api/folders/%d/anilist", folder.ID), nil)
	reqGet.AddCookie(cookie)
	rrGet := httptest.NewRecorder()
	router.ServeHTTP(rrGet, reqGet)
	if rrGet.Code != http.StatusOK {
		t.Errorf("GET after POST: expected 200, got %d", rrGet.Code)
	}
	cached, _ := server.Store().GetFolderAnilist(folder.ID)
	if cached == nil || cached.AnilistID != 99999 {
		t.Errorf("cache not populated: got %+v", cached)
	}
}

func TestHandlePostFolderAnilist_Unauthorized(t *testing.T) {
	server, router, _ := setupAnilistTestData(t)
	folder, _ := server.Store().CreateFolder("/library/NoAuth", "NoAuth", nil)
	server.SetAnilistSearcher(&mockAnilistSearcher{media: &anilist.Media{ID: 1, SiteURL: "https://anilist.co/manga/1"}})

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/anilist", folder.ID), nil)
	// no cookie
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 without auth, got %d", rr.Code)
	}
}
