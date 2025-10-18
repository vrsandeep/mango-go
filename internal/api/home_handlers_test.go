package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vrsandeep/mango-go/internal/api"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestHandleGetHomePageDataContinueReading(t *testing.T) {
	// Setup test server
	server, _, _ := testutil.SetupTestServer(t)
	router := server.Router()

	// Create mock store
	mockStore := &api.MockHomeStore{}
	server.SetHomeStore(mockStore)

	// Setup mock expectations
	expectedItems := []*models.HomeSectionItem{
		{
			SeriesID:       1,
			SeriesTitle:    "Series A",
			ChapterID:      func() *int64 { id := int64(1); return &id }(),
			ChapterTitle:   "Chapter 1",
			CoverArt:       "cover.jpg",
			ProgressPercent: func() *int { p := 50; return &p }(),
			Read:           func() *bool { r := false; return &r }(),
		},
	}
	mockStore.On("GetContinueReading", int64(1), 12).Return(expectedItems, nil)
	mockStore.On("GetNextUp", int64(1), 12).Return([]*models.HomeSectionItem{}, nil)
	mockStore.On("GetRecentlyAdded", 24).Return([]*models.HomeSectionItem{}, nil)
	mockStore.On("GetStartReading", int64(1), 12).Return([]*models.HomeSectionItem{}, nil)

	// Get authenticated cookie
	cookie := testutil.CookieForUser(t, server, "testuser_continue", "password", "user")

	// Make request
	req, _ := http.NewRequest("GET", "/api/home", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code)

	var data models.HomePageData
	err := json.Unmarshal(rr.Body.Bytes(), &data)
	assert.NoError(t, err)

	assert.Len(t, data.ContinueReading, 1)
	assert.Equal(t, "Series A", data.ContinueReading[0].SeriesTitle)
	assert.Equal(t, int64(1), *data.ContinueReading[0].ChapterID)
	assert.Equal(t, 50, *data.ContinueReading[0].ProgressPercent)

	// Verify all mock expectations were met
	mockStore.AssertExpectations(t)
}

func TestHandleGetHomePageDataNextUp(t *testing.T) {
	// Setup test server
	server, _, _ := testutil.SetupTestServer(t)
	router := server.Router()

	// Create mock store
	mockStore := &api.MockHomeStore{}
	server.SetHomeStore(mockStore)

	// Setup mock expectations
	expectedItems := []*models.HomeSectionItem{
		{
			SeriesID:     2,
			SeriesTitle:  "Series B",
			ChapterID:    func() *int64 { id := int64(2); return &id }(),
			ChapterTitle: "Chapter 2",
			CoverArt:     "cover2.jpg",
		},
	}
	mockStore.On("GetContinueReading", int64(1), 12).Return([]*models.HomeSectionItem{}, nil)
	mockStore.On("GetNextUp", int64(1), 12).Return(expectedItems, nil)
	mockStore.On("GetRecentlyAdded", 24).Return([]*models.HomeSectionItem{}, nil)
	mockStore.On("GetStartReading", int64(1), 12).Return([]*models.HomeSectionItem{}, nil)

	// Get authenticated cookie
	cookie := testutil.CookieForUser(t, server, "testuser_nextup", "password", "user")

	// Make request
	req, _ := http.NewRequest("GET", "/api/home", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code)

	var data models.HomePageData
	err := json.Unmarshal(rr.Body.Bytes(), &data)
	assert.NoError(t, err)

	assert.Len(t, data.NextUp, 1)
	assert.Equal(t, "Series B", data.NextUp[0].SeriesTitle)
	assert.Equal(t, int64(2), *data.NextUp[0].ChapterID)

	// Verify all mock expectations were met
	mockStore.AssertExpectations(t)
}

