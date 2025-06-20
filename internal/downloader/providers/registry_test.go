package providers

import (
	"github.com/vrsandeep/mango-go/internal/models"
	"testing"

	"github.com/vrsandeep/mango-go/internal/downloader/providers/mockadex"
)

// resetRegistry is a helper to ensure a clean state for each test run.
func resetRegistry() {
	registry = make(map[string]models.Provider)
}

func TestProviderRegistry(t *testing.T) {
	resetRegistry()
	Register(mockadex.New())

	t.Run("Get All Providers", func(t *testing.T) {
		all := GetAll()
		if len(all) != 1 {
			t.Fatalf("Expected 1 provider, got %d", len(all))
		}
		if all[0].ID != "mockadex" {
			t.Errorf("Expected provider ID 'mockadex', got '%s'", all[0].ID)
		}
	})

	t.Run("Get Existing Provider", func(t *testing.T) {
		p, ok := Get("mockadex")
		if !ok {
			t.Fatal("Expected to find provider 'mockadex', but it was not found")
		}
		if p.GetInfo().Name != "Mockadex" {
			t.Errorf("Expected provider name 'Mockadex', got '%s'", p.GetInfo().Name)
		}
	})

	t.Run("Get Non-existent Provider", func(t *testing.T) {
		_, ok := Get("nonexistent")
		if ok {
			t.Fatal("Expected not to find provider 'nonexistent', but it was found")
		}
	})

	t.Run("Panic on Duplicate Registration", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected registration of a duplicate provider to panic, but it did not")
			}
		}()
		// This should cause a panic
		Register(mockadex.New())
	})
}
