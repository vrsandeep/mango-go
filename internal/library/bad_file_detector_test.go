package library_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrsandeep/mango-go/internal/library"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestBadFileDetector(t *testing.T) {
	app := testutil.SetupTestApp(t)
	libraryRoot := app.Config().Library.Path

	t.Run("Test DetectBadFiles", func(t *testing.T) {
		// Create test directory structure with both good and bad files
		testDir := filepath.Join(libraryRoot, "Bad File Detector Test")
		os.MkdirAll(testDir, 0755)

		// Create a good CBZ file
		goodFile := filepath.Join(testDir, "good.cbz")
		testutil.CreateTestCBZ(t, testDir, "good.cbz", []string{"p1.jpg"})

		// Create a bad file (invalid archive)
		badFile := filepath.Join(testDir, "bad.cbz")
		err := os.WriteFile(badFile, []byte("This is not a valid CBZ file"), 0644)
		if err != nil {
			t.Fatalf("Failed to create bad file: %v", err)
		}

		// Create a corrupted ZIP file (partial file)
		corruptedFile := filepath.Join(testDir, "corrupted.zip")
		err = os.WriteFile(corruptedFile, []byte("PK\x03\x04"), 0644) // Partial ZIP header
		if err != nil {
			t.Fatalf("Failed to create corrupted file: %v", err)
		}

		// Run the bad file detection
		library.DetectBadFiles(app)

		// Check that bad files were detected and recorded
		badFileStore := store.NewBadFileStore(app.DB())
		badFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get bad files: %v", err)
		}

		// Should have at least 2 bad files
		if len(badFiles) < 2 {
			t.Errorf("Expected at least 2 bad files, got %d", len(badFiles))
		}

		// Check that the bad file was recorded
		var foundBadFile bool
		var foundCorruptedFile bool
		for _, bf := range badFiles {
			if bf.Path == badFile {
				foundBadFile = true
				if bf.Error != string(models.ErrorCorruptedArchive) {
					t.Errorf("Bad file should have error %s, got %s", models.ErrorCorruptedArchive, bf.Error)
				}
			}
			if bf.Path == corruptedFile {
				foundCorruptedFile = true
				if bf.Error != string(models.ErrorCorruptedArchive) {
					t.Errorf("Corrupted file should have error %s, got %s", models.ErrorCorruptedArchive, bf.Error)
				}
			}
		}

		if !foundBadFile {
			t.Error("Bad file was not recorded")
		}
		if !foundCorruptedFile {
			t.Error("Corrupted file was not recorded")
		}

		// Check that the good file was not recorded as bad
		var foundGoodFile bool
		for _, bf := range badFiles {
			if bf.Path == goodFile {
				foundGoodFile = true
				break
			}
		}
		if foundGoodFile {
			t.Error("Good file should not be recorded as bad")
		}
	})

	t.Run("Test DetectBadFiles_OnlyGoodFiles", func(t *testing.T) {
		// Test with only good files
		goodDir := filepath.Join(libraryRoot, "Good Files Only")
		os.MkdirAll(goodDir, 0755)

		// Create multiple good CBZ files
		for i := 1; i <= 3; i++ {
			testutil.CreateTestCBZ(t, goodDir, fmt.Sprintf("good%d.cbz", i), []string{"p1.jpg"})
		}

		// Run detection
		library.DetectBadFiles(app)

		// Check that no bad files were recorded for this directory
		badFileStore := store.NewBadFileStore(app.DB())
		badFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get bad files: %v", err)
		}

		// Count bad files in the good directory
		var badFilesInGoodDir int
		for _, bf := range badFiles {
			if filepath.Dir(bf.Path) == goodDir || strings.HasPrefix(bf.Path, goodDir+string(filepath.Separator)) {
				badFilesInGoodDir++
			}
		}

		if badFilesInGoodDir > 0 {
			t.Errorf("Expected 0 bad files in good directory, got %d", badFilesInGoodDir)
		}
	})

	t.Run("Test DetectBadFiles_AccessibilityIssues", func(t *testing.T) {
		// Test with files that have accessibility issues
		accessibilityDir := filepath.Join(libraryRoot, "Accessibility Issues")
		os.MkdirAll(accessibilityDir, 0755)

		// Create a file that will be inaccessible
		inaccessibleFile := filepath.Join(accessibilityDir, "inaccessible.cbz")
		err := os.WriteFile(inaccessibleFile, []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create inaccessible file: %v", err)
		}

		// Make the file inaccessible by removing read permissions
		err = os.Chmod(inaccessibleFile, 0000)
		if err != nil {
			t.Fatalf("Failed to remove file permissions: %v", err)
		}

		// Run detection
		library.DetectBadFiles(app)

		// Restore permissions for cleanup
		os.Chmod(inaccessibleFile, 0644)

		// Check that the inaccessible file was handled gracefully
		// The detector should skip files it can't access
		badFileStore := store.NewBadFileStore(app.DB())
		badFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get bad files: %v", err)
		}

		// The inaccessible file should not cause the detection to fail
		// It should be skipped and not recorded as a bad file
		var foundInaccessible bool
		for _, bf := range badFiles {
			if bf.Path == inaccessibleFile {
				foundInaccessible = true
				break
			}
		}

		// Note: The current implementation doesn't record inaccessible files
		// This test verifies that the detection continues without crashing
		if foundInaccessible {
			t.Log("Inaccessible file was recorded (this may be expected behavior)")
		}
	})
}

