package plugins

import (
	"fmt"
	"log"

	"github.com/vrsandeep/mango-go/internal/models"
)

// LazyPluginProviderAdapter is a wrapper that loads a plugin on first access.
type LazyPluginProviderAdapter struct {
	manager  *PluginManager
	pluginID string
}

// NewLazyPluginProviderAdapter creates a new lazy adapter.
func NewLazyPluginProviderAdapter(manager *PluginManager, pluginID string) *LazyPluginProviderAdapter {
	return &LazyPluginProviderAdapter{
		manager:  manager,
		pluginID: pluginID,
	}
}

// ensureLoaded ensures the plugin is loaded and returns the adapter.
func (l *LazyPluginProviderAdapter) ensureLoaded() (*PluginProviderAdapter, error) {
	// Note: We don't cache the adapter here because plugins can be unloaded.
	// Always get a fresh adapter to ensure we have the latest loaded instance.
	adapter, err := l.manager.LoadPluginIfNeeded(l.pluginID)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin %s: %w", l.pluginID, err)
	}

	return adapter, nil
}

// GetInfo returns plugin information.
func (l *LazyPluginProviderAdapter) GetInfo() models.ProviderInfo {
	// Try to get info from discovered plugin first (no loading required)
	l.manager.mu.RLock()
	if discovered, exists := l.manager.discoveredPlugins[l.pluginID]; exists {
		manifest := discovered.Manifest
		l.manager.mu.RUnlock()

		name := manifest.Name
		if name == "" {
			name = manifest.ID
		}

		return models.ProviderInfo{
			ID:   manifest.ID,
			Name: name,
		}
	}
	l.manager.mu.RUnlock()

	// Fallback: load plugin and get info
	adapter, err := l.ensureLoaded()
	if err != nil {
		log.Printf("Error loading plugin %s for GetInfo: %v", l.pluginID, err)
		return models.ProviderInfo{
			ID:   l.pluginID,
			Name: l.pluginID,
		}
	}

	return adapter.GetInfo()
}

// Search searches for series.
func (l *LazyPluginProviderAdapter) Search(query string) ([]models.SearchResult, error) {
	adapter, err := l.ensureLoaded()
	if err != nil {
		return nil, err
	}
	return adapter.Search(query)
}

// GetChapters gets chapters for a series.
func (l *LazyPluginProviderAdapter) GetChapters(seriesIdentifier string) ([]models.ChapterResult, error) {
	adapter, err := l.ensureLoaded()
	if err != nil {
		return nil, err
	}
	return adapter.GetChapters(seriesIdentifier)
}

// GetPageURLs gets page URLs for a chapter.
func (l *LazyPluginProviderAdapter) GetPageURLs(chapterIdentifier string) ([]string, error) {
	adapter, err := l.ensureLoaded()
	if err != nil {
		return nil, err
	}
	return adapter.GetPageURLs(chapterIdentifier)
}
