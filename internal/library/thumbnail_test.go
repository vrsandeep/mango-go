package library_test

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/vrsandeep/mango-go/internal/library"
)

func TestGenerateThumbnail(t *testing.T) {
	// A valid 1x1 PNG, base64 encoded.
	validPngB64 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII="
	pngData, _ := base64.StdEncoding.DecodeString(validPngB64)

	t.Run("Success case", func(t *testing.T) {
		thumb, err := library.GenerateThumbnail(pngData)
		if err != nil {
			t.Fatalf("GenerateThumbnail failed with valid data: %v", err)
		}
		if !strings.HasPrefix(thumb, "data:image/jpeg;base64,") {
			t.Errorf("Generated thumbnail is not a valid data URI, got: %s", thumb)
		}
		if len(thumb) < 50 {
			t.Errorf("Generated thumbnail seems too short: %s", thumb)
		}
	})

	t.Run("Error case with invalid data", func(t *testing.T) {
		invalidData := []byte("this is not an image")
		_, err := library.GenerateThumbnail(invalidData)
		if err == nil {
			t.Error("GenerateThumbnail should have failed with invalid data, but it did not")
		}
	})
}
