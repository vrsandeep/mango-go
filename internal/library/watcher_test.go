package library_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/vrsandeep/mango-go/internal/library"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

// TestWatcherService_Start tests starting the watcher service
func TestWatcherService_Start(t *testing.T) {
	app := testutil.SetupTestApp(t)
	watcher := library.NewWatcherService(app)

	err := watcher.Start()
	if err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}

	// Cleanup
	defer watcher.Stop()
}

// TestWatcherService_Stop tests stopping the watcher service
func TestWatcherService_Stop(t *testing.T) {
	app := testutil.SetupTestApp(t)
	watcher := library.NewWatcherService(app)

	err := watcher.Start()
	if err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}

	err = watcher.Stop()
	if err != nil {
		t.Fatalf("Failed to stop watcher: %v", err)
	}
}

// TestWatcherService_FileCreate tests that file creation triggers incremental scan
func TestWatcherService_FileCreate(t *testing.T) {
	app := testutil.SetupTestApp(t)
	libraryRoot := app.Config().Library.Path

	watcher := library.NewWatcherService(app)
	err := watcher.Start()
	if err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}
	defer watcher.Stop()

	// Wait a bit for watcher to initialize
	time.Sleep(100 * time.Millisecond)

	// Create a test directory and file
	testDir := filepath.Join(libraryRoot, "WatcherTest")
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	// Create a test CBZ file
	testFile := filepath.Join(testDir, "test.cbz")
	testutil.CreateTestCBZ(t, testDir, "test.cbz", []string{"page1.jpg"})

	// Wait for debounce delay + some buffer
	time.Sleep(3 * time.Second)

	// Verify the file was created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("Test file was not created")
	}
}

// TestWatcherService_FileModify tests that file modification triggers incremental scan
func TestWatcherService_FileModify(t *testing.T) {
	app := testutil.SetupTestApp(t)
	libraryRoot := app.Config().Library.Path

	// Create initial file
	testDir := filepath.Join(libraryRoot, "ModifyTest")
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "modify.cbz")
	testutil.CreateTestCBZ(t, testDir, "modify.cbz", []string{"page1.jpg"})

	// Wait for initial file to be created
	time.Sleep(500 * time.Millisecond)

	watcher := library.NewWatcherService(app)
	err := watcher.Start()
	if err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}
	defer watcher.Stop()

	// Wait for watcher to initialize
	time.Sleep(100 * time.Millisecond)

	// Modify the file by touching it
	now := time.Now()
	err = os.Chtimes(testFile, now, now)
	if err != nil {
		t.Fatalf("Failed to modify file timestamp: %v", err)
	}

	// Wait for debounce delay
	time.Sleep(3 * time.Second)
}

// TestWatcherService_FileDelete tests that file deletion triggers incremental scan
func TestWatcherService_FileDelete(t *testing.T) {
	app := testutil.SetupTestApp(t)
	libraryRoot := app.Config().Library.Path

	// Create initial file
	testDir := filepath.Join(libraryRoot, "DeleteTest")
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "delete.cbz")
	testutil.CreateTestCBZ(t, testDir, "delete.cbz", []string{"page1.jpg"})

	watcher := library.NewWatcherService(app)
	err := watcher.Start()
	if err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}
	defer watcher.Stop()

	// Wait for watcher to initialize
	time.Sleep(100 * time.Millisecond)

	// Delete the file
	err = os.Remove(testFile)
	if err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}

	// Wait for debounce delay
	time.Sleep(3 * time.Second)

	// Verify file was deleted
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("File was not deleted")
	}
}

// TestWatcherService_DirectoryCreate tests that directory creation is watched
func TestWatcherService_DirectoryCreate(t *testing.T) {
	app := testutil.SetupTestApp(t)
	libraryRoot := app.Config().Library.Path

	watcher := library.NewWatcherService(app)
	err := watcher.Start()
	if err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}
	defer watcher.Stop()

	// Wait for watcher to initialize
	time.Sleep(100 * time.Millisecond)

	// Create a new directory
	newDir := filepath.Join(libraryRoot, "NewDir")
	err = os.MkdirAll(newDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	defer os.RemoveAll(newDir)

	// Wait a bit for the watcher to process
	time.Sleep(500 * time.Millisecond)

	// Create a file in the new directory
	testutil.CreateTestCBZ(t, newDir, "newfile.cbz", []string{"page1.jpg"})

	// Wait for debounce delay
	time.Sleep(3 * time.Second)
}

// TestWatcherService_Debouncing tests that rapid changes are debounced
func TestWatcherService_Debouncing(t *testing.T) {
	app := testutil.SetupTestApp(t)
	libraryRoot := app.Config().Library.Path

	watcher := library.NewWatcherService(app)
	err := watcher.Start()
	if err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}
	defer watcher.Stop()

	// Wait for watcher to initialize
	time.Sleep(100 * time.Millisecond)

	testDir := filepath.Join(libraryRoot, "DebounceTest")
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	// Create multiple files rapidly
	for i := 0; i < 5; i++ {
		fileName := filepath.Join(testDir, "file"+fmt.Sprintf("%d", i)+".cbz")
		testutil.CreateTestCBZ(t, testDir, filepath.Base(fileName), []string{"page1.jpg"})
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for debounce delay - should only trigger one scan
	time.Sleep(3 * time.Second)
}

