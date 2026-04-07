package chapterfiles

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mholt/archives"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/util"
)

type archiveHandler struct{}

func (archiveHandler) SupportsBaseName(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".cbz", ".zip", ".cbr", ".rar", ".7z", ".cb7":
		return true
	default:
		return false
	}
}

func (h archiveHandler) Inspect(ctx context.Context, filePath string) ([]*models.Page, []byte, error) {
	switch h.archiveType(filePath) {
	case "zip":
		return h.parseCBZ(filePath)
	case "rar":
		return h.parseCBR(ctx, filePath)
	default:
		return nil, nil, fmt.Errorf("unsupported archive %s: %s", filePath, filepath.Ext(filePath))
	}
}

func (h archiveHandler) Page(ctx context.Context, filePath string, pageIndex int) ([]byte, string, error) {
	switch h.archiveType(filePath) {
	case "zip":
		return h.getPageFromCBZ(filePath, pageIndex)
	case "rar":
		return h.getPageFromCBR(ctx, filePath, pageIndex)
	default:
		return nil, "", fmt.Errorf("unsupported archive type: %s", filepath.Ext(filePath))
	}
}

func (archiveHandler) archiveType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".cbz", ".zip":
		return "zip"
	case ".cbr", ".rar", ".7z", ".cb7":
		return "rar"
	default:
		return "unknown"
	}
}

// IsImageFile reports whether the filename has a common raster image extension.
func IsImageFile(name string) bool {
	return hasImageExtension(name)
}

func hasImageExtension(name string) bool {
	imageExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".bmp":  true,
		".tiff": true,
		".webp": true,
	}
	ext := strings.ToLower(filepath.Ext(name))
	return imageExts[ext]
}

func createAndSortPages(filenames []string) []*models.Page {
	pages := make([]*models.Page, len(filenames))
	for i, filename := range filenames {
		pages[i] = &models.Page{FileName: filename}
	}
	sort.Slice(pages, func(i, j int) bool {
		return pageSortFunc(pages[i].FileName, pages[j].FileName)
	})
	for i := range pages {
		pages[i].Index = i
	}
	return pages
}

func readFirstImageData(files []string, fsys fs.FS) ([]byte, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("no image files found")
	}
	firstImage := files[0]
	f, err := fsys.Open(firstImage)
	if err != nil {
		return nil, fmt.Errorf("failed to open first image file: %w", err)
	}
	defer f.Close()
	return io.ReadAll(f)
}

func findImageFilesInZip(r *zip.ReadCloser) []*zip.File {
	var imageFiles []*zip.File
	for _, f := range r.Reader.File {
		if !f.FileInfo().IsDir() && hasImageExtension(f.Name) {
			imageFiles = append(imageFiles, f)
		}
	}
	return imageFiles
}

func findImageFilesInFS(fsys fs.FS) ([]string, error) {
	var imageFiles []string
	err := fs.WalkDir(fsys, ".", func(fpath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if hasImageExtension(d.Name()) {
			imageFiles = append(imageFiles, fpath)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk file system: %w", err)
	}
	return imageFiles, nil
}

func (archiveHandler) parseCBZ(filePath string) ([]*models.Page, []byte, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, nil, err
	}
	defer r.Close()

	imageFiles := findImageFilesInZip(r)
	if len(imageFiles) == 0 {
		return []*models.Page{}, nil, fmt.Errorf("no image files found in archive")
	}

	filenames := make([]string, len(imageFiles))
	for i, f := range imageFiles {
		filenames[i] = f.Name
	}
	pages := createAndSortPages(filenames)

	firstFile := imageFiles[0]
	rc, err := firstFile.Open()
	if err != nil {
		return pages, nil, fmt.Errorf("failed to open first page for thumbnail: %w", err)
	}
	defer func(rc io.ReadCloser) {
		if closeErr := rc.Close(); closeErr != nil {
			log.Printf("Error closing read closer: %v", closeErr)
		}
	}(rc)

	firstPageData, err := io.ReadAll(rc)
	if err != nil {
		return pages, nil, fmt.Errorf("failed to read first page for thumbnail: %w", err)
	}
	return pages, firstPageData, nil
}

func (archiveHandler) parseCBR(ctx context.Context, path string) ([]*models.Page, []byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	fsys, err := archives.FileSystem(ctx, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file system: %w", err)
	}

	imageFiles, err := findImageFilesInFS(fsys)
	if err != nil {
		return nil, nil, err
	}
	if len(imageFiles) == 0 {
		return nil, nil, fmt.Errorf("no image files found in archive %s", path)
	}

	sort.Slice(imageFiles, func(i, j int) bool {
		return pageSortFunc(imageFiles[i], imageFiles[j])
	})
	pages := createAndSortPages(imageFiles)

	firstPageData, err := readFirstImageData(imageFiles, fsys)
	if err != nil {
		return pages, nil, err
	}
	return pages, firstPageData, nil
}

func (archiveHandler) getPageFromCBZ(filePath string, pageIndex int) ([]byte, string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, "", err
	}
	defer r.Close()

	imageFiles := findImageFilesInZip(r)
	if len(imageFiles) == 0 {
		return nil, "", fmt.Errorf("no image files found in archive")
	}

	sort.Slice(imageFiles, func(i, j int) bool {
		return pageSortFunc(imageFiles[i].Name, imageFiles[j].Name)
	})

	if pageIndex < 0 || pageIndex >= len(imageFiles) {
		return nil, "", fmt.Errorf("page index %d out of bounds (0-%d)", pageIndex, len(imageFiles)-1)
	}

	imageFile := imageFiles[pageIndex]
	rc, err := imageFile.Open()
	if err != nil {
		return nil, "", fmt.Errorf("failed to open image file: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read image file: %w", err)
	}
	return data, imageFile.Name, nil
}

func (archiveHandler) getPageFromCBR(ctx context.Context, filePath string, pageIndex int) ([]byte, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	fsys, err := archives.FileSystem(ctx, filePath, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open file system: %w", err)
	}

	imageFiles, err := findImageFilesInFS(fsys)
	if err != nil {
		return nil, "", err
	}
	if len(imageFiles) == 0 {
		return nil, "", fmt.Errorf("no image files found in archive")
	}

	sort.Slice(imageFiles, func(i, j int) bool {
		return pageSortFunc(imageFiles[i], imageFiles[j])
	})

	if pageIndex < 0 || pageIndex >= len(imageFiles) {
		return nil, "", fmt.Errorf("page index %d out of bounds (0-%d)", pageIndex, len(imageFiles)-1)
	}

	f, err := fsys.Open(imageFiles[pageIndex])
	if err != nil {
		return nil, "", fmt.Errorf("failed to open image file: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read image file: %w", err)
	}
	return data, imageFiles[pageIndex], nil
}

func pageSortFunc(a, b string) bool {
	if strings.HasPrefix(a, "__") || strings.HasPrefix(a, ".") {
		return false
	}
	if strings.HasPrefix(b, "__") || strings.HasPrefix(b, ".") {
		return true
	}
	return util.NaturalSortLess(a, b)
}