func TestHandleGetHomePageDataRecentlyAdded(t *testing.T) {
	// Setup test server
	server, _, _ := testutil.SetupTestServer(t)
	router := server.Router()

	// Create mock store
	mockStore := &api.MockHomeStore{}
	server.SetHomeStore(mockStore)

	// Setup mock expectations
	expectedItems := []*models.HomeSectionItem{
		{
			SeriesID:        1,
			SeriesTitle:     "Series A",
			ChapterID:       func() *int64 { id := int64(1); return &id }(),
			ChapterTitle:    "Chapter 1",
			CoverArt:        "cover.jpg",
			NewChapterCount: 2,
		},
		{
			SeriesID:        2,
			SeriesTitle:     "Series B",
			ChapterID:       func() *int64 { id := int64(2); return &id }(),
			ChapterTitle:    "Chapter 1",
			CoverArt:        "cover2.jpg",
			NewChapterCount: 1,
		},
	}
	mockStore.On("GetContinueReading", int64(1), 12).Return([]*models.HomeSectionItem{}, nil)
	mockStore.On("GetNextUp", int64(1), 12).Return([]*models.HomeSectionItem{}, nil)
	mockStore.On("GetRecentlyAdded", 24).Return(expectedItems, nil)
	mockStore.On("GetStartReading", int64(1), 12).Return([]*models.HomeSectionItem{}, nil)

	// Get authenticated cookie
	cookie := testutil.CookieForUser(t, server, "testuser_recent", "password", "user")

	// Make request
	req, _ := http.NewRequest("GET", "/api/home", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code)

	var data models.HomePageData
	err := json.Unmarshal(rr.Body.Bytes(), &data)
	assert.NoError(t, err)

	assert.Len(t, data.RecentlyAdded, 2)
	assert.Equal(t, "Series A", data.RecentlyAdded[0].SeriesTitle)
	assert.Equal(t, 2, data.RecentlyAdded[0].NewChapterCount)
	assert.Equal(t, "Series B", data.RecentlyAdded[1].SeriesTitle)
	assert.Equal(t, 1, data.RecentlyAdded[1].NewChapterCount)

	// Verify all mock expectations were met
	mockStore.AssertExpectations(t)
}

func TestHandleGetHomePageDataStartReading(t *testing.T) {
	// Setup test server
	server, _, _ := testutil.SetupTestServer(t)
	router := server.Router()

	// Create mock store
	mockStore := &api.MockHomeStore{}
	server.SetHomeStore(mockStore)

	// Setup mock expectations
	expectedItems := []*models.HomeSectionItem{
		{
			SeriesID:    3,
			SeriesTitle: "Series C",
			CoverArt:    "cover3.jpg",
		},
	}
	mockStore.On("GetContinueReading", int64(1), 12).Return([]*models.HomeSectionItem{}, nil)
	mockStore.On("GetNextUp", int64(1), 12).Return([]*models.HomeSectionItem{}, nil)
	mockStore.On("GetRecentlyAdded", 24).Return([]*models.HomeSectionItem{}, nil)
	mockStore.On("GetStartReading", int64(1), 12).Return(expectedItems, nil)

	// Get authenticated cookie
	cookie := testutil.CookieForUser(t, server, "testuser_start", "password", "user")

	// Make request
	req, _ := http.NewRequest("GET", "/api/home", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code)

	var data models.HomePageData
	err := json.Unmarshal(rr.Body.Bytes(), &data)
	assert.NoError(t, err)

	assert.Len(t, data.StartReading, 1)
	assert.Equal(t, "Series C", data.StartReading[0].SeriesTitle)

	// Verify all mock expectations were met
	mockStore.AssertExpectations(t)
}

func TestHandleGetHomePageDataEmptyLibrary(t *testing.T) {
	// Setup test server
	server, _, _ := testutil.SetupTestServer(t)
	router := server.Router()

	// Create mock store
	mockStore := &api.MockHomeStore{}
	server.SetHomeStore(mockStore)

	// Setup mock expectations - all return empty
	mockStore.On("GetContinueReading", int64(1), 12).Return([]*models.HomeSectionItem{}, nil)
	mockStore.On("GetNextUp", int64(1), 12).Return([]*models.HomeSectionItem{}, nil)
	mockStore.On("GetRecentlyAdded", 24).Return([]*models.HomeSectionItem{}, nil)
	mockStore.On("GetStartReading", int64(1), 12).Return([]*models.HomeSectionItem{}, nil)

	// Get authenticated cookie
	cookie := testutil.CookieForUser(t, server, "testuser_empty", "password", "user")

	// Make request
	req, _ := http.NewRequest("GET", "/api/home", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code)

	var data models.HomePageData
	err := json.Unmarshal(rr.Body.Bytes(), &data)
	assert.NoError(t, err)

	assert.Len(t, data.ContinueReading, 0)
	assert.Len(t, data.NextUp, 0)
	assert.Len(t, data.RecentlyAdded, 0)
	assert.Len(t, data.StartReading, 0)

	// Verify all mock expectations were met
	mockStore.AssertExpectations(t)
}

