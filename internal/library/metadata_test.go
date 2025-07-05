// This file contains tests for the metadata extraction logic.

package library_test

import (
	"testing"

	"github.com/vrsandeep/mango-go/internal/library"
)

func TestExtractMetadataFromPath(t *testing.T) {
	// Use a table-driven test to cover multiple scenarios
	testCases := []struct {
		name            string
		filePath        string
		libraryPath     string
		expectedSeries  string
		expectedChapter string
	}{
		{
			name:            "Standard Case",
			filePath:        "/path/to/library/Manga Series/Chapter 01.cbz",
			libraryPath:     "/path/to/library",
			expectedSeries:  "Manga Series",
			expectedChapter: "Chapter 01.cbz",
		},
		{
			name:            "File in Library Root",
			filePath:        "/path/to/library/Root Chapter.cbz",
			libraryPath:     "/path/to/library",
			expectedSeries:  "Unknown Series",
			expectedChapter: "Root Chapter.cbz",
		},
		{
			name:            "Nested Directory",
			filePath:        "/path/to/library/Author Name/Series Name/Vol 01.cbz",
			libraryPath:     "/path/to/library/Author Name",
			expectedSeries:  "Series Name",
			expectedChapter: "Vol 01.cbz",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			series, chapter := library.ExtractMetadataFromPath(tc.filePath, tc.libraryPath)
			if series != tc.expectedSeries {
				t.Errorf("Expected series '%s', but got '%s'", tc.expectedSeries, series)
			}
			if chapter != tc.expectedChapter {
				t.Errorf("Expected chapter '%s', but got '%s'", tc.expectedChapter, chapter)
			}
		})
	}
}
