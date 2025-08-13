// This file defines the data structure for tracking bad/corrupted archive files.

package models

import "time"

// BadFile represents a corrupted or invalid archive file in the library.
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
	ErrorCorruptedArchive BadFileError = "corrupted_archive"
	ErrorInvalidFormat    BadFileError = "invalid_format"
	ErrorPasswordProtected BadFileError = "password_protected"
	ErrorEmptyArchive     BadFileError = "empty_archive"
	ErrorUnsupportedFormat BadFileError = "unsupported_format"
	ErrorIOError          BadFileError = "io_error"
)

// String returns the human-readable error description
func (e BadFileError) String() string {
	switch e {
	case ErrorCorruptedArchive:
		return "Corrupted Archive"
	case ErrorInvalidFormat:
		return "Invalid Format"
	case ErrorPasswordProtected:
		return "Password Protected"
	case ErrorEmptyArchive:
		return "Empty Archive"
	case ErrorUnsupportedFormat:
		return "Unsupported Format"
	case ErrorIOError:
		return "I/O Error"
	default:
		return "Unknown Error"
	}
}
