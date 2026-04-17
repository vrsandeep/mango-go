package chapterfiles

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gen2brain/go-fitz"
	"github.com/vrsandeep/mango-go/internal/models"
)

// pdfRasterDPI balances reader quality and thumbnail / hash payload size.
const pdfRasterDPI = 150.0

// syntheticPageFileName ends in .png so API Content-Type mapping stays unchanged.
const syntheticPageFileName = "page.png"

type pdfHandler struct{}

func (pdfHandler) SupportsBaseName(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".pdf")
}

func (pdfHandler) Inspect(ctx context.Context, path string) ([]*models.Page, []byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	doc, err := fitz.New(path)
	if err != nil {
		return nil, nil, fmt.Errorf("pdf inspect: %w", err)
	}
	defer doc.Close()

	n := doc.NumPage()
	if n <= 0 {
		return nil, nil, fmt.Errorf("no pages found in chapter file")
	}

	pages := make([]*models.Page, n)
	for i := 0; i < n; i++ {
		pages[i] = &models.Page{FileName: fmt.Sprintf("%04d.png", i+1), Index: i}
	}

	first, err := doc.ImagePNG(0, pdfRasterDPI)
	if err != nil {
		return pages, nil, fmt.Errorf("pdf first page raster: %w", err)
	}
	return pages, first, nil
}

func (pdfHandler) Page(ctx context.Context, path string, pageIndex int) ([]byte, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	doc, err := fitz.New(path)
	if err != nil {
		return nil, "", fmt.Errorf("pdf page: %w", err)
	}
	defer doc.Close()

	n := doc.NumPage()
	if pageIndex < 0 || pageIndex >= n {
		return nil, "", fmt.Errorf("page index %d out of range for %d pages", pageIndex, n)
	}

	data, err := doc.ImagePNG(pageIndex, pdfRasterDPI)
	if err != nil {
		return nil, "", fmt.Errorf("pdf page raster: %w", err)
	}
	return data, syntheticPageFileName, nil
}
