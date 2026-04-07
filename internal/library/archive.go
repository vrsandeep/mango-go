// Package library re-exports chapter file helpers for archives (CBZ/CBR/…).
// Type-specific logic lives in internal/library/chapterfiles.

package library

import (
	"context"

	"github.com/vrsandeep/mango-go/internal/library/chapterfiles"
	"github.com/vrsandeep/mango-go/internal/models"
)

// IsImageFile reports whether the filename has a common raster image extension.
func IsImageFile(name string) bool {
	return chapterfiles.IsImageFile(name)
}

// IsSupportedChapterFile reports whether the basename is a supported chapter file (archive today).
func IsSupportedChapterFile(name string) bool {
	return chapterfiles.IsSupportedChapterFile(name)
}

// IsSupportedArchive is an alias for IsSupportedChapterFile (archives only in phase 1).
func IsSupportedArchive(name string) bool {
	return IsSupportedChapterFile(name)
}

// InspectChapterFile loads page metadata and first-page bytes for the chapter file at path.
func InspectChapterFile(ctx context.Context, path string) ([]*models.Page, []byte, error) {
	return chapterfiles.InspectChapterFile(ctx, path)
}

// ParseArchive loads the archive and returns pages and first-page image bytes.
// Deprecated: use InspectChapterFile; kept for existing callers and tests.
func ParseArchive(filePath string) (pages []*models.Page, firstPageData []byte, err error) {
	return chapterfiles.InspectChapterFile(context.Background(), filePath)
}

// GetChapterPage returns page bytes and a filename hint for Content-Type.
func GetChapterPage(ctx context.Context, path string, pageIndex int) ([]byte, string, error) {
	return chapterfiles.GetChapterPage(ctx, path, pageIndex)
}

// GetPageFromArchive extracts a page from a chapter file (0-based index).
// Deprecated: use GetChapterPage with request context.
func GetPageFromArchive(filePath string, pageIndex int) ([]byte, string, error) {
	return chapterfiles.GetChapterPage(context.Background(), filePath, pageIndex)
}
