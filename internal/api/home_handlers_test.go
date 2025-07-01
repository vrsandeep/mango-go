package api

// import (
// 	"encoding/json"
// 	"net/http"
// 	"net/http/httptest"
// 	"testing"
// 	"time"

// 	"github.com/vrsandeep/mango-go/internal/models"
// )

// func TestHandleGetHomePageData(t *testing.T) {
// 	server, _ := setupTestServer(t)
// 	router := server.Router()

// 	req, _ := http.NewRequest("GET", "/api/home", nil)
// 	req.AddCookie(CookieForUser(t, server, "testuser", "password", "user"))
// 	rr := httptest.NewRecorder()

// 	// Setup a complex DB state
// 	user, _ := server.store.GetUserByUsername("testuser")

// 	// Series 1: Continue Reading
// 	server.db.Exec(`INSERT INTO series (id, title, path, created_at, updated_at) VALUES (1, 'Series A', '/a', ?, ?)`, time.Now().Add(-5*time.Hour), time.Now().Add(-5*time.Hour))
// 	server.db.Exec(`INSERT INTO chapters (id, series_id, path, page_count, created_at, updated_at) VALUES (1, 1, 'A-1', 2, ?, ?)`, time.Now().Add(-5*time.Hour), time.Now().Add(-5*time.Hour))
// 	server.db.Exec("INSERT INTO user_chapter_progress (user_id, chapter_id, progress_percent, read, updated_at) VALUES (?, 1, 50, 0, ?)", user.ID, time.Now())
// 	// Series 2: Next Up (Chapter 3 is read, 4 is next)
// 	server.db.Exec(`INSERT INTO series (id, title, path, created_at, updated_at) VALUES (2, 'Series B', '/b', ?, ?)`, time.Now().Add(-4*time.Hour), time.Now().Add(-4*time.Hour))
// 	server.db.Exec(`INSERT INTO chapters (id, series_id, path, page_count, created_at, updated_at) VALUES (2, 2, 'B-1', 2, ?, ?)`, time.Now().Add(-4*time.Hour), time.Now().Add(-4*time.Hour))
// 	server.db.Exec(`INSERT INTO chapters (id, series_id, path, page_count, created_at, updated_at) VALUES (3, 2, 'B-2', 2, ?, ?)`, time.Now().Add(-4*time.Hour), time.Now().Add(-4*time.Hour))
// 	server.db.Exec("INSERT INTO user_chapter_progress (user_id, chapter_id, progress_percent, read, updated_at) VALUES (?, 3, 100, 1, ?)", user.ID, time.Now().Add(-1*time.Hour))
// 	// Series 3: Recently Added (Chapter 5)
// 	server.db.Exec(`INSERT INTO series (id, title, path, created_at, updated_at) VALUES (3, 'Series C', '/c', ?, ?)`, time.Now().Add(-3*time.Hour), time.Now().Add(-3*time.Hour))
// 	server.db.Exec(`INSERT INTO chapters (id, series_id, path, page_count, created_at, updated_at) VALUES (5, 3, 'C-2', 2, ?, ?)`, time.Now(), time.Now())
// 	// Series 4: Start Reading (no progress for user)
// 	server.db.Exec(`INSERT INTO series (id, title, path, created_at, updated_at) VALUES (4, 'Series D', '/d', ?, ?)`, time.Now().Add(-2*time.Hour), time.Now().Add(-2*time.Hour))
// 	server.db.Exec(`INSERT INTO chapters (id, series_id, path, page_count, created_at, updated_at) VALUES (4, 4, 'D-1', 2, ?, ?)`, time.Now().Add(-2*time.Hour), time.Now().Add(-2*time.Hour))
// 	// Series 5: Fully read by user, should not appear in Continue or Next Up
// 	server.db.Exec(`INSERT INTO series (id, title, path, created_at, updated_at) VALUES (5, 'Series E', '/e', ?, ?)`, time.Now().Add(-1*time.Hour), time.Now().Add(-1*time.Hour))
// 	server.db.Exec(`INSERT INTO chapters (id, series_id, path, page_count, created_at, updated_at) VALUES (6, 5, 'E-1', 2, ?, ?)`, time.Now().Add(-1*time.Hour), time.Now().Add(-1*time.Hour))
// 	server.db.Exec("INSERT INTO user_chapter_progress (user_id, chapter_id, progress_percent, read) VALUES (?, 6, 100, 1)", user.ID)

// 	router.ServeHTTP(rr, req)

// 	if status := rr.Code; status != http.StatusOK {
// 		t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
// 	}

// 	var data models.HomePageData
// 	if err := json.Unmarshal(rr.Body.Bytes(), &data); err != nil {
// 		t.Fatalf("Failed to unmarshal response: %v", err)
// 	}

// 	if len(data.ContinueReading) != 1 || *data.ContinueReading[0].ChapterID != 1 {
// 		t.Errorf("Expected 1 item in Continue Reading (chapter 1), got %d", len(data.ContinueReading))
// 	}
// 	if len(data.NextUp) < 1 || *data.NextUp[0].ChapterID != 4 {
// 		t.Errorf("Expected Next Up item to be chapter 4, got %d items", len(data.NextUp))
// 	}
// 	if len(data.RecentlyAdded) < 1 {
// 		t.Error("Expected items in Recently Added")
// 	}
// 	if len(data.StartReading) < 1 {
// 		t.Error("Expected items in Start Reading")
// 	}
// }
