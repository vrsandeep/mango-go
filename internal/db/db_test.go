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

	// Create a series
	_, err = db.Exec("INSERT INTO series (title, path, created_at, updated_at) VALUES (?, ?, datetime('now'), datetime('now'))",
		"Test Series", "/test/path")
	if err != nil {
		t.Fatalf("Failed to create test series: %v", err)
	}

	// Create a chapter
	_, err = db.Exec("INSERT INTO chapters (series_id, path, page_count, created_at, updated_at) VALUES (?, ?, ?, datetime('now'), datetime('now'))",
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

	// Create user series settings
	_, err = db.Exec("INSERT INTO user_series_settings (user_id, series_id, sort_by, sort_dir) VALUES (?, ?, ?, ?)",
		1, 1, "auto", "asc")
	if err != nil {
		t.Fatalf("Failed to create test user series settings: %v", err)
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

	// Verify user_series_settings was deleted
	err = db.QueryRow("SELECT COUNT(*) FROM user_series_settings WHERE user_id = 1").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check user_series_settings: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 records in user_series_settings after user deletion, got %d", count)
	}

	// Test 4: Delete series and verify cascade delete
	_, err = db.Exec("DELETE FROM series WHERE id = 1")
	if err != nil {
		t.Fatalf("Failed to delete series: %v", err)
	}

	// Verify chapters were deleted
	err = db.QueryRow("SELECT COUNT(*) FROM chapters WHERE series_id = 1").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check chapters: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 records in chapters after series deletion, got %d", count)
	}

	// Verify user_series_settings was deleted
	err = db.QueryRow("SELECT COUNT(*) FROM user_series_settings WHERE series_id = 1").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check user_series_settings: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 records in user_series_settings after series deletion, got %d", count)
	}
}
