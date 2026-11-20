package banners_test

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrsandeep/mango-go/internal/banners"
)

func jpegOf(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 20, G: 40, B: 80, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func TestValidate_RejectsSmallOrNarrow(t *testing.T) {
	if _, err := banners.Validate(jpegOf(100, 100)); err == nil {
		t.Fatal("expected dimension error")
	} else if _, ok := err.(*banners.DimensionError); !ok {
		t.Fatalf("got %T %v", err, err)
	}
	if _, err := banners.Validate(jpegOf(900, 900)); err == nil {
		t.Fatal("expected a landscape image")
	}
	if _, err := banners.Validate(jpegOf(1000, 800)); err != nil {
		t.Fatal(err)
	}
	ext, err := banners.Validate(jpegOf(1200, 400))
	if err != nil {
		t.Fatal(err)
	}
	if ext != ".jpg" {
		t.Fatalf("ext %q", ext)
	}
}

func TestSaveBytes_WritesAcceptedBanner(t *testing.T) {
	dir := t.TempDir()
	rel, err := banners.SaveBytes(dir, "Blue Lock", jpegOf(1200, 400))
	if err != nil {
		t.Fatal(err)
	}
	if !banners.ValidFileName(filepath.Base(rel)) {
		t.Fatalf("name %q", rel)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.Base(rel))); err != nil {
		t.Fatal(err)
	}
}

func TestSave_DownloadsIntoBannersFolder(t *testing.T) {
	banners.SetAllowLoopbackForTests(true)
	t.Cleanup(func() { banners.SetAllowLoopbackForTests(false) })

	body := []byte("not-a-real-jpeg")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(body)
	}))
	defer srv.Close()

	dir := t.TempDir()
	rel, err := banners.Save(dir, "Chainsaw Man", srv.URL+"/banner.jpg")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !strings.HasPrefix(rel, "banners/") || !strings.Contains(rel, "-banner-") || !strings.HasSuffix(rel, ".jpg") {
		t.Fatalf("relative path: %q", rel)
	}
	name := filepath.Base(rel)
	if !banners.ValidFileName(name) {
		t.Fatalf("file name rejected: %s", name)
	}
	full := filepath.Join(dir, name)
	got, err := os.ReadFile(full)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if string(got) != string(body) {
		t.Fatalf("file contents mismatch")
	}
	if banners.PublicURL(rel) != "/"+rel {
		t.Errorf("public URL: %q", banners.PublicURL(rel))
	}

	again, err := banners.Save(dir, "Chainsaw Man", rel)
	if err != nil {
		t.Fatalf("Save existing path: %v", err)
	}
	if again != rel {
		t.Errorf("existing path rewritten: %q", again)
	}
}

func TestValidate_RejectsUnreadableBytes(t *testing.T) {
	_, err := banners.Validate([]byte("not an image"))
	if !errors.Is(err, banners.ErrUnsupportedImage) {
		t.Fatalf("got %v", err)
	}
}

func TestDimensionErrorMessage(t *testing.T) {
	_, err := banners.Validate(jpegOf(100, 100))
	msg := err.Error()
	if !strings.Contains(msg, "100×100") || !strings.Contains(msg, "800×300") {
		t.Fatalf("message %q", msg)
	}
}

func TestPublicURL_RejectsUnknownPath(t *testing.T) {
	if banners.PublicURL("covers/foo.jpg") != "" {
		t.Fatal("expected empty public URL")
	}
	if banners.PublicURL("/banners/blue-lock-banner-ab12c.jpg") != "/banners/blue-lock-banner-ab12c.jpg" {
		t.Fatal("leading slash should still be a public URL")
	}
}

func TestSave_RejectsNonURL(t *testing.T) {
	_, err := banners.Save(t.TempDir(), "Series", "not-a-url")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSave_DownloadErrors(t *testing.T) {
	for _, raw := range []string{
		"http://127.0.0.1:1/banner.jpg",
		"http://10.1.2.3/banner.jpg",
		"http://169.254.169.254/latest",
		"file:///etc/passwd",
	} {
		if _, err := banners.Save(t.TempDir(), "Series", raw); err == nil {
			t.Fatalf("%s was downloaded", raw)
		}
	}

	banners.SetAllowLoopbackForTests(true)
	t.Cleanup(func() { banners.SetAllowLoopbackForTests(false) })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer srv.Close()
	_, err := banners.Save(t.TempDir(), "Series", srv.URL+"/banner.jpg")
	if err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("got %v", err)
	}
}