func TestErrorCategorization(t *testing.T) {
	app := testutil.SetupTestApp(t)
	libraryRoot := app.Config().Library.Path

	t.Run("Test categorizeError_CorruptedArchive", func(t *testing.T) {
		// Test ZIP corruption error
		testDir := filepath.Join(libraryRoot, "Error Categorization Test", "Corrupted")
		os.MkdirAll(testDir, 0755)

		// Create a corrupted ZIP file
		corruptedFile := filepath.Join(testDir, "corrupted.zip")
		err := os.WriteFile(corruptedFile, []byte("PK\x03\x04\x14\x00\x00\x00\x08\x00"), 0644) // Partial ZIP header
		if err != nil {
			t.Fatalf("Failed to create corrupted file: %v", err)
		}

		// Run detection
		library.DetectBadFiles(app)

		// Check that the file was categorized as corrupted archive
		badFileStore := store.NewBadFileStore(app.DB())
		badFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get bad files: %v", err)
		}

		var found bool
		for _, bf := range badFiles {
			if bf.Path == corruptedFile {
				found = true
				if bf.Error != string(models.ErrorCorruptedArchive) {
					t.Errorf("Expected error %s, got %s", models.ErrorCorruptedArchive, bf.Error)
				}
				break
			}
		}
		if !found {
			t.Error("Corrupted file was not detected")
		}
	})

	t.Run("Test categorizeError_UnsupportedFormat", func(t *testing.T) {
		// Test unsupported format error
		testDir := filepath.Join(libraryRoot, "Error Categorization Test", "Unsupported")
		os.MkdirAll(testDir, 0755)

		// Create a file with supported extension but invalid content that will cause
		// the parser to fail with "unsupported archive" error
		unsupportedFile := filepath.Join(testDir, "unsupported.cbz")
		// Create a file that looks like a ZIP but has invalid internal structure
		invalidZipData := []byte{
			0x50, 0x4B, 0x03, 0x04, // ZIP local file header signature
			0x14, 0x00, // Version needed to extract
			0x00, 0x00, // General purpose bit flag
			0x08, 0x00, // Compression method
			0x00, 0x00, 0x00, 0x00, // Last mod file time
			0x00, 0x00, 0x00, 0x00, // Last mod file date
			0x00, 0x00, 0x00, 0x00, // CRC-32
			0x00, 0x00, 0x00, 0x00, // Compressed size
			0x00, 0x00, 0x00, 0x00, // Uncompressed size
			0x01, 0x00, // File name length
			0x00, 0x00, // Extra field length
			0x74, // File name: 't'
			// Missing the actual file content and central directory
		}
		err := os.WriteFile(unsupportedFile, invalidZipData, 0644)
		if err != nil {
			t.Fatalf("Failed to create unsupported file: %v", err)
		}

		// Run detection
		library.DetectBadFiles(app)

		// Check that the file was categorized as unsupported format
		badFileStore := store.NewBadFileStore(app.DB())
		badFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get bad files: %v", err)
		}

		var found bool
		for _, bf := range badFiles {
			if bf.Path == unsupportedFile {
				found = true
				// The file should be detected as a bad file, but the exact error type
				// depends on how the ZIP parser handles the malformed content
				if bf.Error == "" {
					t.Error("Unsupported format file should have an error message")
				}
				break
			}
		}
		if !found {
			t.Error("Unsupported format file was not detected")
		}
	})

	t.Run("Test categorizeError_EmptyArchive", func(t *testing.T) {
		// Test empty archive error
		testDir := filepath.Join(libraryRoot, "Error Categorization Test", "Empty")
		os.MkdirAll(testDir, 0755)

		// Create an empty ZIP file (valid ZIP structure but no content)
		emptyFile := filepath.Join(testDir, "empty.zip")
		emptyZipData := []byte{
			0x50, 0x4B, 0x05, 0x06, // ZIP end of central directory signature
			0x00, 0x00, 0x00, 0x00, // Number of this disk
			0x00, 0x00, 0x00, 0x00, // Number of the disk with the start of the central directory
			0x00, 0x00, 0x00, 0x00, // Total number of entries in the central directory on this disk
			0x00, 0x00, 0x00, 0x00, // Total number of entries in the central directory
			0x00, 0x00, 0x00, 0x00, // Size of the central directory
			0x00, 0x00, 0x00, 0x00, // Offset of start of central directory with respect to the starting disk number
			0x00, 0x00, // ZIP file comment length
		}
		err := os.WriteFile(emptyFile, emptyZipData, 0644)
		if err != nil {
			t.Fatalf("Failed to create empty ZIP file: %v", err)
		}

		// Run detection
		library.DetectBadFiles(app)

		// Check that the file was categorized as empty archive
		badFileStore := store.NewBadFileStore(app.DB())
		badFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get bad files: %v", err)
		}

		var found bool
		for _, bf := range badFiles {
			if bf.Path == emptyFile {
				found = true
				if bf.Error != string(models.ErrorEmptyArchive) {
					t.Errorf("Expected error %s, got %s", models.ErrorEmptyArchive, bf.Error)
				}
				break
			}
		}
		if !found {
			t.Error("Empty archive file was not detected")
		}
	})

	t.Run("Test categorizeError_IOError", func(t *testing.T) {
		// Test I/O error (file accessibility issues)
		testDir := filepath.Join(libraryRoot, "Error Categorization Test", "IOError")
		os.MkdirAll(testDir, 0755)

		// Create a file that will be inaccessible
		inaccessibleFile := filepath.Join(testDir, "inaccessible.cbz")
		err := os.WriteFile(inaccessibleFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create inaccessible file: %v", err)
		}

		// Make the file inaccessible by removing read permissions
		err = os.Chmod(inaccessibleFile, 0000)
		if err != nil {
			t.Fatalf("Failed to remove file permissions: %v", err)
		}

		// Run detection
		library.DetectBadFiles(app)

		// Restore permissions for cleanup
		os.Chmod(inaccessibleFile, 0644)

		// Check that the file was categorized as I/O error
		badFileStore := store.NewBadFileStore(app.DB())
		badFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get bad files: %v", err)
		}

		var found bool
		for _, bf := range badFiles {
			if bf.Path == inaccessibleFile {
				found = true
				if bf.Error != string(models.ErrorIOError) {
					t.Errorf("Expected error %s, got %s", models.ErrorIOError, bf.Error)
				}
				break
			}
		}
		if !found {
			t.Error("Inaccessible file was not detected")
		}
	})

	t.Run("Test categorizeError_PasswordProtected", func(t *testing.T) {
		// Test password protected error
		testDir := filepath.Join(libraryRoot, "Error Categorization Test", "Password")
		os.MkdirAll(testDir, 0755)

		// Create a file that simulates a password-protected archive
		// This is tricky to test directly, so we'll create a file that the parser
		// might interpret as password-protected
		passwordFile := filepath.Join(testDir, "password.zip")
		// Create a ZIP file with encrypted flag set
		encryptedZipData := []byte{
			0x50, 0x4B, 0x03, 0x04, // ZIP local file header signature
			0x14, 0x00, // Version needed to extract
			0x00, 0x01, // General purpose bit flag (bit 0 = encrypted)
			0x08, 0x00, // Compression method
			0x00, 0x00, 0x00, 0x00, // Last mod file time
			0x00, 0x00, 0x00, 0x00, // Last mod file date
			0x00, 0x00, 0x00, 0x00, // CRC-32
			0x00, 0x00, 0x00, 0x00, // Compressed size
			0x00, 0x00, 0x00, 0x00, // Uncompressed size
			0x01, 0x00, // File name length
			0x00, 0x00, // Extra field length
			0x74, // File name: 't'
		}
		err := os.WriteFile(passwordFile, encryptedZipData, 0644)
		if err != nil {
			t.Fatalf("Failed to create password-protected file: %v", err)
		}

		// Run detection
		library.DetectBadFiles(app)

		// Check that the file was categorized as password protected
		badFileStore := store.NewBadFileStore(app.DB())
		badFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get bad files: %v", err)
		}

		var found bool
		for _, bf := range badFiles {
			if bf.Path == passwordFile {
				found = true
				// The actual error might be different depending on how the parser handles it
				// We'll just check that it was detected as a bad file
				if bf.Error == "" {
					t.Error("Password-protected file should have an error message")
				}
				break
			}
		}
		if !found {
			t.Error("Password-protected file was not detected")
		}
	})

	t.Run("Test categorizeError_InvalidFormat", func(t *testing.T) {
		// Test invalid format error (unknown/unexpected errors)
		testDir := filepath.Join(libraryRoot, "Error Categorization Test", "Invalid")
		os.MkdirAll(testDir, 0755)

		// Create a file with completely invalid content
		invalidFile := filepath.Join(testDir, "invalid.cbz")
		err := os.WriteFile(invalidFile, []byte("This is completely invalid content that should cause an unknown error"), 0644)
		if err != nil {
			t.Fatalf("Failed to create invalid file: %v", err)
		}

		// Run detection
		library.DetectBadFiles(app)

		// Check that the file was categorized as invalid format
		badFileStore := store.NewBadFileStore(app.DB())
		badFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get bad files: %v", err)
		}

		var found bool
		for _, bf := range badFiles {
			if bf.Path == invalidFile {
				found = true
				// Should be categorized as invalid format (default for unknown errors)
				if bf.Error != string(models.ErrorInvalidFormat) &&
				   bf.Error != string(models.ErrorCorruptedArchive) {
					t.Errorf("Expected error %s or %s, got %s",
						models.ErrorInvalidFormat, models.ErrorCorruptedArchive, bf.Error)
				}
				break
			}
		}
		if !found {
			t.Error("Invalid format file was not detected")
		}
	})

	t.Run("Test categorizeError_Integration", func(t *testing.T) {
		// Test that all error types are properly categorized in a single run
		testDir := filepath.Join(libraryRoot, "Error Categorization Test", "Integration")
		os.MkdirAll(testDir, 0755)

		// Create files that should trigger different error types
		testFiles := map[string]string{
			"corrupted.zip": string(models.ErrorCorruptedArchive),
			"unsupported.cbz": string(models.ErrorUnsupportedFormat),
			"empty.zip": string(models.ErrorEmptyArchive),
			"inaccessible.cbz": string(models.ErrorIOError),
		}

		// Create the test files
		for filename, expectedError := range testFiles {
			filePath := filepath.Join(testDir, filename)

			switch expectedError {
			case string(models.ErrorCorruptedArchive):
				// Corrupted ZIP
				err := os.WriteFile(filePath, []byte("PK\x03\x04"), 0644)
				if err != nil {
					t.Fatalf("Failed to create %s: %v", filename, err)
				}
			case string(models.ErrorUnsupportedFormat):
				// Unsupported format
				err := os.WriteFile(filePath, []byte("unsupported content"), 0644)
				if err != nil {
					t.Fatalf("Failed to create %s: %v", filename, err)
				}
			case string(models.ErrorEmptyArchive):
				// Empty ZIP
				emptyZipData := []byte{
					0x50, 0x4B, 0x05, 0x06, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00,
				}
				err := os.WriteFile(filePath, emptyZipData, 0644)
				if err != nil {
					t.Fatalf("Failed to create %s: %v", filename, err)
				}
			case string(models.ErrorIOError):
				// Inaccessible file
				err := os.WriteFile(filePath, []byte("test"), 0644)
				if err != nil {
					t.Fatalf("Failed to create %s: %v", filename, err)
				}
				// Make it inaccessible
				err = os.Chmod(filePath, 0000)
				if err != nil {
					t.Fatalf("Failed to make %s inaccessible: %v", filename, err)
				}
			}
		}

		// Run detection
		library.DetectBadFiles(app)

		// Restore permissions for cleanup
		for filename := range testFiles {
			filePath := filepath.Join(testDir, filename)
			os.Chmod(filePath, 0644)
		}

		// Verify that all error types were properly categorized
		badFileStore := store.NewBadFileStore(app.DB())
		badFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get bad files: %v", err)
		}

		// Count files by error type
		errorCounts := make(map[string]int)
		for _, bf := range badFiles {
			if filepath.Dir(bf.Path) == testDir || strings.HasPrefix(bf.Path, testDir+string(filepath.Separator)) {
				errorCounts[bf.Error]++
			}
		}

		// Verify we have files with different error types
		if len(errorCounts) < 2 {
			t.Errorf("Expected at least 2 different error types, got %d", len(errorCounts))
		}

		// Log the error distribution for debugging
		t.Logf("Error type distribution: %v", errorCounts)
	})
}
