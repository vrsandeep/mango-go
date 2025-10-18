package api

import (
	"log"
	"net/http"
	"sync"

	"github.com/vrsandeep/mango-go/internal/models"
)

func (s *Server) handleGetHomePageData(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)
	if user == nil {
		RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var wg sync.WaitGroup
	var homeData models.HomePageData

	// Use a mutex to safely handle concurrent writes to the errors slice
	var mu sync.Mutex
	var errors []error

	// Fetch all four sections concurrently for maximum performance.
	wg.Add(4)

	go func() {
		defer wg.Done()
		data, err := s.homeStore.GetContinueReading(user.ID, 12)
		if err != nil {
			mu.Lock()
			errors = append(errors, err)
			mu.Unlock()
		}
		homeData.ContinueReading = data
	}()

	go func() {
		defer wg.Done()
		data, err := s.homeStore.GetNextUp(user.ID, 12)
		if err != nil {
			mu.Lock()
			errors = append(errors, err)
			mu.Unlock()
		}
		homeData.NextUp = data
	}()

	go func() {
		defer wg.Done()
		data, err := s.homeStore.GetRecentlyAdded(24) // Fetch more items to group
		if err != nil {
			mu.Lock()
			errors = append(errors, err)
			mu.Unlock()
		}
		homeData.RecentlyAdded = data
	}()

	go func() {
		defer wg.Done()
		data, err := s.homeStore.GetStartReading(user.ID, 12)
		if err != nil {
			mu.Lock()
			errors = append(errors, err)
			mu.Unlock()
		}
		homeData.StartReading = data
	}()

	wg.Wait()

	if len(errors) > 0 {
		for _, e := range errors {
			log.Printf("Error fetching home page data: %v", e)
		}
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve home page data")
		return
	}

	RespondWithJSON(w, http.StatusOK, homeData)
}
