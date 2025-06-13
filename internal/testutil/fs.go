package testutil

import (
	"archive/zip"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

const tinyPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII="

// createTestCBZ is a helper function that creates a temporary CBZ file
// for testing purposes. It returns the path to the created file.
func CreateTestCBZFile(t *testing.T, dir, name string) string {
	files := []string{"01.jpg", "03.png", "02.jpeg"}
	return CreateTestCBZ(t, dir, name, files)
}

// CreateTestCBZ is a helper function that creates a temporary CBZ file with
// a given set of page names and uses tinyPNG as image data.
func CreateTestCBZ(t *testing.T, dir, name string, pages []string) string {
	return CreateTestCBZWithThumbnail(t, dir, name, pages, tinyPNG)
}

func CreateTestCBZWithThumbnail(t *testing.T, dir, name string, pages []string, b64ImageData string) string {
	t.Helper()

	// Decode the provided image data.
	imageData, err := base64.StdEncoding.DecodeString(b64ImageData)
	if err != nil {
		t.Fatalf("Failed to decode image data: %v", err)
	}

	filePath := filepath.Join(dir, name)
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create temp cbz file: %v", err)
	}
	t.Cleanup(func() { file.Close() })

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	for _, page := range pages {
		writer, err := zipWriter.Create(page)
		if err != nil {
			t.Fatalf("Failed to create entry '%s' in zip: %v", page, err)
		}
		// Write the actual image data to the zip entry.
		_, err = writer.Write(imageData)
		if err != nil {
			t.Fatalf("Failed to write image data to zip entry: %v", err)
		}
	}
	return filePath
}
