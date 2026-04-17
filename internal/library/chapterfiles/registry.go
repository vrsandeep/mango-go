// Package chapterfiles routes chapter files (archives and PDF) to type-specific handlers.
package chapterfiles

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/vrsandeep/mango-go/internal/models"
)

// Handler implements inspect and page extraction for one chapter file kind (e.g. CBZ/CBR).
type Handler interface {
	SupportsBaseName(baseName string) bool
	Inspect(ctx context.Context, path string) (pages []*models.Page, firstPageData []byte, err error)
	Page(ctx context.Context, path string, pageIndex int) (data []byte, fileName string, err error)
}

// Registry holds ordered handlers; the first match for SupportsBaseName wins.
type Registry struct {
	handlers []Handler
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds a handler (later registrations are checked first).
func (r *Registry) Register(h Handler) {
	r.handlers = append([]Handler{h}, r.handlers...)
}

func (r *Registry) handlerForPath(path string) Handler {
	base := filepath.Base(path)
	for _, h := range r.handlers {
		if h.SupportsBaseName(base) {
			return h
		}
	}
	return nil
}

// IsSupportedChapterFile reports whether any registered handler supports the file basename.
func (r *Registry) IsSupportedChapterFile(baseName string) bool {
	for _, h := range r.handlers {
		if h.SupportsBaseName(baseName) {
			return true
		}
	}
	return false
}

// InspectChapterFile loads page list and first-page raster bytes for hashing/thumbnails.
func (r *Registry) InspectChapterFile(ctx context.Context, path string) ([]*models.Page, []byte, error) {
	h := r.handlerForPath(path)
	if h == nil {
		return nil, nil, fmt.Errorf("unsupported chapter file %s: %s", path, filepath.Ext(path))
	}
	return h.Inspect(ctx, path)
}

// GetChapterPage returns raw page bytes and a logical filename (for Content-Type from extension).
func (r *Registry) GetChapterPage(ctx context.Context, path string, pageIndex int) ([]byte, string, error) {
	h := r.handlerForPath(path)
	if h == nil {
		return nil, "", fmt.Errorf("unsupported chapter file type: %s", filepath.Ext(path))
	}
	return h.Page(ctx, path, pageIndex)
}

var defaultRegistry *Registry

func init() {
	defaultRegistry = NewRegistry()
	defaultRegistry.Register(&archiveHandler{})
	defaultRegistry.Register(&pdfHandler{})
}

// RegisterHandler adds a handler to the default registry (e.g. PDF from another package's init).
// Handlers registered later are consulted first.
func RegisterHandler(h Handler) {
	defaultRegistry.Register(h)
}

// Default returns the process-wide chapter file registry.
func Default() *Registry {
	return defaultRegistry
}

// IsSupportedChapterFile uses the default registry.
func IsSupportedChapterFile(baseName string) bool {
	return defaultRegistry.IsSupportedChapterFile(baseName)
}

// InspectChapterFile uses the default registry.
func InspectChapterFile(ctx context.Context, path string) ([]*models.Page, []byte, error) {
	return defaultRegistry.InspectChapterFile(ctx, path)
}

// GetChapterPage uses the default registry.
func GetChapterPage(ctx context.Context, path string, pageIndex int) ([]byte, string, error) {
	return defaultRegistry.GetChapterPage(ctx, path, pageIndex)
}
