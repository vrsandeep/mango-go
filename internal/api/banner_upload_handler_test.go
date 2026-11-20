package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrsandeep/mango-go/internal/api"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func bannerJPEG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 70}); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func postBanner(t *testing.T, router http.Handler, cookie *http.Cookie, folderID int64, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("banner_file", "banner.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	writer.Close()

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/banner", folderID), &body)
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func TestHandleUploadFolderBanner(t *testing.T) {
	server, _, _ := testutil.SetupTestServer(t)
	appData := t.TempDir()
	server.SetAppDataPath(appData)
	router := server.Router()
	cookie := testutil.CookieForUser(t, server, "banneruser", "password", "user")
	folder, err := server.Store().CreateFolder("/library/Blue Lock", "Blue Lock", nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("rejects small image", func(t *testing.T) {
		rr := postBanner(t, router, cookie, folder.ID, bannerJPEG(200, 200))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
		}
		var resp struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if resp.Error == "" {
			t.Fatal("expected dimension error message")
		}
	})

	t.Run("accepts wide banner", func(t *testing.T) {
		rr := postBanner(t, router, cookie, folder.ID, bannerJPEG(1200, 400))
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
		}
		var resp struct {
			BannerImage string `json:"bannerImage"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if resp.BannerImage == "" {
			t.Fatal("missing bannerImage")
		}
		updated, err := server.Store().GetFolder(folder.ID)
		if err != nil {
			t.Fatal(err)
		}
		if updated.Banner == "" {
			t.Fatal("banner path not stored")
		}

		getReq, _ := http.NewRequest("GET", resp.BannerImage, nil)
		getReq.AddCookie(cookie)
		getRR := httptest.NewRecorder()
		router.ServeHTTP(getRR, getReq)
		if getRR.Code != http.StatusOK {
			t.Fatalf("get banner status %d", getRR.Code)
		}

		browseReq, _ := http.NewRequest("GET", fmt.Sprintf("/api/browse?folderId=%d", folder.ID), nil)
		browseReq.AddCookie(cookie)
		browseRR := httptest.NewRecorder()
		router.ServeHTTP(browseRR, browseReq)
		var browse struct {
			CurrentFolder *struct {
				Banner string `json:"banner"`
			} `json:"current_folder"`
		}
		if err := json.Unmarshal(browseRR.Body.Bytes(), &browse); err != nil {
			t.Fatal(err)
		}
		if browse.CurrentFolder == nil || browse.CurrentFolder.Banner != resp.BannerImage {
			t.Fatalf("browse banner %#v", browse.CurrentFolder)
		}

		previous := updated.Banner
		again := postBanner(t, router, cookie, folder.ID, bannerJPEG(1200, 400))
		if again.Code != http.StatusOK {
			t.Fatalf("replace status %d body %s", again.Code, again.Body.String())
		}
		if _, err := os.Stat(filepath.Join(appData, "banners", filepath.Base(previous))); !os.IsNotExist(err) {
			t.Fatalf("old banner still present: %v", err)
		}
	})

	t.Run("rejects bad names and missing files", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/banners/not a name.jpg", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("bad name status %d", rr.Code)
		}

		server.SetAppDataPath("")
		req, _ = http.NewRequest("GET", "/banners/missing-banner-ab12c.jpg", nil)
		req.AddCookie(cookie)
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("missing file status %d", rr.Code)
		}
		server.SetAppDataPath(appData)
	})

	t.Run("rejects bad folder and bad upload", func(t *testing.T) {
		rr := postBanner(t, router, cookie, 0, bannerJPEG(1200, 400))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("id 0 status %d", rr.Code)
		}
		req, _ := http.NewRequest("POST", "/api/folders/nope/banner", strings.NewReader("x"))
		req.AddCookie(cookie)
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("bad id status %d", rr.Code)
		}
		rr = postBanner(t, router, cookie, 99999, bannerJPEG(1200, 400))
		if rr.Code != http.StatusNotFound {
			t.Fatalf("missing folder status %d body %s", rr.Code, rr.Body.String())
		}

		req, _ = http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/banner", folder.ID), nil)
		req.AddCookie(cookie)
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("missing file status %d", rr.Code)
		}
		rr = postBanner(t, router, cookie, folder.ID, []byte("hello"))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("unsupported status %d body %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("save failure is a server error", func(t *testing.T) {
		blocked := filepath.Join(t.TempDir(), "not-a-dir")
		if err := os.WriteFile(blocked, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		server.SetAppDataPath(blocked)
		rr := postBanner(t, router, cookie, folder.ID, bannerJPEG(1200, 400))
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
		}
	})
}

func TestSetAppDataPathNilServer(t *testing.T) {
	(&api.Server{}).SetAppDataPath(t.TempDir())
}
