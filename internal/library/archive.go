// This file is responsible for parsing archive files like .cbz (ZIP) and
// .cbr (RAR) to get a list of the image files they contain.

package library

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mholt/archives"
	"github.com/vrsandeep/mango-go/internal/models"
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

// ParseArchive dispatches to the correct parser based on file extension.
func ParseArchive(filePath string) (pages []*models.Page, firstPageData []byte, err error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".cbz", ".zip":
		return parseCBZ(filePath)
	case ".cbr", ".rar", ".7z", ".cb7":
		return parseCBR(filePath)
	default:
		return nil, nil, fmt.Errorf("unsupported archive type: %s", ext)
	}
}

// parseCBZ reads a .cbz (zip) file and returns a sorted list of image pages.
func parseCBZ(filePath string) ([]*models.Page, []byte, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, nil, err
	}
	defer r.Close()

	var pages []*models.Page
	var imageFiles []*zip.File
	for _, f := range r.File {
		// Skip directories and non-image files
		if f.FileInfo().IsDir() || !IsImageFile(f.Name) {
			continue
		}
		pages = append(pages, &models.Page{FileName: f.Name})
		imageFiles = append(imageFiles, f)
	}

	// Sort pages alphabetically by filename to ensure correct order.
	sort.Slice(pages, func(i, j int) bool {
		return pages[i].FileName < pages[j].FileName
	})

	// Assign index after sorting
	for i := range pages {
		pages[i].Index = i
	}

	// If there are images, read the first one for the thumbnail.
	var firstPageData []byte
	if len(imageFiles) > 0 {
		rc, err := imageFiles[0].Open()
		if err != nil {
			return pages, nil, fmt.Errorf("failed to open first page for thumbnail: %w", err)
		}
		defer rc.Close()

		firstPageData, err = io.ReadAll(rc)
		if err != nil {
			return pages, nil, fmt.Errorf("failed to read first page for thumbnail: %w", err)
		}
	}

	return pages, firstPageData, nil
}

// parseCBR opens the given archive or directory path, finds all image files, picks the lexically first image file,
// and returns its bytes.
func parseCBR(path string) ([]*models.Page, []byte, error) {
	var pages []*models.Page
	// Create a virtual file system from the path (archive, dir, etc)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fsys, err := archives.FileSystem(ctx, path, nil)
	if err != nil {
		// return nil, "", fmt.Errorf("failed to open file system: %w", err)
		return []*models.Page{}, nil, err
	}

	var imageFiles []string

	// Walk the virtual FS to find image files
	err = fs.WalkDir(fsys, ".", func(fpath string, d fs.DirEntry, err error) error {
		if err != nil {
			// If error walking, stop
			return err
		}
		if d.IsDir() {
			return nil
		}
		if IsImageFile(d.Name()) {
			imageFiles = append(imageFiles, fpath)
			pages = append(pages, &models.Page{FileName: fpath})
		}
		return nil
	})
	if err != nil {
		return []*models.Page{}, nil, err
	}

	if len(imageFiles) == 0 {
		return []*models.Page{}, nil, err
	}

	// Sort image files lexically and pick the first
	sort.Strings(imageFiles)
	firstImage := imageFiles[0]

	// Open the first image file
	f, err := fsys.Open(firstImage)
	if err != nil {
		return []*models.Page{}, nil, err
	}
	defer f.Close()

	// Read all bytes from the first image file
	data, err := io.ReadAll(f)
	if err != nil {
		return []*models.Page{}, nil, err
	}

	// Sort pages alphabetically by filename to ensure correct order.
	sort.Slice(pages, func(i, j int) bool {
		return pages[i].FileName < pages[j].FileName
	})

	// Assign index after sorting
	for i := range pages {
		pages[i].Index = i
	}
	return pages, data, nil
}
