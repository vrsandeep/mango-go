package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateFolderPath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "library")

	// Create the base directory
	if err := os.MkdirAll(basePath, 0755); err != nil {
		t.Fatalf("Failed to create base directory: %v", err)
	}

	tests := []struct {
		name        string
		folderPath  string
		basePath    string
		expectError bool
		setup       func() // Optional setup function
		cleanup     func() // Optional cleanup function
	}{
		{
			name:        "valid existing directory",
			folderPath:  "existing",
			basePath:    basePath,
			expectError: false,
			setup: func() {
				os.MkdirAll(filepath.Join(basePath, "existing"), 0755)
			},
		},
		{
			name:        "valid non-existing directory (can be created)",
			folderPath:  "new_folder",
			basePath:    basePath,
			expectError: false,
		},
		{
			name:        "valid nested directory",
			folderPath:  "nested/deep/folder",
			basePath:    basePath,
			expectError: false,
		},
		{
			name:        "empty folder path",
			folderPath:  "",
			basePath:    basePath,
			expectError: true,
		},
		{
			name:        "directory traversal attempt",
			folderPath:  "../../etc/passwd",
			basePath:    basePath,
			expectError: true,
		},
		{
			name:        "directory traversal with dots",
			folderPath:  "folder/../other",
			basePath:    basePath,
			expectError: true,
		},
		{
			name:        "absolute path (should work)",
			folderPath:  filepath.Join(basePath, "absolute_test"),
			basePath:    basePath,
			expectError: false,
		},
		{
			name:        "path exists but is a file",
			folderPath:  "file_path",
			basePath:    basePath,
			expectError: true,
			setup: func() {
				file, _ := os.Create(filepath.Join(basePath, "file_path"))
				file.Close()
			},
		},
		{
			name:        "path with invalid characters",
			folderPath:  "folder<with>invalid:chars",
			basePath:    basePath,
			expectError: false, // Should be sanitized and work
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup if provided
			if tt.setup != nil {
				tt.setup()
			}

			// Cleanup if provided
			if tt.cleanup != nil {
				defer tt.cleanup()
			}

			err := ValidateFolderPath(tt.folderPath, tt.basePath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestSanitizeFolderPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal path",
			input:    "folder/subfolder",
			expected: "folder/subfolder",
		},
		{
			name:     "path with backslashes",
			input:    "folder\\subfolder",
			expected: "folder/subfolder",
		},
		{
			name:     "path with leading slashes",
			input:    "/folder/subfolder",
			expected: "folder/subfolder",
		},
		{
			name:     "path with trailing slashes",
			input:    "folder/subfolder/",
			expected: "folder/subfolder",
		},
		{
			name:     "path with multiple slashes",
			input:    "folder//subfolder",
			expected: "folder/subfolder",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only slashes",
			input:    "///",
			expected: "",
		},
		{
			name:     "mixed separators",
			input:    "folder\\subfolder/another",
			expected: "folder/subfolder/another",
		},
		{
			name:     "path with dots",
			input:    "folder/./subfolder",
			expected: "folder/subfolder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFolderPath(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFolderPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateFolderPathPermissions(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "library")

	// Create the base directory
	if err := os.MkdirAll(basePath, 0755); err != nil {
		t.Fatalf("Failed to create base directory: %v", err)
	}

	// Test with read-only directory
	readOnlyDir := filepath.Join(basePath, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0444); err != nil {
		t.Fatalf("Failed to create read-only directory: %v", err)
	}

	// Test that we can't write to read-only directory
	err := ValidateFolderPath("readonly", basePath)
	if err == nil {
		t.Error("Expected error for read-only directory, but got none")
	}

	// Test that we can create a directory in a writable location
	err = ValidateFolderPath("writable", basePath)
	if err != nil {
		t.Errorf("Expected no error for writable directory, but got: %v", err)
	}
}

func TestValidateFolderPathNonExistentParent(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "library")

	// Test creating a directory with non-existent parent
	err := ValidateFolderPath("deep/nested/folder", basePath)
	if err != nil {
		t.Errorf("Expected no error for nested directory creation, but got: %v", err)
	}

	// Note: The validation function creates and then removes the directory
	// as part of the validation process, so we don't expect it to exist
	// The test verifies that the validation passes, which means the path
	// can be created successfully
}
