// This file is responsible for parsing archive files like .cbz (ZIP) and
// .cbr (RAR) to get a list of the image files they contain.

package library

import (
	"archive/zip"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrsandeep/mango-go/internal/models"
)

// isImageFile checks if a filename has a common image file extension.
func isImageFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp"
}

// ParseArchive dispatches to the correct parser based on file extension.
func ParseArchive(filePath string) ([]*models.Page, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".cbz":
		return parseCBZ(filePath)
	case ".cbr":
		return parseCBR(filePath)
	default:
		return nil, fmt.Errorf("unsupported archive type: %s", ext)
	}
}

// parseCBZ reads a .cbz (zip) file and returns a sorted list of image pages.
func parseCBZ(filePath string) ([]*models.Page, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var pages []*models.Page
	for _, f := range r.File {
		// Skip directories and non-image files
		if f.FileInfo().IsDir() || !isImageFile(f.Name) {
			continue
		}
		pages = append(pages, &models.Page{FileName: f.Name})
	}

	// Sort pages alphabetically by filename to ensure correct order.
	sort.Slice(pages, func(i, j int) bool {
		return pages[i].FileName < pages[j].FileName
	})

	// Assign index after sorting
	for i := range pages {
		pages[i].Index = i
	}

	return pages, nil
}

// parseCBR is a placeholder for RAR file parsing.
//
// **Proof of Concept Research Note:**
// Implementing this will require a CGo binding to a C library like `libunarr`
// or finding a pure Go RAR library. A popular CGo choice is `github.com/gen2brain/go-unarr`.
// This would be a key task for a future milestone.
func parseCBR(filePath string) ([]*models.Page, error) {
	log.Printf("Parsing CBR files is not yet implemented. File: %s", filePath)
	// Return an empty list for now so the scanner can continue.
	return []*models.Page{}, nil
}
