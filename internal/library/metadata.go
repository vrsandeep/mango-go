// This file handles the logic for extracting metadata from file paths.
// The current implementation is simple and uses directory and file names.

package library

import (
	"path/filepath"
)

// ExtractMetadataFromPath uses simple heuristics to determine the series title
// and chapter name from a file's path.
// For example: /path/to/library/One-Piece/Chapter-100.cbz
// Series Title: One-Piece
// Chapter Name: Chapter-100.cbz
func ExtractMetadataFromPath(filePath, libraryPath string) (seriesTitle, chapterName string) {
	// The chapter name is simply the file name
	chapterName = filepath.Base(filePath)

	// The series title is the name of the parent directory of the file
	dir := filepath.Dir(filePath)
	seriesTitle = filepath.Base(dir)

	// If the file is in the root of the library, the series might be unknown
	// or derived from the filename itself. This logic can be improved later.
	if dir == libraryPath {
		seriesTitle = "Unknown Series"
	}

	return seriesTitle, chapterName
}
