package plugins

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrsandeep/mango-go/internal/core"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/store"
)

// RepositoryService handles plugin repository operations
type RepositoryService struct {
	app      *core.App
	store    *store.Store
	manager  PluginManagerInterface
	client   *http.Client
}

// NewRepositoryService creates a new repository service
func NewRepositoryService(app *core.App, storeInstance *store.Store, manager PluginManagerInterface) *RepositoryService {
	return &RepositoryService{
		app:     app,
		store:   storeInstance,
		manager: manager,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchRepository fetches and parses a repository manifest from a URL
func (rs *RepositoryService) FetchRepository(url string) (*models.RepositoryManifest, error) {
	resp, err := rs.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("repository returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read repository data: %w", err)
	}

	var manifest models.RepositoryManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse repository JSON: %w", err)
	}

	return &manifest, nil
}

// GetAvailablePlugins fetches and returns available plugins from a repository
func (rs *RepositoryService) GetAvailablePlugins(repositoryID int64) ([]models.RepositoryPlugin, error) {
	repo, err := rs.store.GetRepositoryByID(repositoryID)
	if err != nil {
		return nil, fmt.Errorf("repository not found: %w", err)
	}

	manifest, err := rs.FetchRepository(repo.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repository manifest: %w", err)
	}

	// Filter plugins by API version compatibility
	compatiblePlugins := make([]models.RepositoryPlugin, 0)
	for _, plugin := range manifest.Plugins {
		if err := ValidateAPIVersion(plugin.APIVersion); err == nil {
			compatiblePlugins = append(compatiblePlugins, plugin)
		}
	}

	return compatiblePlugins, nil
}

// InstallPlugin installs a plugin from a repository
func (rs *RepositoryService) InstallPlugin(pluginID string, repositoryID int64) error {
	// Get repository
	repo, err := rs.store.GetRepositoryByID(repositoryID)
	if err != nil {
		return fmt.Errorf("repository not found: %w", err)
	}

	// Fetch repository manifest
	manifest, err := rs.FetchRepository(repo.URL)
	if err != nil {
		return fmt.Errorf("failed to fetch repository: %w", err)
	}

	// Find the plugin
	var plugin models.RepositoryPlugin
	found := false
	for _, p := range manifest.Plugins {
		if p.ID == pluginID {
			plugin = p
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("plugin %s not found in repository", pluginID)
	}

	// Validate API version compatibility
	if err := ValidateAPIVersion(plugin.APIVersion); err != nil {
		return fmt.Errorf("plugin API version incompatible: %w", err)
	}

	// Create plugin directory
	pluginDir := filepath.Join(rs.app.Config().Plugins.Path, pluginID)
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// Download plugin.json
	manifestData, err := rs.downloadFile(plugin.ManifestURL)
	if err != nil {
		return fmt.Errorf("failed to download plugin.json: %w", err)
	}

	manifestPath := filepath.Join(pluginDir, "plugin.json")
	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		return fmt.Errorf("failed to write plugin.json: %w", err)
	}

	// Parse manifest to get entry point
	var pluginManifest struct {
		EntryPoint string `json:"entry_point"`
	}
	if err := json.Unmarshal(manifestData, &pluginManifest); err != nil {
		return fmt.Errorf("failed to parse plugin.json: %w", err)
	}

	if pluginManifest.EntryPoint == "" {
		pluginManifest.EntryPoint = "index.js"
	}

	// Download entry point file
	entryPointURL := strings.TrimSuffix(plugin.DownloadURL, "/") + "/" + pluginManifest.EntryPoint
	entryPointData, err := rs.downloadFile(entryPointURL)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", pluginManifest.EntryPoint, err)
	}

	entryPointPath := filepath.Join(pluginDir, pluginManifest.EntryPoint)
	if err := os.WriteFile(entryPointPath, entryPointData, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", pluginManifest.EntryPoint, err)
	}

	// Download any additional files referenced in the manifest (if needed in future)
	// For now, we only download plugin.json and entry_point file

	// Load the plugin
	if err := rs.manager.LoadPlugin(pluginDir); err != nil {
		// Cleanup on failure
		os.RemoveAll(pluginDir)
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	// Track installation
	var repoID sql.NullInt64
	if repositoryID > 0 {
		repoID = sql.NullInt64{Int64: repositoryID, Valid: true}
	}
	if err := rs.store.CreateOrUpdateInstalledPlugin(pluginID, repoID, plugin.Version); err != nil {
		// Log error but don't fail installation
		fmt.Printf("Warning: failed to track plugin installation: %v\n", err)
	}

	return nil
}

// CheckForUpdates checks all repositories for plugin updates
func (rs *RepositoryService) CheckForUpdates() ([]models.PluginUpdateInfo, error) {
	repositories, err := rs.store.GetAllRepositories()
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories: %w", err)
	}

	installedPlugins, err := rs.store.GetAllInstalledPlugins()
	if err != nil {
		return nil, fmt.Errorf("failed to get installed plugins: %w", err)
	}

	// Create a map of installed plugins by ID
	installedMap := make(map[string]*models.InstalledPlugin)
	for _, inst := range installedPlugins {
		installedMap[inst.PluginID] = inst
	}

	updates := make([]models.PluginUpdateInfo, 0)

	for _, repo := range repositories {
		manifest, err := rs.FetchRepository(repo.URL)
		if err != nil {
			// Log error but continue with other repositories
			fmt.Printf("Warning: failed to fetch repository %s: %v\n", repo.URL, err)
			continue
		}

		for _, plugin := range manifest.Plugins {
			installed, exists := installedMap[plugin.ID]
			if !exists {
				continue // Plugin not installed
			}

			// Check if installed plugin came from this repository
			if installed.RepositoryID.Valid && installed.RepositoryID.Int64 != repo.ID {
				continue // Plugin from different repository
			}

			// Validate API version before checking updates
			if err := ValidateAPIVersion(plugin.APIVersion); err != nil {
				continue // Skip incompatible plugins
			}

			// Compare versions using semantic versioning
			isNewer, err := IsNewerVersion(installed.InstalledVersion, plugin.Version)
			if err != nil {
				// If version comparison fails (invalid version), log and skip
				// This handles edge cases where version strings might not be semantic
				continue
			}

			if isNewer {
				updates = append(updates, models.PluginUpdateInfo{
					PluginID:         plugin.ID,
					Name:             plugin.Name,
					InstalledVersion: installed.InstalledVersion,
					AvailableVersion: plugin.Version,
					RepositoryID:     repo.ID,
					RepositoryName:   repo.Name,
					HasUpdate:        true,
				})
			}
		}
	}

	return updates, nil
}

// downloadFile downloads a file from a URL
func (rs *RepositoryService) downloadFile(url string) ([]byte, error) {
	resp, err := rs.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

