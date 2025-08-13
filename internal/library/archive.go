// This file is responsible for parsing archive files like .cbz (ZIP) and
// .cbr (RAR) to get a list of the image files they contain.

package library

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

// IsImageFile checks if a filename has a common image file extension.
func IsImageFile(name string) bool {
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

// IsSupportedArchive checks if a filename has a supported archive extension.
func IsSupportedArchive(name string) bool {
	archiveExts := map[string]bool{
		".cbz": true,
		".zip": true,
		".cbr": true,
		".rar": true,
		".7z":  true,
		".cb7": true,
	}
	ext := strings.ToLower(filepath.Ext(name))
	return archiveExts[ext]
}

// getArchiveType returns the normalized archive type based on file extension
func getArchiveType(filePath string) string {
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

// createAndSortPages creates Page models from filenames and sorts them
func createAndSortPages(filenames []string) []*models.Page {
	pages := make([]*models.Page, len(filenames))
	for i, filename := range filenames {
		pages[i] = &models.Page{FileName: filename}
	}

	// Sort pages alphabetically by filename to ensure correct order
	sort.Slice(pages, func(i, j int) bool {
		return pageSortFunc(pages[i].FileName, pages[j].FileName)
	})

	// Assign index after sorting
	for i := range pages {
		pages[i].Index = i
	}

	return pages
}

// readFirstImageData reads the first image file from a list of files
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

// findImageFilesInZip finds all image files in a ZIP archive
func findImageFilesInZip(r *zip.ReadCloser) []*zip.File {
	var imageFiles []*zip.File
	for _, f := range r.Reader.File {
		if !f.FileInfo().IsDir() && IsImageFile(f.Name) {
			imageFiles = append(imageFiles, f)
		}
	}
	return imageFiles
}

// findImageFilesInFS finds all image files in a filesystem
func findImageFilesInFS(fsys fs.FS) ([]string, error) {
	var imageFiles []string

	err := fs.WalkDir(fsys, ".", func(fpath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if IsImageFile(d.Name()) {
			imageFiles = append(imageFiles, fpath)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk file system: %w", err)
	}

	return imageFiles, nil
}

// ParseArchive loads the archive file and returns a list of pages and the first page's data.
func ParseArchive(filePath string) (pages []*models.Page, firstPageData []byte, err error) {
	archiveType := getArchiveType(filePath)
	switch archiveType {
	case "zip":
		return parseCBZ(filePath)
	case "rar":
		return parseCBR(filePath)
	default:
		return nil, nil, fmt.Errorf("unsupported archive %s: %s", filePath, filepath.Ext(filePath))
	}
}

// parseCBZ reads a .cbz (zip) file and returns a sorted list of image pages.
func parseCBZ(filePath string) ([]*models.Page, []byte, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, nil, err
	}
	defer r.Close()

	imageFiles := findImageFilesInZip(r)
	if len(imageFiles) == 0 {
		return []*models.Page{}, nil, fmt.Errorf("no image files found in archive")
	}

	// Extract filenames for page creation
	filenames := make([]string, len(imageFiles))
	for i, f := range imageFiles {
		filenames[i] = f.Name
	}

	pages := createAndSortPages(filenames)

	// Read the first image for thumbnail
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

// parseCBR opens the given archive or directory path, finds all image files, picks the lexically first image file,
// and returns its bytes.
func parseCBR(path string) ([]*models.Page, []byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

	// Sort image files lexically
	sort.Slice(imageFiles, func(i, j int) bool {
		return pageSortFunc(imageFiles[i], imageFiles[j])
	})

	pages := createAndSortPages(imageFiles)

	// Read the first image for thumbnail
	firstPageData, err := readFirstImageData(imageFiles, fsys)
	if err != nil {
		return pages, nil, err
	}

	return pages, firstPageData, nil
}

// GetPageFromArchive extracts a specific page from an archive file and returns its data.
// pageIndex is 0-based.
func GetPageFromArchive(filePath string, pageIndex int) ([]byte, string, error) {
	archiveType := getArchiveType(filePath)
	switch archiveType {
	case "zip":
		return getPageFromCBZ(filePath, pageIndex)
	case "rar":
		return getPageFromCBR(filePath, pageIndex)
	default:
		return nil, "", fmt.Errorf("unsupported archive type: %s", filepath.Ext(filePath))
	}
}

// getPageFromCBZ extracts a specific page from a .cbz (zip) file.
func getPageFromCBZ(filePath string, pageIndex int) ([]byte, string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, "", err
	}
	defer r.Close()

	imageFiles := findImageFilesInZip(r)
	if len(imageFiles) == 0 {
		return nil, "", fmt.Errorf("no image files found in archive")
	}

	// Sort files alphabetically to ensure correct order
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

// getPageFromCBR extracts a specific page from a .cbr/.rar/.7z/.cb7 file.
func getPageFromCBR(filePath string, pageIndex int) ([]byte, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

	// Sort image files lexically
	sort.Slice(imageFiles, func(i, j int) bool {
		return pageSortFunc(imageFiles[i], imageFiles[j])
	})

	if pageIndex < 0 || pageIndex >= len(imageFiles) {
		return nil, "", fmt.Errorf("page index %d out of bounds (0-%d)", pageIndex, len(imageFiles)-1)
	}

	// Open the specific image file
	f, err := fsys.Open(imageFiles[pageIndex])
	if err != nil {
		return nil, "", fmt.Errorf("failed to open image file: %w", err)
	}
	defer f.Close()

	// Read all bytes from the image file
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read image file: %w", err)
	}

	return data, imageFiles[pageIndex], nil
}

// pageSortFunc sorts pages by filename.
// It handles files starting with __ or . by putting them at the end.
// It also handles natural sorting of numbers.
// It does not handle files starting with __ or . that are not numbers. It
// will put them at the end.
func pageSortFunc(a, b string) bool {

	// if a.Name starts with __ it is likely a hidden file
	if strings.HasPrefix(a, "__") || strings.HasPrefix(a, ".") {
		return false
	}
	if strings.HasPrefix(b, "__") || strings.HasPrefix(b, ".") {
		return true
	}
	return util.NaturalSortLess(a, b)
}
