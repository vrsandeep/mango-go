package banners

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/nfnt/resize"
	"github.com/vrsandeep/mango-go/internal/httpguard"
	_ "golang.org/x/image/webp"
)

const FolderName = "banners"

const maxBannerBytes = 20 << 20

// allowBannerLoopback is set by tests that download from httptest servers.
var allowBannerLoopback atomic.Bool

// SetAllowLoopbackForTests permits banner downloads to dial loopback.
// Production leaves this false.
func SetAllowLoopbackForTests(allow bool) {
	allowBannerLoopback.Store(allow)
}

// Uploaded banners must be a landscape image so they can sit behind the folder title.
// Wider uploads are scaled down to MaxWidth, unless that would make the image shorter than MinHeight.
const (
	MinWidth  = 800
	MinHeight = 300
	MaxWidth  = 2000
)

// ErrUnsupportedImage is returned when the upload is not a JPEG, PNG, or WebP.
var ErrUnsupportedImage = errors.New("unsupported image format or corrupt file")

// DimensionError means an uploaded image is too small or not wide enough.
type DimensionError struct {
	Width  int
	Height int
}

func (e *DimensionError) Error() string {
	return fmt.Sprintf(
		"Banner must be at least %d×%d pixels and wider than it is tall (got %d×%d)",
		MinWidth, MinHeight, e.Width, e.Height,
	)
}

var fileNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*-banner-[a-z0-9]{5}\.(jpg|jpeg|png|webp)$`)

// ValidFileName reports whether name is a banner file we generated.
func ValidFileName(name string) bool {
	return fileNamePattern.MatchString(name)
}

// PublicURL turns a stored relative path into the URL the browser loads.
func PublicURL(stored string) string {
	rel, ok := Relative(stored)
	if !ok {
		return ""
	}
	return "/" + filepath.ToSlash(rel)
}

// Relative accepts "banners/file.jpg" or "/banners/file.jpg".
func Relative(stored string) (string, bool) {
	stored = strings.TrimPrefix(stored, "/")
	stored = filepath.ToSlash(filepath.Clean(stored))
	base := filepath.Base(stored)
	if stored != FolderName+"/"+base || !ValidFileName(base) {
		return "", false
	}
	return stored, true
}

// Save downloads a remote banner into dir and returns the relative path
// (banners/<slug>-banner-<xxxxx>.<ext>). An empty source stores nothing.
// A path that is already local is returned unchanged.
func Save(dir, seriesName, sourceURL string) (string, error) {
	if sourceURL == "" {
		return "", nil
	}
	if rel, ok := Relative(sourceURL); ok {
		return rel, nil
	}
	if !strings.HasPrefix(sourceURL, "http://") && !strings.HasPrefix(sourceURL, "https://") {
		return "", fmt.Errorf("banner source is not a URL")
	}

	allowLoopback := allowBannerLoopback.Load()
	if err := httpguard.ValidateURL(sourceURL, allowLoopback); err != nil {
		return "", err
	}
	client := httpguard.NewClient(30*time.Second, allowLoopback)
	resp, err := client.Get(sourceURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("banner download returned %s", resp.Status)
	}

	body, err := httpguard.ReadBody(resp.Body, maxBannerBytes)
	if err != nil {
		return "", err
	}
	ext := extFor(resp.Header.Get("Content-Type"), sourceURL)
	return writeFile(dir, seriesName, ext, bytes.NewReader(body))
}

// SaveBytes stores an uploaded banner in dir after checking its dimensions.
// Images wider than MaxWidth are scaled down. WebP is re-encoded as JPEG when it is scaled.
func SaveBytes(dir, seriesName string, data []byte) (string, error) {
	ext, err := Validate(data)
	if err != nil {
		return "", err
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", ErrUnsupportedImage
	}
	nw, nh, shrink := fittedSize(cfg.Width, cfg.Height)
	if !shrink {
		return writeFile(dir, seriesName, ext, bytes.NewReader(data))
	}

	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", ErrUnsupportedImage
	}
	resized := resize.Resize(uint(nw), uint(nh), img, resize.Lanczos3)
	encoded, ext, err := encodeImage(resized, format)
	if err != nil {
		return "", err
	}
	return writeFile(dir, seriesName, ext, bytes.NewReader(encoded))
}

// fittedSize returns the size to store. shrink is false when the original should be kept.
func fittedSize(w, h int) (int, int, bool) {
	if w <= MaxWidth {
		return w, h, false
	}
	scale := float64(MaxWidth) / float64(w)
	if int(math.Round(float64(h)*scale)) < MinHeight {
		scale = float64(MinHeight) / float64(h)
	}
	if scale >= 1 {
		return w, h, false
	}
	return int(math.Round(float64(w) * scale)), int(math.Round(float64(h) * scale)), true
}

func encodeImage(img image.Image, format string) ([]byte, string, error) {
	var buf bytes.Buffer
	switch format {
	case "png":
		if err := png.Encode(&buf, img); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), ".png", nil
	default:
		// JPEG stays JPEG. WebP has no encoder here, so a scaled WebP is stored as JPEG.
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), ".jpg", nil
	}
}

// Validate checks that data is a JPEG, PNG, or WebP banner of the required size.
func Validate(data []byte) (string, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", ErrUnsupportedImage
	}
	ext, ok := extByFormat[format]
	if !ok {
		return "", ErrUnsupportedImage
	}
	if cfg.Width < MinWidth || cfg.Height < MinHeight || cfg.Width <= cfg.Height {
		return "", &DimensionError{Width: cfg.Width, Height: cfg.Height}
	}
	return ext, nil
}

// Remove deletes a previously stored banner. Paths that are not local banner files are ignored.
func Remove(dir, stored string) error {
	rel, ok := Relative(stored)
	if !ok {
		return nil
	}
	err := os.Remove(filepath.Join(dir, filepath.Base(rel)))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func writeFile(dir, seriesName, ext string, r io.Reader) (string, error) {
	suffix, err := randomSuffix(5)
	if err != nil {
		return "", err
	}
	name := slug(seriesName) + "-banner-" + suffix + ext
	if !ValidFileName(name) {
		return "", fmt.Errorf("invalid banner file name %q", name)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	full := filepath.Join(dir, name)
	f, err := os.OpenFile(full, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		os.Remove(full)
		return "", err
	}
	return FolderName + "/" + name, nil
}

func slug(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	dash := false
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
			continue
		}
		if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		s = "series"
	}
	if len(s) > 80 {
		s = strings.Trim(s[:80], "-")
	}
	return s
}

func randomSuffix(n int) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i, v := range buf {
		out[i] = alphabet[int(v)%len(alphabet)]
	}
	return string(out), nil
}

var extByFormat = map[string]string{
	"jpeg": ".jpg",
	"png":  ".png",
	"webp": ".webp",
}

func extFor(contentType, sourceURL string) string {
	ct := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	switch ct {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	}
	lower := strings.ToLower(strings.Split(sourceURL, "?")[0])
	switch {
	case strings.HasSuffix(lower, ".png"):
		return ".png"
	case strings.HasSuffix(lower, ".webp"):
		return ".webp"
	default:
		return ".jpg"
	}
}
