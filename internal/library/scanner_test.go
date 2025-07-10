// This file tests the main library scanner.

package library_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/library"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

// // https://gist.github.com/ondrek/7413434
// const (
// 	tinyPNG_A = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII="             // Transparent
// 	tinyPNG_B = "iVBORw0KGgoAAAANSUhEUgAAAAoAAAAKCAYAAACNMs+9AAAAFUlEQVR42mP8z8BQz0AEYBxVSF+FABJADveWkH6oAAAAAElFTkSuQmCC" // Red
// 	tinyPNG_C = "iVBORw0KGgoAAAANSUhEUgAAAAoAAAAKCAYAAACNMs+9AAAAFUlEQVR42mNk+M9Qz0AEYBxVSF+FAAhKDveksOjmAAAAAElFTkSuQmCC" // Green
// 	tinyPNG_D = "iVBORw0KGgoAAAANSUhEUgAAAAoAAAAKCAYAAACNMs+9AAAAFUlEQVR42mNkYPhfz0AEYBxVSF+FAP5FDvcfRYWgAAAAAElFTkSuQmCC" // Blue
// 	tinyPNG_E = "iVBORw0KGgoAAAANSUhEUgAAAAoAAAAKCAYAAACNMs+9AAAAFUlEQVR42mP8/5+hnoEIwDiqkL4KAcT9GO0U4BxoAAAAAElFTkSuQmCC" // Yellow
// )

// // setupTestLibraryAndDB creates a temporary library structure and an in-memory DB.
// func setupTestLibraryAndDB(t *testing.T) (string, *sql.DB) {
// 	t.Helper()
// 	// Setup DB
// 	db := testutil.SetupTestDB(t)

// 	// Setup Library
// 	rootDir := t.TempDir()
// 	seriesADir := filepath.Join(rootDir, "Series A")
// 	os.Mkdir(seriesADir, 0755)
// 	// Create two chapters. The scanner processes files in alphabetical order,
// 	// so "ch1.cbz" will be scanned first. Its thumbnail should be used for the series.
// 	testutil.CreateTestCBZWithThumbnail(t, seriesADir, "ch1.cbz", []string{"pageA1.jpg"}, tinyPNG_A)
// 	testutil.CreateTestCBZWithThumbnail(t, seriesADir, "ch2.cbz", []string{"pageB1.jpg"}, tinyPNG_B)

// 	return rootDir, db
// }

func TestScannerIntegration(t *testing.T) {
	libraryPath, db := testutil.SetupTestLibraryAndDB(t)
	defer db.Close()

	// Configure and run the scanner
	cfg := &config.Config{Library: struct {
		Path string `mapstructure:"path"`
	}{Path: libraryPath}}
	scanner := library.NewScanner(cfg, db)
	if err := scanner.Scan(nil, nil); err != nil {
		t.Fatalf("scanner.Scan() failed: %v", err)
	}

	var seriesThumbnail string
	err := db.QueryRow("SELECT thumbnail FROM series WHERE title = 'Series A'").Scan(&seriesThumbnail)
	if err != nil {
		t.Fatalf("Failed to query series thumbnail: %v", err)
	}

	if !strings.HasPrefix(seriesThumbnail, "data:image/jpeg;base64,") {
		t.Error("Series thumbnail is not a valid data URI")
	}

	// Query both chapters to check their thumbnails
	rows, err := db.Query("SELECT path, thumbnail FROM chapters WHERE series_id = (SELECT id FROM series WHERE title = 'Series A') ORDER BY path")
	if err != nil {
		t.Fatalf("Failed to query chapters: %v", err)
	}
	defer rows.Close()

	chapterThumbnails := make(map[string]string)
	for rows.Next() {
		var path, thumbnail string
		if err := rows.Scan(&path, &thumbnail); err != nil {
			t.Fatalf("Failed to scan chapter row: %v", err)
		}
		chapterThumbnails[filepath.Base(path)] = thumbnail
	}

	if len(chapterThumbnails) != 2 {
		t.Fatalf("Expected to find 2 chapters, but found %d", len(chapterThumbnails))
	}

	ch1Thumb, ok := chapterThumbnails["ch1.cbz"]
	if !ok {
		t.Fatal("Chapter 'ch1.cbz' not found in database")
	}
	if !strings.HasPrefix(ch1Thumb, "data:image/jpeg;base64,") {
		t.Error("Thumbnail for 'ch1.cbz' is not a valid data URI")
	}

	ch2Thumb, ok := chapterThumbnails["ch2.cbz"]
	if !ok {
		t.Fatal("Chapter 'ch2.cbz' not found in database")
	}
	if !strings.HasPrefix(ch2Thumb, "data:image/jpeg;base64,") {
		t.Error("Thumbnail for 'ch2.cbz' is not a valid data URI")
	}

	// IMPORTANT: Verify that the series thumbnail matches the thumbnail of the FIRST chapter.
	if seriesThumbnail != ch1Thumb {
		t.Error("Series thumbnail does not match the thumbnail of the first chapter ('ch1.cbz')")
	}

	// Also verify it does NOT match the second chapter's thumbnail, proving it wasn't overwritten.
	if seriesThumbnail == ch2Thumb {
		t.Error("Series thumbnail incorrectly matches the thumbnail of the second chapter ('ch2.cbz')")
	}

	// Verify the database state after the scan
	s := store.New(db)
	tx, _ := db.Begin()
	defer tx.Rollback()

	// Check if "Series A" was created
	seriesID, err := s.GetOrCreateSeries(tx, "Series A", filepath.Join(libraryPath, "Series A"))
	if err != nil {
		t.Fatalf("Failed to find 'Series A' in database after scan: %v", err)
	}
	if seriesID == 0 {
		t.Fatal("Series ID should not be 0")
	}

	// Check if the chapter was created correctly
	var pageCount int
	var chapterPath string
	err = tx.QueryRow("SELECT path, page_count FROM chapters WHERE series_id = ?", seriesID).Scan(&chapterPath, &pageCount)
	if err != nil {
		t.Fatalf("Failed to find chapter for 'Series A': %v", err)
	}
	if pageCount != 1 {
		t.Errorf("Expected page count of 1 for scanned chapter, got %d", pageCount)
	}
	expectedPath := filepath.Join(libraryPath, "Series A", "ch1.cbz")
	if chapterPath != expectedPath {
		t.Errorf("Expected chapter path '%s', got '%s'", expectedPath, chapterPath)
	}
}

