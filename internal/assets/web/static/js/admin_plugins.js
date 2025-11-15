import { checkAuth } from './auth.js';

document.addEventListener('DOMContentLoaded', async () => {
  const currentUser = await checkAuth('admin');
  if (!currentUser) return;

  // --- State Management ---
  let state = {
    installedPlugins: [],
    repositories: [],
    availablePlugins: {},
    updates: [],
  };

  // --- DOM Elements ---
  const pluginsList = document.getElementById('plugins-list');
  const browsePluginsBtn = document.getElementById('browse-plugins-btn');
  const checkUpdatesBtn = document.getElementById('check-updates-btn');
  const reloadAllBtn = document.getElementById('reload-all-btn');
  const updatesSection = document.getElementById('updates-section');
  const updatesList = document.getElementById('updates-list');

  // Modal elements
  const browsePluginsModal = document.getElementById('browse-plugins-modal');
  const browsePluginsModalClose = document.getElementById('browse-plugins-modal-close');
  const browsePluginsTitle = document.getElementById('browse-plugins-title');
  const pluginsGrid = document.getElementById('plugins-grid');
  const repositorySelect = document.getElementById('repository-select');

  // --- Helper Functions ---
  const escapeHtml = text => {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  };

  // --- API Functions ---
  const fetchInstalledPlugins = async () => {
    try {
      const response = await fetch('/api/plugins');
      if (!response.ok) throw new Error('Failed to fetch installed plugins');
      state.installedPlugins = await response.json();
      renderInstalledPlugins();
    } catch (error) {
      console.error('Error fetching installed plugins:', error);
      if (window.toast) {
        toast.error('Failed to load installed plugins');
      }
    }
  };

  const fetchRepositories = async () => {
    try {
      const response = await fetch('/api/plugin-repositories');
      if (!response.ok) throw new Error('Failed to fetch repositories');
      state.repositories = await response.json();
      renderRepositorySelector();
    } catch (error) {
      console.error('Error fetching repositories:', error);
    }
  };

  const fetchAvailablePlugins = async repositoryId => {
    try {
      const response = await fetch(`/api/plugin-repositories/${repositoryId}/plugins`);
      if (!response.ok) throw new Error('Failed to fetch available plugins');
      return await response.json();
    } catch (error) {
      console.error('Error fetching available plugins:', error);
      if (window.toast) {
        toast.error('Failed to load available plugins');
      }
      return [];
    }
  };

  const installPlugin = async (pluginId, repositoryId) => {
    try {
      const response = await fetch('/api/admin/plugin-repositories/install', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ plugin_id: pluginId, repository_id: repositoryId }),
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to install plugin');
      }

      if (window.toast) {
        toast.success(`Plugin ${pluginId} installed successfully`);
      }
      await fetchInstalledPlugins();
      await loadPluginsForRepository(repositoryId);
    } catch (error) {
      console.error('Error installing plugin:', error);
      if (window.toast) {
        toast.error(error.message || 'Failed to install plugin');
      }
    }
  };

  const updatePlugin = async (pluginId, repositoryId) => {
    try {
      const response = await fetch('/api/admin/plugin-repositories/update', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ plugin_id: pluginId, repository_id: repositoryId }),
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to update plugin');
      }

      if (window.toast) {
        toast.success(`Plugin ${pluginId} updated successfully`);
      }
      await fetchInstalledPlugins();
      await checkForUpdates();
      await loadPluginsForRepository(repositoryId);
    } catch (error) {
      console.error('Error updating plugin:', error);
      if (window.toast) {
        toast.error(error.message || 'Failed to update plugin');
      }
    }
  };

  const reloadPlugin = async pluginId => {
    try {
      const response = await fetch(`/api/admin/plugins/${pluginId}/reload`, {
        method: 'POST',
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to reload plugin');
      }

      if (window.toast) {
        toast.success(`Plugin ${pluginId} reloaded successfully`);
      }
      await fetchInstalledPlugins();
    } catch (error) {
      console.error('Error reloading plugin:', error);
      if (window.toast) {
        toast.error(error.message || 'Failed to reload plugin');
      }
    }
  };

  const unloadPlugin = async pluginId => {
    if (!confirm(`Are you sure you want to unload plugin "${pluginId}"?`)) {
      return;
    }

    try {
      const response = await fetch(`/api/admin/plugins/${pluginId}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to unload plugin');
      }

      if (window.toast) {
        toast.success(`Plugin ${pluginId} unloaded successfully`);
      }
      await fetchInstalledPlugins();
    } catch (error) {
      console.error('Error unloading plugin:', error);
      if (window.toast) {
        toast.error(error.message || 'Failed to unload plugin');
      }
    }
  };

  const reloadAllPlugins = async () => {
    if (!confirm('Are you sure you want to reload all plugins?')) {
      return;
    }

    reloadAllBtn.disabled = true;
    reloadAllBtn.innerHTML = '<i class="ph-bold ph-spinner ph-spin"></i> Reloading...';

    try {
      const response = await fetch('/api/admin/plugins/reload', {
        method: 'POST',
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to reload plugins');
      }

      if (window.toast) {
        toast.success('All plugins reloaded successfully');
      }
      await fetchInstalledPlugins();
    } catch (error) {
      console.error('Error reloading plugins:', error);
      if (window.toast) {
        toast.error(error.message || 'Failed to reload plugins');
      }
    } finally {
      reloadAllBtn.disabled = false;
      reloadAllBtn.innerHTML = '<i class="ph-bold ph-arrow-clockwise"></i> Reload All Plugins';
    }
  };

  const checkForUpdates = async () => {
    checkUpdatesBtn.disabled = true;
    checkUpdatesBtn.innerHTML = '<i class="ph-bold ph-spinner ph-spin"></i> Checking...';

    try {
      const response = await fetch('/api/admin/plugin-repositories/check-updates', {
        method: 'POST',
      });

      if (!response.ok) throw new Error('Failed to check for updates');

      state.updates = await response.json();
      renderUpdates();

      if (state.updates.length > 0) {
        if (window.toast) {
          toast.info(`Found ${state.updates.length} plugin update(s) available`);
        }
      } else {
        if (window.toast) {
          toast.success('All plugins are up to date');
        }
      }
    } catch (error) {
      console.error('Error checking for updates:', error);
      if (window.toast) {
        toast.error('Failed to check for updates');
      }
    } finally {
      checkUpdatesBtn.disabled = false;
      checkUpdatesBtn.innerHTML = '<i class="ph-bold ph-arrow-clockwise"></i> Check for Updates';
    }
  };

  // --- Rendering Functions ---
  const renderInstalledPlugins = () => {
    if (state.installedPlugins.length === 0) {
      pluginsList.innerHTML = `
        <div class="empty-state">
          <i class="ph-bold ph-package"></i>
          <h3>No plugins installed</h3>
          <p>Click "Browse & Install Plugins" to install plugins from repositories</p>
        </div>
      `;
      return;
    }

    pluginsList.innerHTML = state.installedPlugins
      .map(
        plugin => {
          const hasUpdate = state.updates.find(u => u.plugin_id === plugin.id);
          const updateInfo = hasUpdate
            ? `<span class="update-available">Update available: v${escapeHtml(hasUpdate.available_version)}</span>`
            : '';

          return `
        <div class="plugin-card ${plugin.loaded ? '' : 'plugin-error'}">
          <div class="plugin-header">
            <div>
              <h3>${escapeHtml(plugin.name || plugin.id)}</h3>
              ${plugin.version ? `<span class="plugin-version">v${escapeHtml(plugin.version)}</span>` : ''}
            </div>
            ${plugin.loaded ? '<span class="status-badge status-active">Loaded</span>' : '<span class="status-badge status-error">Failed</span>'}
          </div>
          ${plugin.description ? `<p class="plugin-description">${escapeHtml(plugin.description)}</p>` : ''}
          <div class="plugin-meta">
            ${plugin.author ? `<span><i class="ph-bold ph-user"></i> ${escapeHtml(plugin.author)}</span>` : ''}
            ${plugin.license ? `<span><i class="ph-bold ph-certificate"></i> ${escapeHtml(plugin.license)}</span>` : ''}
            <span><i class="ph-bold ph-code"></i> API ${escapeHtml(plugin.api_version || 'N/A')}</span>
          </div>
          ${updateInfo ? `<div class="plugin-update-info">${updateInfo}</div>` : ''}
          ${plugin.error ? `<div class="plugin-error-msg"><i class="ph-bold ph-warning"></i> ${escapeHtml(plugin.error)}</div>` : ''}
          <div class="plugin-actions">
            ${hasUpdate ? `<button class="btn btn-primary update-plugin-btn" data-plugin-id="${plugin.id}" data-repo-id="${hasUpdate.repository_id}">
              <i class="ph-bold ph-arrow-clockwise"></i>
              Update
            </button>` : ''}
            ${plugin.loaded ? `<button class="btn btn-secondary reload-plugin-btn" data-plugin-id="${plugin.id}">
              <i class="ph-bold ph-arrow-clockwise"></i>
              Reload
            </button>` : ''}
            ${plugin.loaded ? `<button class="btn btn-danger unload-plugin-btn" data-plugin-id="${plugin.id}">
              <i class="ph-bold ph-x"></i>
              Unload
            </button>` : ''}
          </div>
        </div>
      `;
        }
      )
      .join('');

    // Attach event listeners
    document.querySelectorAll('.reload-plugin-btn').forEach(btn => {
      btn.addEventListener('click', e => {
        const pluginId = e.target.closest('.reload-plugin-btn').dataset.pluginId;
        reloadPlugin(pluginId);
      });
    });

    document.querySelectorAll('.unload-plugin-btn').forEach(btn => {
      btn.addEventListener('click', e => {
        const pluginId = e.target.closest('.unload-plugin-btn').dataset.pluginId;
        unloadPlugin(pluginId);
      });
    });

    document.querySelectorAll('.update-plugin-btn').forEach(btn => {
      btn.addEventListener('click', async e => {
        const pluginId = e.target.closest('.update-plugin-btn').dataset.pluginId;
        const repoId = parseInt(e.target.closest('.update-plugin-btn').dataset.repoId);
        btn.disabled = true;
        btn.innerHTML = '<i class="ph-bold ph-spinner ph-spin"></i> Updating...';
        await updatePlugin(pluginId, repoId);
        btn.disabled = false;
      });
    });
  };

  const renderRepositorySelector = () => {
    if (state.repositories.length === 0) {
      repositorySelect.innerHTML = '<option value="">No repositories available</option>';
      return;
    }

    repositorySelect.innerHTML =
      '<option value="">Select a repository...</option>' +
      state.repositories
        .map(repo => `<option value="${repo.id}">${escapeHtml(repo.name || repo.url)}</option>`)
        .join('');
  };

  const loadPluginsForRepository = async repositoryId => {
    if (!repositoryId) {
      pluginsGrid.innerHTML = `
        <div class="empty-state">
          <i class="ph-bold ph-package"></i>
          <h3>Select a repository to browse plugins</h3>
        </div>
      `;
      return;
    }

    const repo = state.repositories.find(r => r.id == repositoryId);
    if (!repo) return;

    pluginsGrid.innerHTML = '<div class="loading-state"><i class="ph-bold ph-spinner ph-spin"></i> Loading plugins...</div>';

    const plugins = await fetchAvailablePlugins(repositoryId);
    state.availablePlugins[repositoryId] = plugins;

    if (plugins.length === 0) {
      pluginsGrid.innerHTML = `
        <div class="empty-state">
          <i class="ph-bold ph-package"></i>
          <h3>No plugins available</h3>
          <p>This repository doesn't have any compatible plugins</p>
        </div>
      `;
      return;
    }

    pluginsGrid.innerHTML = plugins
      .map(plugin => {
        const installed = state.installedPlugins.find(p => p.id === plugin.id);
        const hasUpdate = state.updates.find(u => u.plugin_id === plugin.id);
        const canUpdate = installed && (hasUpdate || (installed.version && installed.version !== plugin.version));

        return `
      <div class="plugin-card">
        <div class="plugin-header">
          <h4>${escapeHtml(plugin.name)}</h4>
          <span class="plugin-version">v${escapeHtml(plugin.version)}</span>
        </div>
        <p class="plugin-description">${escapeHtml(plugin.description || 'No description')}</p>
        <div class="plugin-meta">
          ${plugin.author ? `<span><i class="ph-bold ph-user"></i> ${escapeHtml(plugin.author)}</span>` : ''}
          ${plugin.license ? `<span><i class="ph-bold ph-certificate"></i> ${escapeHtml(plugin.license)}</span>` : ''}
          <span><i class="ph-bold ph-code"></i> API ${escapeHtml(plugin.api_version)}</span>
        </div>
        <div class="plugin-actions">
          ${
            installed
              ? canUpdate
                ? `
                <button class="btn btn-primary update-plugin-btn" data-plugin-id="${plugin.id}" data-repo-id="${repositoryId}">
                  <i class="ph-bold ph-arrow-clockwise"></i>
                  Update (${escapeHtml(installed.version || 'unknown')} → ${escapeHtml(plugin.version)})
                </button>
              `
                : `
                <span class="installed-badge">
                  <i class="ph-bold ph-check-circle"></i>
                  Installed ${installed.version ? `(v${escapeHtml(installed.version)})` : ''}
                </span>
              `
              : `
              <button class="btn btn-primary install-plugin-btn" data-plugin-id="${plugin.id}" data-repo-id="${repositoryId}">
                <i class="ph-bold ph-download"></i>
                Install
              </button>
            `
          }
        </div>
      </div>
    `;
      })
      .join('');

    // Attach event listeners
    document.querySelectorAll('.install-plugin-btn').forEach(btn => {
      btn.addEventListener('click', async e => {
        const pluginId = e.target.closest('.install-plugin-btn').dataset.pluginId;
        const repoId = parseInt(e.target.closest('.install-plugin-btn').dataset.repoId);
        btn.disabled = true;
        btn.innerHTML = '<i class="ph-bold ph-spinner ph-spin"></i> Installing...';
        await installPlugin(pluginId, repoId);
        btn.disabled = false;
      });
    });

    document.querySelectorAll('.update-plugin-btn').forEach(btn => {
      btn.addEventListener('click', async e => {
        const pluginId = e.target.closest('.update-plugin-btn').dataset.pluginId;
        const repoId = parseInt(e.target.closest('.update-plugin-btn').dataset.repoId);
        btn.disabled = true;
        btn.innerHTML = '<i class="ph-bold ph-spinner ph-spin"></i> Updating...';
        await updatePlugin(pluginId, repoId);
        btn.disabled = false;
      });
    });
  };

  const renderUpdates = () => {
    if (state.updates.length === 0) {
      updatesSection.style.display = 'none';
      return;
    }

    updatesSection.style.display = 'block';
    updatesList.innerHTML = state.updates
      .map(
        update => `
      <div class="update-item">
        <div class="update-info">
          <h4>${escapeHtml(update.name)}</h4>
          <p class="update-versions">
            <span class="version-old">v${escapeHtml(update.installed_version)}</span>
            <i class="ph-bold ph-arrow-right"></i>
            <span class="version-new">v${escapeHtml(update.available_version)}</span>
          </p>
          <p class="update-repo">From: ${escapeHtml(update.repository_name)}</p>
        </div>
        <div class="update-actions">
          <button class="btn btn-primary update-plugin-btn" data-plugin-id="${update.plugin_id}" data-repo-id="${update.repository_id}">
            <i class="ph-bold ph-arrow-clockwise"></i>
            Update
          </button>
        </div>
      </div>
    `
      )
      .join('');

    // Attach event listeners for update buttons
    document.querySelectorAll('#updates-list .update-plugin-btn').forEach(btn => {
      btn.addEventListener('click', async e => {
        const pluginId = e.target.closest('.update-plugin-btn').dataset.pluginId;
        const repoId = parseInt(e.target.closest('.update-plugin-btn').dataset.repoId);
        btn.disabled = true;
        btn.innerHTML = '<i class="ph-bold ph-spinner ph-spin"></i> Updating...';
        await updatePlugin(pluginId, repoId);
        btn.disabled = false;
      });
    });
  };

  // --- Modal Functions ---
  const openBrowsePluginsModal = () => {
    browsePluginsModal.style.display = 'flex';
    repositorySelect.value = '';
    pluginsGrid.innerHTML = `
      <div class="empty-state">
        <i class="ph-bold ph-package"></i>
        <h3>Select a repository to browse plugins</h3>
      </div>
    `;
  };

  const closeBrowsePluginsModal = () => {
    browsePluginsModal.style.display = 'none';
  };

  // --- Event Listeners ---
  browsePluginsBtn.addEventListener('click', openBrowsePluginsModal);
  browsePluginsModalClose.addEventListener('click', closeBrowsePluginsModal);
  checkUpdatesBtn.addEventListener('click', checkForUpdates);
  reloadAllBtn.addEventListener('click', reloadAllPlugins);

  repositorySelect.addEventListener('change', async e => {
    const repoId = e.target.value;
    if (repoId) {
      await loadPluginsForRepository(parseInt(repoId));
    } else {
      pluginsGrid.innerHTML = `
        <div class="empty-state">
          <i class="ph-bold ph-package"></i>
          <h3>Select a repository to browse plugins</h3>
        </div>
      `;
    }
  });

  // Close modal when clicking overlay
  browsePluginsModal.addEventListener('click', e => {
    if (e.target === browsePluginsModal) closeBrowsePluginsModal();
  });

  // --- Initialization ---
  await fetchInstalledPlugins();
  await fetchRepositories();
  await checkForUpdates();
});

