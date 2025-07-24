package db_test

import (
	"testing"

	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestForeignKeyCascadeDelete(t *testing.T) {
	// Setup test database with migrations already applied
	db := testutil.SetupTestDB(t)

	// Test 1: Verify foreign keys are enabled
	var foreignKeysEnabled int
	err := db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeysEnabled)
	if err != nil {
		t.Fatalf("Failed to check foreign keys status: %v", err)
	}
	if foreignKeysEnabled != 1 {
		t.Errorf("Foreign keys should be enabled, got: %d", foreignKeysEnabled)
	}

	// Test 2: Create test data and verify cascade delete works
	// Create a user
	_, err = db.Exec("INSERT INTO users (username, password_hash, role, created_at) VALUES (?, ?, ?, datetime('now'))",
		"testuser", "hash", "user")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create a folder
	_, err = db.Exec("INSERT INTO folders (name, path, created_at, updated_at) VALUES (?, ?, datetime('now'), datetime('now'))",
		"Test Folder", "/test/path")
	if err != nil {
		t.Fatalf("Failed to create test folder: %v", err)
	}

	// Create a chapter
	_, err = db.Exec("INSERT INTO chapters (folder_id, path, page_count, created_at, updated_at) VALUES (?, ?, ?, datetime('now'), datetime('now'))",
		1, "/test/chapter", 10)
	if err != nil {
		t.Fatalf("Failed to create test chapter: %v", err)
	}

	// Create user progress
	_, err = db.Exec("INSERT INTO user_chapter_progress (user_id, chapter_id, progress_percent, read, updated_at) VALUES (?, ?, ?, ?, datetime('now'))",
		1, 1, 50, false)
	if err != nil {
		t.Fatalf("Failed to create test user progress: %v", err)
	}

	// Create user folder settings
	_, err = db.Exec("INSERT INTO user_folder_settings (user_id, folder_id, sort_by, sort_dir) VALUES (?, ?, ?, ?)",
		1, 1, "auto", "asc")
	if err != nil {
		t.Fatalf("Failed to create test user folder settings: %v", err)
	}

	// Test 3: Delete user and verify cascade delete
	_, err = db.Exec("DELETE FROM users WHERE id = 1")
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	// Verify user_chapter_progress was deleted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM user_chapter_progress WHERE user_id = 1").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check user_chapter_progress: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 records in user_chapter_progress after user deletion, got %d", count)
	}

	// Verify user_folder_settings was deleted
	err = db.QueryRow("SELECT COUNT(*) FROM user_folder_settings WHERE user_id = 1").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check user_folder_settings: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 records in user_folder_settings after user deletion, got %d", count)
	}

	// Test 4: Delete folder and verify cascade delete
	_, err = db.Exec("DELETE FROM folders WHERE id = 1")
	if err != nil {
		t.Fatalf("Failed to delete folder: %v", err)
	}

	// Verify chapters were deleted
	err = db.QueryRow("SELECT COUNT(*) FROM chapters WHERE folder_id = 1").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check chapters: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 records in chapters after folder deletion, got %d", count)
	}

	// Verify user_folder_settings was deleted
	err = db.QueryRow("SELECT COUNT(*) FROM user_folder_settings WHERE folder_id = 1").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check user_folder_settings: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 records in user_folder_settings after folder deletion, got %d", count)
	}
}
