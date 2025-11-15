import { checkAuth } from './auth.js';

document.addEventListener('DOMContentLoaded', async () => {
  const currentUser = await checkAuth('admin');
  if (!currentUser) return;

  // --- State Management ---
  let state = {
    repositories: [],
    availablePlugins: {},
    updates: [],
    installedPlugins: [],
  };

  // --- DOM Elements ---
  const repositoriesList = document.getElementById('repositories-list');
  const addRepoBtn = document.getElementById('add-repo-btn');
  const checkUpdatesBtn = document.getElementById('check-updates-btn');
  const updatesSection = document.getElementById('updates-section');
  const updatesList = document.getElementById('updates-list');

  // Modal elements
  const addRepoModal = document.getElementById('add-repo-modal');
  const addRepoModalClose = document.getElementById('add-repo-modal-close');
  const addRepoCancel = document.getElementById('add-repo-cancel');
  const addRepoSubmit = document.getElementById('add-repo-submit');
  const addRepoForm = document.getElementById('add-repo-form');
  const repoUrlInput = document.getElementById('repo-url');
  const repoNameInput = document.getElementById('repo-name');
  const repoDescriptionInput = document.getElementById('repo-description');

  const browsePluginsModal = document.getElementById('browse-plugins-modal');
  const browsePluginsModalClose = document.getElementById('browse-plugins-modal-close');
  const browsePluginsTitle = document.getElementById('browse-plugins-title');
  const pluginsGrid = document.getElementById('plugins-grid');

  // --- API Functions ---
  const fetchRepositories = async () => {
    try {
      const response = await fetch('/api/plugin-repositories');
      if (!response.ok) throw new Error('Failed to fetch repositories');
      state.repositories = await response.json();
      renderRepositories();
    } catch (error) {
      console.error('Error fetching repositories:', error);
      if (window.toast) {
        toast.error('Failed to load repositories');
      }
    }
  };

  const fetchInstalledPlugins = async () => {
    try {
      const response = await fetch('/api/plugins');
      if (!response.ok) throw new Error('Failed to fetch installed plugins');
      state.installedPlugins = await response.json();
    } catch (error) {
      console.error('Error fetching installed plugins:', error);
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

  const addRepository = async (url, name, description) => {
    try {
      const response = await fetch('/api/admin/plugin-repositories', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url, name, description }),
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to add repository');
      }

      const repo = await response.json();
      if (window.toast) {
        toast.success('Repository added successfully');
      }
      closeAddRepoModal();
      await fetchRepositories();
      return repo;
    } catch (error) {
      console.error('Error adding repository:', error);
      if (window.toast) {
        toast.error(error.message || 'Failed to add repository');
      }
      throw error;
    }
  };

  const deleteRepository = async repositoryId => {
    if (!confirm('Are you sure you want to delete this repository?')) return;

    try {
      const response = await fetch(`/api/admin/plugin-repositories/${repositoryId}`, {
        method: 'DELETE',
      });

      if (!response.ok) throw new Error('Failed to delete repository');

      if (window.toast) {
        toast.success('Repository deleted successfully');
      }
      await fetchRepositories();
    } catch (error) {
      console.error('Error deleting repository:', error);
      if (window.toast) {
        toast.error('Failed to delete repository');
      }
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
      // Refresh the plugins grid if modal is open
      if (browsePluginsModal.style.display !== 'none') {
        const currentRepoId = browsePluginsModal.dataset.repositoryId;
        if (currentRepoId) {
          await loadPluginsForRepository(parseInt(currentRepoId));
        }
      }
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
      // Refresh the plugins grid if modal is open
      if (browsePluginsModal.style.display !== 'none') {
        const currentRepoId = browsePluginsModal.dataset.repositoryId;
        if (currentRepoId) {
          await loadPluginsForRepository(parseInt(currentRepoId));
        }
      }
    } catch (error) {
      console.error('Error updating plugin:', error);
      if (window.toast) {
        toast.error(error.message || 'Failed to update plugin');
      }
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
  const renderRepositories = () => {
    if (state.repositories.length === 0) {
      repositoriesList.innerHTML = `
        <div class="empty-state">
          <i class="ph-bold ph-package"></i>
          <h3>No repositories</h3>
          <p>Add a repository to browse and install plugins</p>
        </div>
      `;
      return;
    }

    const defaultRepositoryURL =
      'https://raw.githubusercontent.com/vrsandeep/mango-go-plugins/master/repository.json';

    repositoriesList.innerHTML = state.repositories
      .map(repo => {
        const isDefault = repo.url === defaultRepositoryURL;
        return `
      <div class="repository-card ${isDefault ? 'default-repository' : ''}">
        <div class="repository-info">
          <h3>${escapeHtml(repo.name || 'Unnamed Repository')}${isDefault ? ' <span class="default-badge">Default</span>' : ''}</h3>
          <p class="repository-url">${escapeHtml(repo.url)}</p>
          ${repo.description ? `<p class="repository-description">${escapeHtml(repo.description)}</p>` : ''}
        </div>
        <div class="repository-actions">
          <button class="btn btn-secondary browse-plugins-btn" data-repo-id="${repo.id}" data-repo-name="${escapeHtml(repo.name || 'Repository')}">
            <i class="ph-bold ph-magnifying-glass"></i>
            Browse Plugins
          </button>
          ${
            isDefault
              ? ''
              : `<button class="btn btn-danger delete-repo-btn" data-repo-id="${repo.id}">
            <i class="ph-bold ph-trash"></i>
            Delete
          </button>`
          }
        </div>
      </div>
    `;
      })
      .join('');

    // Attach event listeners
    document.querySelectorAll('.browse-plugins-btn').forEach(btn => {
      btn.addEventListener('click', async e => {
        const repoId = parseInt(e.target.closest('.browse-plugins-btn').dataset.repoId);
        const repoName = e.target.closest('.browse-plugins-btn').dataset.repoName;
        await loadPluginsForRepository(repoId, repoName);
      });
    });

    document.querySelectorAll('.delete-repo-btn').forEach(btn => {
      btn.addEventListener('click', e => {
        const repoId = parseInt(e.target.closest('.delete-repo-btn').dataset.repoId);
        deleteRepository(repoId);
      });
    });
  };

  const loadPluginsForRepository = async (repositoryId, repositoryName) => {
    browsePluginsTitle.textContent = `Available Plugins - ${repositoryName}`;
    browsePluginsModal.dataset.repositoryId = repositoryId;
    pluginsGrid.innerHTML =
      '<div class="loading-state"><i class="ph-bold ph-spinner ph-spin"></i> Loading plugins...</div>';
    openBrowsePluginsModal();

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
        // Check if update is available (either has update entry or version differs)
        const canUpdate = installed && (hasUpdate || installed.version !== plugin.version);

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
                  Update (${escapeHtml(installed.version)} → ${escapeHtml(plugin.version)})
                </button>
              `
                : `
                <span class="installed-badge">
                  <i class="ph-bold ph-check-circle"></i>
                  Installed (v${escapeHtml(installed.version)})
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
  const openAddRepoModal = () => {
    addRepoModal.style.display = 'flex';
    repoUrlInput.focus();
  };

  const closeAddRepoModal = () => {
    addRepoModal.style.display = 'none';
    addRepoForm.reset();
  };

  const openBrowsePluginsModal = () => {
    browsePluginsModal.style.display = 'flex';
  };

  const closeBrowsePluginsModal = () => {
    browsePluginsModal.style.display = 'none';
  };

  // --- Event Listeners ---
  addRepoBtn.addEventListener('click', openAddRepoModal);
  addRepoModalClose.addEventListener('click', closeAddRepoModal);
  addRepoCancel.addEventListener('click', closeAddRepoModal);
  checkUpdatesBtn.addEventListener('click', checkForUpdates);

  browsePluginsModalClose.addEventListener('click', closeBrowsePluginsModal);

  // Close modals when clicking overlay
  addRepoModal.addEventListener('click', e => {
    if (e.target === addRepoModal) closeAddRepoModal();
  });

  browsePluginsModal.addEventListener('click', e => {
    if (e.target === browsePluginsModal) closeBrowsePluginsModal();
  });

  // Add repository form submission
  addRepoSubmit.addEventListener('click', async e => {
    e.preventDefault();
    const url = repoUrlInput.value.trim();
    const name = repoNameInput.value.trim();
    const description = repoDescriptionInput.value.trim();

    if (!url) {
      if (window.toast) {
        toast.error('Repository URL is required');
      }
      return;
    }

    addRepoSubmit.disabled = true;
    addRepoSubmit.innerHTML = '<i class="ph-bold ph-spinner ph-spin"></i> Adding...';

    try {
      await addRepository(url, name || null, description || null);
    } finally {
      addRepoSubmit.disabled = false;
      addRepoSubmit.innerHTML = 'Add Repository';
    }
  });

  // Escape HTML helper
  const escapeHtml = text => {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  };

  // --- Initialization ---
  await fetchRepositories();
  await fetchInstalledPlugins();
  await checkForUpdates(); // Check for updates on page load
});

