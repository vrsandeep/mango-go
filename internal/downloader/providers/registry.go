package providers

import (
	"fmt"

	"github.com/vrsandeep/mango-go/internal/models"
)

var registry = make(map[string]models.Provider)

// Register adds a new provider to the registry. It's called at startup.
func Register(p models.Provider) {
	info := p.GetInfo()
	if _, exists := registry[info.ID]; exists {
		// Panic is appropriate here as it's a developer error during setup.
		panic(fmt.Sprintf("provider with ID '%s' is already registered", info.ID))
	}
	registry[info.ID] = p
}

// Get returns a provider by its ID.
func Get(id string) (models.Provider, bool) {
	p, ok := registry[id]
	return p, ok
}

// GetAll returns a list of information for all registered providers.
func GetAll() []models.ProviderInfo {
	var providers []models.ProviderInfo
	for _, p := range registry {
		providers = append(providers, p.GetInfo())
	}
	return providers
}