func TestHandleGetHomePageDataUnauthorized(t *testing.T) {
	// Setup test server
	server, _, _ := testutil.SetupTestServer(t)
	router := server.Router()

	// Make request without authentication
	req, _ := http.NewRequest("GET", "/api/home", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestHandleGetHomePageDataStoreError(t *testing.T) {
	// Setup test server
	server, _, _ := testutil.SetupTestServer(t)
	router := server.Router()

	// Create mock store
	mockStore := &api.MockHomeStore{}
	server.SetHomeStore(mockStore)

	// Setup mock expectations - one method returns error
	mockStore.On("GetContinueReading", int64(1), 12).Return([]*models.HomeSectionItem{}, nil)
	mockStore.On("GetNextUp", int64(1), 12).Return([]*models.HomeSectionItem{}, nil)
	mockStore.On("GetRecentlyAdded", 24).Return([]*models.HomeSectionItem{}, assert.AnError)
	mockStore.On("GetStartReading", int64(1), 12).Return([]*models.HomeSectionItem{}, nil)

	// Get authenticated cookie
	cookie := testutil.CookieForUser(t, server, "testuser_error", "password", "user")

	// Make request
	req, _ := http.NewRequest("GET", "/api/home", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	// Verify all mock expectations were met
	mockStore.AssertExpectations(t)
}

func TestHandleGetHomePageDataCompleteFlow(t *testing.T) {
	// Setup test server
	server, _, _ := testutil.SetupTestServer(t)
	router := server.Router()

	// Create mock store
	mockStore := &api.MockHomeStore{}
	server.SetHomeStore(mockStore)

	// Setup mock expectations with realistic data
	continueReading := []*models.HomeSectionItem{
		{
			SeriesID:       1,
			SeriesTitle:    "Series A",
			ChapterID:      func() *int64 { id := int64(1); return &id }(),
			ChapterTitle:   "Chapter 1",
			CoverArt:       "cover.jpg",
			ProgressPercent: func() *int { p := 50; return &p }(),
		},
	}

	nextUp := []*models.HomeSectionItem{
		{
			SeriesID:     1,
			SeriesTitle:  "Series A",
			ChapterID:    func() *int64 { id := int64(2); return &id }(),
			ChapterTitle: "Chapter 2",
			CoverArt:     "cover.jpg",
		},
	}

	recentlyAdded := []*models.HomeSectionItem{
		{
			SeriesID:        2,
			SeriesTitle:     "Series B",
			ChapterID:       func() *int64 { id := int64(3); return &id }(),
			ChapterTitle:    "Chapter 1",
			CoverArt:        "cover2.jpg",
			NewChapterCount: 1,
		},
	}

	startReading := []*models.HomeSectionItem{
		{
			SeriesID:    3,
			SeriesTitle: "Series C",
			CoverArt:    "cover3.jpg",
		},
	}

	mockStore.On("GetContinueReading", int64(1), 12).Return(continueReading, nil)
	mockStore.On("GetNextUp", int64(1), 12).Return(nextUp, nil)
	mockStore.On("GetRecentlyAdded", 24).Return(recentlyAdded, nil)
	mockStore.On("GetStartReading", int64(1), 12).Return(startReading, nil)

	// Get authenticated cookie
	cookie := testutil.CookieForUser(t, server, "testuser_complete", "password", "user")

	// Make request
	req, _ := http.NewRequest("GET", "/api/home", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code)

	var data models.HomePageData
	err := json.Unmarshal(rr.Body.Bytes(), &data)
	assert.NoError(t, err)

	// Verify all sections have data
	assert.Len(t, data.ContinueReading, 1)
	assert.Len(t, data.NextUp, 1)
	assert.Len(t, data.RecentlyAdded, 1)
	assert.Len(t, data.StartReading, 1)

	// Verify specific data
	assert.Equal(t, "Series A", data.ContinueReading[0].SeriesTitle)
	assert.Equal(t, "Series A", data.NextUp[0].SeriesTitle)
	assert.Equal(t, "Series B", data.RecentlyAdded[0].SeriesTitle)
	assert.Equal(t, "Series C", data.StartReading[0].SeriesTitle)

	// Verify all mock expectations were met
	mockStore.AssertExpectations(t)
}