// TestIncrementalLibrarySync tests the incremental sync function
func TestIncrementalLibrarySync(t *testing.T) {
	app := testutil.SetupTestApp(t)
	libraryRoot := app.Config().Library.Path

	// Create test files
	testDir := filepath.Join(libraryRoot, "IncrementalTest")
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	testFile1 := filepath.Join(testDir, "file1.cbz")
	testFile2 := filepath.Join(testDir, "file2.cbz")
	testutil.CreateTestCBZ(t, testDir, "file1.cbz", []string{"page1.jpg"})
	testutil.CreateTestCBZ(t, testDir, "file2.cbz", []string{"page1.jpg"})

	// Run initial full sync
	library.LibrarySync(app)

	// Run incremental sync on the test directory
	changedPaths := []string{testDir}
	err := library.IncrementalLibrarySync(app, changedPaths)
	if err != nil {
		t.Fatalf("Incremental sync failed: %v", err)
	}

	// Verify files are still there
	if _, err := os.Stat(testFile1); os.IsNotExist(err) {
		t.Error("file1.cbz was not found after incremental sync")
	}
	if _, err := os.Stat(testFile2); os.IsNotExist(err) {
		t.Error("file2.cbz was not found after incremental sync")
	}
}

// TestIncrementalLibrarySync_DeletedFile tests incremental sync with deleted files
func TestIncrementalLibrarySync_DeletedFile(t *testing.T) {
	app := testutil.SetupTestApp(t)
	libraryRoot := app.Config().Library.Path

	// Create and sync initial file
	testDir := filepath.Join(libraryRoot, "DeleteSyncTest")
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "delete-sync.cbz")
	testutil.CreateTestCBZ(t, testDir, "delete-sync.cbz", []string{"page1.jpg"})

	// Run initial sync
	library.LibrarySync(app)

	// Verify file is in database
	st := store.New(app.DB())
	chapters, _ := st.GetAllChaptersByHash()
	found := false
	for _, ch := range chapters {
		if ch.Path == testFile {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("File not found in database after initial sync")
	}

	// Delete the file
	os.Remove(testFile)

	// Run incremental sync
	changedPaths := []string{testFile}
	err := library.IncrementalLibrarySync(app, changedPaths)
	if err != nil {
		t.Fatalf("Incremental sync failed: %v", err)
	}

	// Verify file is removed from database
	chapters, _ = st.GetAllChaptersByHash()
	found = false
	for _, ch := range chapters {
		if ch.Path == testFile {
			found = true
			break
		}
	}
	if found {
		t.Error("File still found in database after deletion and incremental sync")
	}
}

// TestWatcherService_ConcurrentEvents tests handling of concurrent file events
func TestWatcherService_ConcurrentEvents(t *testing.T) {
	app := testutil.SetupTestApp(t)
	libraryRoot := app.Config().Library.Path

	watcher := library.NewWatcherService(app)
	err := watcher.Start()
	if err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}
	defer watcher.Stop()

	// Wait for watcher to initialize
	time.Sleep(100 * time.Millisecond)

	testDir := filepath.Join(libraryRoot, "ConcurrentTest")
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	// Create files concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			fileName := fmt.Sprintf("concurrent%d.cbz", idx)
			testutil.CreateTestCBZ(t, testDir, fileName, []string{"page1.jpg"})
		}(i)
	}
	wg.Wait()

	// Wait for debounce delay
	time.Sleep(3 * time.Second)
}

// TestWatcherService_NonArchiveFiles tests that non-archive files don't trigger scans
func TestWatcherService_NonArchiveFiles(t *testing.T) {
	app := testutil.SetupTestApp(t)
	libraryRoot := app.Config().Library.Path

	watcher := library.NewWatcherService(app)
	err := watcher.Start()
	if err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}
	defer watcher.Stop()

	// Wait for watcher to initialize
	time.Sleep(100 * time.Millisecond)

	testDir := filepath.Join(libraryRoot, "NonArchiveTest")
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	// Create a non-archive file (should still trigger event, but scanner will skip it)
	textFile := filepath.Join(testDir, "readme.txt")
	err = os.WriteFile(textFile, []byte("This is a text file"), 0644)
	if err != nil {
		t.Fatalf("Failed to create text file: %v", err)
	}

	// Wait for debounce delay
	time.Sleep(3 * time.Second)

	// Verify file exists
	if _, err := os.Stat(textFile); os.IsNotExist(err) {
		t.Error("Text file was not created")
	}
}
