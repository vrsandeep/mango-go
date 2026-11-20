package api_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

type searchCounter struct {
	calls *int
	inner *mockAnilistSearcher
}

func (s searchCounter) SearchManga(title string) (*anilist.Media, error) {
	*s.calls++
	return s.inner.SearchManga(title)
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
	_ = server.Store().SetFolderAnilist(folder.ID, 12345, "https://anilist.co/manga/12345", "https://cover.example/large.jpg", "Romaji Title", "English Title", "banners/cached-banner-ab12c.jpg", true)

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
		BannerImage string `json:"bannerImage"`
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
	if resp.BannerImage != "/banners/cached-banner-ab12c.jpg" {
		t.Errorf("bannerImage: got %q", resp.BannerImage)
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
	calls := 0
	server.SetAnilistSearcher(searchCounter{calls: &calls, inner: &mockAnilistSearcher{media: nil}})

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/anilist", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404 (no manga found), got %d", rr.Code)
	}
	cached, err := server.Store().GetFolderAnilist(folder.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !cached.BannerFetched || cached.SiteURL != "" {
		t.Fatalf("miss was not stored: %+v", cached)
	}

	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/folders/%d/anilist", folder.ID), nil)
	req.AddCookie(cookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("cached miss status %d", rr.Code)
	}
	if calls != 1 {
		t.Fatalf("AniList calls = %d, want 1", calls)
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
		ID:          99999,
		SiteURL:     "https://anilist.co/manga/99999",
		Title:       &anilist.Title{Romaji: "Mock Romaji", English: "Mock English"},
		CoverImage:  &anilist.CoverImage{Large: "https://cover.example/mock.jpg"},
		BannerImage: "",
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
		BannerImage string `json:"bannerImage"`
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
	if resp.BannerImage != "" {
		t.Errorf("bannerImage: got %q, want empty", resp.BannerImage)
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
	if cached != nil && (!cached.BannerFetched || cached.BannerPath != "") {
		t.Errorf("empty banner should be stored as fetched: %+v", cached)
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

func TestHandlePostFolderAnilist_StoresBannerImage(t *testing.T) {
	server, router, cookie := setupAnilistTestData(t)
	folder, _ := server.Store().CreateFolder("/library/Bannered", "Bannered", nil)
	server.SetAnilistSearcher(&mockAnilistSearcher{media: &anilist.Media{
		ID:          55,
		SiteURL:     "https://anilist.co/manga/55",
		BannerImage: "https://cdn.example/banner.jpg",
	}})
	server.SetBannerSaver(func(seriesName, sourceURL string) (string, error) {
		return "banners/bannered-banner-ab12c.jpg", nil
	})

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/anilist", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		BannerImage string `json:"bannerImage"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.BannerImage != "/banners/bannered-banner-ab12c.jpg" {
		t.Errorf("bannerImage: got %q", resp.BannerImage)
	}
	cached, err := server.Store().GetFolderAnilist(folder.ID)
	if err != nil {
		t.Fatalf("GetFolderAnilist: %v", err)
	}
	if !cached.BannerFetched || cached.BannerPath != "banners/bannered-banner-ab12c.jpg" {
		t.Errorf("cache banner: fetched=%v path=%q", cached.BannerFetched, cached.BannerPath)
	}
}

func TestHandleGetFolderAnilist_RefreshesNullBanner(t *testing.T) {
	server, db, _ := testutil.SetupTestServer(t)
	router := server.Router()
	cookie := testutil.GetAuthCookie(t, server, "user", "pw", "user")
	folder, _ := server.Store().CreateFolder("/library/Legacy Banner", "Legacy Banner", nil)
	_, err := db.Exec(`INSERT INTO folder_anilist_cache (folder_id, anilist_id, site_url, created_at)
		VALUES (?, ?, ?, datetime('now'))`, folder.ID, 42, "https://anilist.co/manga/42")
	if err != nil {
		t.Fatalf("insert legacy cache: %v", err)
	}
	server.SetAnilistSearcher(&mockAnilistSearcher{media: &anilist.Media{
		ID:          42,
		SiteURL:     "https://anilist.co/manga/42",
		Title:       &anilist.Title{Romaji: "Legacy Romaji", English: "Legacy English"},
		CoverImage:  &anilist.CoverImage{Large: "https://cover.example/legacy.jpg"},
		BannerImage: "https://cdn.example/refreshed.jpg",
	}})
	server.SetBannerSaver(func(seriesName, sourceURL string) (string, error) {
		return "banners/legacy-banner-ab12c.jpg", nil
	})

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/folders/%d/anilist", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		BannerImage string `json:"bannerImage"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.BannerImage != "/banners/legacy-banner-ab12c.jpg" {
		t.Errorf("bannerImage: got %q", resp.BannerImage)
	}
	cached, err := server.Store().GetFolderAnilist(folder.ID)
	if err != nil {
		t.Fatalf("GetFolderAnilist: %v", err)
	}
	if !cached.BannerFetched || cached.BannerPath != "banners/legacy-banner-ab12c.jpg" {
		t.Errorf("cache after refresh: fetched=%v path=%q", cached.BannerFetched, cached.BannerPath)
	}
	if cached.TitleRomaji != "Legacy Romaji" || cached.CoverImageURL != "https://cover.example/legacy.jpg" {
		t.Errorf("refresh did not keep title and cover: %+v", cached)
	}
}

func TestHandlePostFolderAnilist_DownloadsBanner(t *testing.T) {
	server, router, cookie := setupAnilistTestData(t)
	server.SetAppDataPath(t.TempDir())
	folder, _ := server.Store().CreateFolder("/library/Downloaded", "Downloaded", nil)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write([]byte("banner-bytes"))
	}))
	defer srv.Close()
	server.SetAnilistSearcher(&mockAnilistSearcher{media: &anilist.Media{
		ID:          77,
		SiteURL:     "https://anilist.co/manga/77",
		BannerImage: srv.URL + "/banner.jpg",
	}})

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/anilist", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	cached, err := server.Store().GetFolderAnilist(folder.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(cached.BannerPath, "banners/") {
		t.Fatalf("banner path %q", cached.BannerPath)
	}
}