func TestSave_ContentTypeAndSuffix(t *testing.T) {
	banners.SetAllowLoopbackForTests(true)
	t.Cleanup(func() { banners.SetAllowLoopbackForTests(false) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, ".png"):
			w.Header().Set("Content-Type", "image/png; charset=binary")
		case strings.HasSuffix(r.URL.Path, ".webp"):
			w.Header().Set("Content-Type", "application/octet-stream")
		default:
			w.Header().Set("Content-Type", "image/webp")
		}
		w.Write([]byte("img"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	pngRel, err := banners.Save(dir, "Series", srv.URL+"/banner.png")
	if err != nil || !strings.HasSuffix(pngRel, ".png") {
		t.Fatalf("png: %v %q", err, pngRel)
	}
	webpRel, err := banners.Save(dir, "Series", srv.URL+"/banner.webp?x=1")
	if err != nil || !strings.HasSuffix(webpRel, ".webp") {
		t.Fatalf("webp suffix: %v %q", err, webpRel)
	}
	typed, err := banners.Save(dir, "Series", srv.URL+"/banner")
	if err != nil || !strings.HasSuffix(typed, ".webp") {
		t.Fatalf("webp type: %v %q", err, typed)
	}
}

func TestSaveBytes_DownscalesWideImage(t *testing.T) {
	dir := t.TempDir()
	rel, err := banners.SaveBytes(dir, "Wide", jpegOf(4000, 1000))
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(dir, filepath.Base(rel)))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != banners.MaxWidth || cfg.Height != 500 {
		t.Fatalf("got %d×%d", cfg.Width, cfg.Height)
	}
}

func TestSaveBytes_KeepsShortWideImage(t *testing.T) {
	dir := t.TempDir()
	rel, err := banners.SaveBytes(dir, "Short", jpegOf(4000, 300))
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(dir, filepath.Base(rel)))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != 4000 || cfg.Height != 300 {
		t.Fatalf("got %d×%d, scaling would drop the image below the minimum height", cfg.Width, cfg.Height)
	}
}

func TestSaveBytes_RejectsSmallImage(t *testing.T) {
	_, err := banners.SaveBytes(t.TempDir(), "Series", jpegOf(50, 50))
	if err == nil {
		t.Fatal("expected dimension error")
	}
}

func TestSaveBytes_SlugEdgesAndUnwritableDir(t *testing.T) {
	dir := t.TempDir()
	rel, err := banners.SaveBytes(dir, "@@@", jpegOf(1200, 400))
	if err != nil || !strings.HasPrefix(filepath.Base(rel), "series-banner-") {
		t.Fatalf("empty slug: %v %q", err, rel)
	}
	longName := strings.Repeat("Long Title ", 20)
	rel, err = banners.SaveBytes(dir, longName, jpegOf(1200, 400))
	if err != nil {
		t.Fatal(err)
	}
	if slug := strings.Split(filepath.Base(rel), "-banner-")[0]; len(slug) > 80 {
		t.Fatalf("slug too long: %d %q", len(slug), slug)
	}

	blocked := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocked, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := banners.SaveBytes(filepath.Join(blocked, "banners"), "Series", jpegOf(1200, 400)); err == nil {
		t.Fatal("expected mkdir failure")
	}

	readonly := t.TempDir()
	if err := os.Chmod(readonly, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(readonly, 0o755) })
	if _, err := banners.SaveBytes(readonly, "Series", jpegOf(1200, 400)); err == nil {
		t.Fatal("expected create failure")
	}
}

func TestRemove(t *testing.T) {
	dir := t.TempDir()
	rel, err := banners.SaveBytes(dir, "Blue Lock", jpegOf(1200, 400))
	if err != nil {
		t.Fatal(err)
	}
	full := filepath.Join(dir, filepath.Base(rel))
	if err := banners.Remove(dir, "/"+rel); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(full); !os.IsNotExist(err) {
		t.Fatalf("file still present: %v", err)
	}
	if err := banners.Remove(dir, rel); err != nil {
		t.Fatal(err)
	}
	if err := banners.Remove(dir, "https://example.com/banner.jpg"); err != nil {
		t.Fatal(err)
	}
}

func TestSave_Empty(t *testing.T) {
	rel, err := banners.Save(t.TempDir(), "Series", "")
	if err != nil {
		t.Fatal(err)
	}
	if rel != "" {
		t.Errorf("got %q", rel)
	}
}
