package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateFolderPath checks if a folder path is valid and accessible.
// If the path doesn't exist, it checks if it can be created.
// Returns an error if the path is invalid or cannot be accessed/created.
func ValidateFolderPath(folderPath string, basePath string) error {
	if folderPath == "" {
		return fmt.Errorf("folder path cannot be empty")
	}

	// Clean the path to remove any directory traversal attempts
	cleanPath := filepath.Clean(folderPath)

	// Check for directory traversal attempts
	if strings.Contains(folderPath, "..") {
		return fmt.Errorf("folder path contains invalid directory traversal")
	}

	// If the path is absolute, validate it directly
	if filepath.IsAbs(cleanPath) {
		return validateAbsolutePath(cleanPath)
	}

	// For relative paths, join with base path
	fullPath := filepath.Join(basePath, cleanPath)
	return validateAbsolutePath(fullPath)
}

// validateAbsolutePath validates an absolute path
func validateAbsolutePath(fullPath string) error {
	// Check if the path already exists
	info, err := os.Stat(fullPath)
	if err == nil {
		// Path exists, check if it's a directory
		if !info.IsDir() {
			return fmt.Errorf("path exists but is not a directory: %s", fullPath)
		}
		// Check if we have write permissions
		if err := checkWritePermission(fullPath); err != nil {
			return fmt.Errorf("no write permission for existing directory: %w", err)
		}
		return nil
	}

	// Path doesn't exist, check if we can create it
	if os.IsNotExist(err) {
		return checkCanCreatePath(fullPath)
	}

	// Other error (permission denied, etc.)
	return fmt.Errorf("cannot access path: %w", err)
}

// checkWritePermission checks if we have write permission to a directory
func checkWritePermission(dirPath string) error {
	// Try to create a temporary file in the directory
	tempFile := filepath.Join(dirPath, ".mango_temp_check")
	file, err := os.Create(tempFile)
	if err != nil {
		return err
	}
	file.Close()

	// Clean up the temporary file
	os.Remove(tempFile)
	return nil
}

// checkCanCreatePath checks if we can create a directory at the given path
func checkCanCreatePath(fullPath string) error {
	// Get the parent directory
	parentDir := filepath.Dir(fullPath)

	// Check if parent directory exists and is writable
	if info, err := os.Stat(parentDir); err != nil {
		if os.IsNotExist(err) {
			// Parent doesn't exist, try to create it recursively
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("cannot create parent directory: %w", err)
			}
		} else {
			return fmt.Errorf("cannot access parent directory: %w", err)
		}
	} else if !info.IsDir() {
		return fmt.Errorf("parent path exists but is not a directory: %s", parentDir)
	}

	// Check write permission on parent directory
	if err := checkWritePermission(parentDir); err != nil {
		return fmt.Errorf("no write permission for parent directory: %w", err)
	}

	// Try to create the directory
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return fmt.Errorf("cannot create directory: %w", err)
	}

	// Clean up the test directory
	os.RemoveAll(fullPath)
	return nil
}

// SanitizeFolderPath cleans and sanitizes a folder path
func SanitizeFolderPath(folderPath string) string {
	if folderPath == "" {
		return ""
	}

	// Clean the path
	cleanPath := filepath.Clean(folderPath)

	// Remove any leading/trailing slashes
	cleanPath = strings.Trim(cleanPath, "/\\")

	// Replace backslashes with forward slashes for consistency
	cleanPath = strings.ReplaceAll(cleanPath, "\\", "/")

	return cleanPath
}