func TestHandleGetFolderAnilist_RefreshDownloadFailureLeavesNull(t *testing.T) {
	server, db, _ := testutil.SetupTestServer(t)
	router := server.Router()
	cookie := testutil.GetAuthCookie(t, server, "user", "pw", "user")
	folder, _ := server.Store().CreateFolder("/library/Download Fail", "Download Fail", nil)
	_, err := db.Exec(`INSERT INTO folder_anilist_cache (folder_id, anilist_id, site_url, created_at)
		VALUES (?, ?, ?, datetime('now'))`, folder.ID, 9, "https://anilist.co/manga/9")
	if err != nil {
		t.Fatalf("insert legacy cache: %v", err)
	}
	server.SetAnilistSearcher(&mockAnilistSearcher{media: &anilist.Media{
		ID:          9,
		SiteURL:     "https://anilist.co/manga/9",
		BannerImage: "https://cdn.example/banner.jpg",
	}})
	server.SetBannerSaver(func(seriesName, sourceURL string) (string, error) {
		return "", errors.New("disk full")
	})

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/folders/%d/anilist", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	cached, err := server.Store().GetFolderAnilist(folder.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cached.BannerFetched {
		t.Fatal("failed download should stay unfetched")
	}
}

func TestHandleGetFolderAnilist_RefreshErrorLeavesNull(t *testing.T) {
	server, db, _ := testutil.SetupTestServer(t)
	router := server.Router()
	cookie := testutil.GetAuthCookie(t, server, "user", "pw", "user")
	folder, _ := server.Store().CreateFolder("/library/Refresh Fail", "Refresh Fail", nil)
	_, err := db.Exec(`INSERT INTO folder_anilist_cache (folder_id, anilist_id, site_url, created_at)
		VALUES (?, ?, ?, datetime('now'))`, folder.ID, 8, "https://anilist.co/manga/8")
	if err != nil {
		t.Fatalf("insert legacy cache: %v", err)
	}
	server.SetAnilistSearcher(&mockAnilistSearcher{err: errors.New("anilist down")})

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/folders/%d/anilist", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	cached, err := server.Store().GetFolderAnilist(folder.ID)
	if err != nil {
		t.Fatalf("GetFolderAnilist: %v", err)
	}
	if cached.BannerFetched {
		t.Error("BannerFetched: failed refresh should leave the banner unfetched")
	}
}
