// Bad-file records for chapter files that fail library inspection.

package models

import "time"

// BadFile represents a corrupted or invalid chapter file on disk.
type BadFile struct {
	ID          int64     `json:"id"`
	Path        string    `json:"path"`
	FileName    string    `json:"file_name"`
	Error       string    `json:"error"`
	FileSize    int64     `json:"file_size"`
	DetectedAt  time.Time `json:"detected_at"`
	LastChecked time.Time `json:"last_checked"`
}

// BadFileError represents different types of file errors
type BadFileError string

const (
	// ErrorCorruptedChapterFile means the file could not be parsed (e.g. invalid zip).
	ErrorCorruptedChapterFile BadFileError = "corrupted_archive"
	// ErrorCorruptedArchive is an alias for ErrorCorruptedChapterFile (same stored value).
	ErrorCorruptedArchive = ErrorCorruptedChapterFile

	ErrorInvalidFormat     BadFileError = "invalid_format"
	ErrorPasswordProtected BadFileError = "password_protected"

	// ErrorEmptyChapterFile means there were no readable pages (e.g. no images in archive).
	ErrorEmptyChapterFile BadFileError = "empty_archive"
	// ErrorEmptyArchive is an alias for ErrorEmptyChapterFile (same stored value).
	ErrorEmptyArchive = ErrorEmptyChapterFile

	ErrorUnsupportedFormat BadFileError = "unsupported_format"
	ErrorIOError           BadFileError = "io_error"
)

// String returns a short human-readable label for admin UI and exports.
func (e BadFileError) String() string {
	switch e {
	case ErrorCorruptedChapterFile:
		return "Corrupt file"
	case ErrorInvalidFormat:
		return "Invalid format"
	case ErrorPasswordProtected:
		return "Password protected"
	case ErrorEmptyChapterFile:
		return "No readable pages"
	case ErrorUnsupportedFormat:
		return "Unsupported format"
	case ErrorIOError:
		return "I/O error"
	default:
		return "Unknown error"
	}
}
