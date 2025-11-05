package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/plugins"
)

// handleListRepositories lists all plugin repositories
func (s *Server) handleListRepositories(w http.ResponseWriter, r *http.Request) {
	repositories, err := s.store.GetAllRepositories()
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list repositories: %v", err))
		return
	}

	RespondWithJSON(w, http.StatusOK, repositories)
}

// handleCreateRepository creates a new plugin repository
func (s *Server) handleCreateRepository(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL         string `json:"url"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.URL == "" {
		RespondWithError(w, http.StatusBadRequest, "URL is required")
		return
	}

	// Validate that URL is a valid repository URL (fetch and parse)
	manager := plugins.GetGlobalManager()
	if manager == nil {
		RespondWithError(w, http.StatusInternalServerError, "Plugin manager not initialized")
		return
	}
	repoService := plugins.NewRepositoryService(s.app, s.store, manager)
	_, err := repoService.FetchRepository(req.URL)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid repository URL: %v", err))
		return
	}

	repo, err := s.store.CreateRepository(req.URL, req.Name, req.Description)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create repository: %v", err))
		return
	}

	RespondWithJSON(w, http.StatusCreated, repo)
}

// handleGetRepositoryPlugins gets available plugins from a repository
func (s *Server) handleGetRepositoryPlugins(w http.ResponseWriter, r *http.Request) {
	repositoryID := chi.URLParam(r, "repositoryID")
	if repositoryID == "" {
		RespondWithError(w, http.StatusBadRequest, "Repository ID is required")
		return
	}

	var repoID int64
	if _, err := fmt.Sscanf(repositoryID, "%d", &repoID); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid repository ID")
		return
	}

	manager := plugins.GetGlobalManager()
	if manager == nil {
		RespondWithError(w, http.StatusInternalServerError, "Plugin manager not initialized")
		return
	}
	repoService := plugins.NewRepositoryService(s.app, s.store, manager)
	availablePlugins, err := repoService.GetAvailablePlugins(repoID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get plugins: %v", err))
		return
	}

	RespondWithJSON(w, http.StatusOK, availablePlugins)
}

// handleInstallPlugin installs a plugin from a repository
func (s *Server) handleInstallPlugin(w http.ResponseWriter, r *http.Request) {
	var req models.PluginInstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.PluginID == "" {
		RespondWithError(w, http.StatusBadRequest, "Plugin ID is required")
		return
	}

	manager := plugins.GetGlobalManager()
	if manager == nil {
		RespondWithError(w, http.StatusInternalServerError, "Plugin manager not initialized")
		return
	}
	repoService := plugins.NewRepositoryService(s.app, s.store, manager)

	if err := repoService.InstallPlugin(req.PluginID, req.RepositoryID); err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to install plugin: %v", err))
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Plugin %s installed successfully", req.PluginID),
	})
}

// handleCheckUpdates checks for plugin updates across all repositories
func (s *Server) handleCheckUpdates(w http.ResponseWriter, r *http.Request) {
	manager := plugins.GetGlobalManager()
	if manager == nil {
		RespondWithError(w, http.StatusInternalServerError, "Plugin manager not initialized")
		return
	}
	repoService := plugins.NewRepositoryService(s.app, s.store, manager)

	updates, err := repoService.CheckForUpdates()
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to check for updates: %v", err))
		return
	}

	RespondWithJSON(w, http.StatusOK, updates)
}

// handleUpdatePlugin updates an installed plugin to the latest version from its repository
func (s *Server) handleUpdatePlugin(w http.ResponseWriter, r *http.Request) {
	var req models.PluginInstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.PluginID == "" {
		RespondWithError(w, http.StatusBadRequest, "Plugin ID is required")
		return
	}

	manager := plugins.GetGlobalManager()
	if manager == nil {
		RespondWithError(w, http.StatusInternalServerError, "Plugin manager not initialized")
		return
	}
	repoService := plugins.NewRepositoryService(s.app, s.store, manager)

	// Get installed plugin info
	installed, err := s.store.GetInstalledPlugin(req.PluginID)
	if err != nil {
		RespondWithError(w, http.StatusNotFound, fmt.Sprintf("Plugin %s is not installed", req.PluginID))
		return
	}

	// Determine repository ID
	var repoID int64
	if installed.RepositoryID.Valid {
		repoID = installed.RepositoryID.Int64
	} else if req.RepositoryID > 0 {
		repoID = req.RepositoryID
	} else {
		RespondWithError(w, http.StatusBadRequest, "Repository ID is required")
		return
	}

	// Install/update the plugin
	if err := repoService.InstallPlugin(req.PluginID, repoID); err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update plugin: %v", err))
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Plugin %s updated successfully", req.PluginID),
	})
}

// handleDeleteRepository deletes a repository
func (s *Server) handleDeleteRepository(w http.ResponseWriter, r *http.Request) {
	repositoryID := chi.URLParam(r, "repositoryID")
	if repositoryID == "" {
		RespondWithError(w, http.StatusBadRequest, "Repository ID is required")
		return
	}

	var repoID int64
	if _, err := fmt.Sscanf(repositoryID, "%d", &repoID); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid repository ID")
		return
	}

	if err := s.store.DeleteRepository(repoID); err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete repository: %v", err))
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Repository deleted successfully",
	})
}