func TestLibrarySync(t *testing.T) {
	app := testutil.SetupTestApp(t) // Sets up in-memory DB, config, etc.
	st := store.New(app.DB)
	libraryRoot := app.Config.Library.Path

	// --- Test 1: Initial Scan ---
	t.Run("Initial Scan", func(t *testing.T) {
		// Create mock file structure
		os.MkdirAll(filepath.Join(libraryRoot, "Series A", "Volume 1"), 0755)
		testutil.CreateTestCBZ(t, filepath.Join(libraryRoot, "Series A", "Volume 1"), "ch1.cbz", []string{"p1.jpg"})

		library.LibrarySync(app)

		// Verify folder structure
		folders, _ := st.GetAllFoldersByPath()
		if len(folders) != 2 {
			t.Fatalf("Expected 2 folders, got %d", len(folders))
		}
		if _, ok := folders[filepath.Join(libraryRoot, "Series A")]; !ok {
			t.Error("Series A folder not created")
		}

		// Verify chapter
		chapters, _ := st.GetAllChaptersByHash()
		if len(chapters) != 1 {
			t.Fatalf("Expected 1 chapter, got %d", len(chapters))
		}
	})

	// --- Test 2: Moved File ---
	t.Run("Moved File Detection", func(t *testing.T) {
		// Move the file
		os.MkdirAll(filepath.Join(libraryRoot, "Series B"), 0755)
		oldPath := filepath.Join(libraryRoot, "Series A", "Volume 1", "ch1.cbz")
		newPath := filepath.Join(libraryRoot, "Series B", "ch1-moved.cbz")
		os.Rename(oldPath, newPath)

		library.LibrarySync(app)

		chapters, _ := st.GetAllChaptersByHash()
		if len(chapters) != 1 {
			t.Fatalf("Expected 1 chapter after move, got %d", len(chapters))
		}

		var found bool
		for _, ch := range chapters {
			if ch.Path == newPath {
				found = true
			}
		}
		if !found {
			t.Error("Chapter path was not updated after move")
		}
	})

	// --- Test 3: Pruning ---
	t.Run("Pruning Deleted File", func(t *testing.T) {
		os.Remove(filepath.Join(libraryRoot, "Series B", "ch1-moved.cbz"))

		library.LibrarySync(app)

		chapters, _ := st.GetAllChaptersByHash()
		if len(chapters) != 0 {
			t.Errorf("Expected 0 chapters after pruning, got %d", len(chapters))
		}
	})
}
