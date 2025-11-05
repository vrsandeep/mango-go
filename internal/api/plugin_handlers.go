package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vrsandeep/mango-go/internal/plugins"
)

// handleListPlugins lists all loaded plugins
func (s *Server) handleListPlugins(w http.ResponseWriter, r *http.Request) {
	manager := plugins.GetGlobalManager()
	if manager == nil {
		RespondWithError(w, http.StatusInternalServerError, "Plugin manager not initialized")
		return
	}

	pluginList := manager.ListPlugins()
	RespondWithJSON(w, http.StatusOK, pluginList)
}

// handleGetPluginInfo gets information about a specific plugin
func (s *Server) handleGetPluginInfo(w http.ResponseWriter, r *http.Request) {
	pluginID := chi.URLParam(r, "pluginID")

	manager := plugins.GetGlobalManager()
	if manager == nil {
		RespondWithError(w, http.StatusInternalServerError, "Plugin manager not initialized")
		return
	}

	pluginInfo, exists := manager.GetPluginInfo(pluginID)
	if !exists {
		RespondWithError(w, http.StatusNotFound, "Plugin not found")
		return
	}

	RespondWithJSON(w, http.StatusOK, pluginInfo)
}

// handleReloadPlugin reloads a specific plugin
func (s *Server) handleReloadPlugin(w http.ResponseWriter, r *http.Request) {
	pluginID := chi.URLParam(r, "pluginID")

	manager := plugins.GetGlobalManager()
	if manager == nil {
		RespondWithError(w, http.StatusInternalServerError, "Plugin manager not initialized")
		return
	}

	if err := manager.ReloadPlugin(pluginID); err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to reload plugin: %v", err))
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Plugin %s reloaded successfully", pluginID),
	})
}

// handleReloadAllPlugins reloads all plugins
func (s *Server) handleReloadAllPlugins(w http.ResponseWriter, r *http.Request) {
	manager := plugins.GetGlobalManager()
	if manager == nil {
		RespondWithError(w, http.StatusInternalServerError, "Plugin manager not initialized")
		return
	}

	if err := manager.ReloadAllPlugins(); err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to reload plugins: %v", err))
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "All plugins reloaded successfully",
	})
}

// handleUnloadPlugin unloads a specific plugin
func (s *Server) handleUnloadPlugin(w http.ResponseWriter, r *http.Request) {
	pluginID := chi.URLParam(r, "pluginID")

	manager := plugins.GetGlobalManager()
	if manager == nil {
		RespondWithError(w, http.StatusInternalServerError, "Plugin manager not initialized")
		return
	}

	if err := manager.UnloadPlugin(pluginID); err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to unload plugin: %v", err))
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Plugin %s unloaded successfully", pluginID),
	})
}

