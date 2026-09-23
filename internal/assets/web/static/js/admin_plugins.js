import { checkAuth } from './auth.js';

const DEFAULT_REPOSITORY_URL =
  'https://raw.githubusercontent.com/vrsandeep/mango-go-plugins/master/repository.json';
const AUTO_UPDATE_POLL_MS = 3000;

const notify = {
  success: message => window.toast?.success(message),
  error: message => window.toast?.error(message),
  info: message => window.toast?.info(message),
};

const escapeHtml = text => {
  const div = document.createElement('div');
  div.textContent = text ?? '';
  return div.innerHTML;
};

const pluralize = (count, singular, plural = `${singular}s`) =>
  `${count} ${count === 1 ? singular : plural}`;

const emptyState = (title, message = '') => `
  <div class="empty-state">
    <i class="ph-bold ph-package"></i>
    <h3>${title}</h3>
    ${message ? `<p>${message}</p>` : ''}
  </div>
`;

const statusRow = (kind, icon, body) => `
  <div class="status-row status-row-${kind}">
    <i class="ph-bold ${icon}${kind === 'info' ? ' ph-spin' : ''}"></i>
    ${body}
  </div>
`;

const pluginMeta = plugin => `
  <div class="plugin-meta">
    ${plugin.author ? `<span><i class="ph-bold ph-user"></i> ${escapeHtml(plugin.author)}</span>` : ''}
    ${plugin.license ? `<span><i class="ph-bold ph-certificate"></i> ${escapeHtml(plugin.license)}</span>` : ''}
    <span><i class="ph-bold ph-code"></i> API ${escapeHtml(plugin.api_version || 'N/A')}</span>
  </div>
`;

const closestDataset = (event, selector) => event.target.closest(selector)?.dataset;

const errorFromResponse = async (response, fallback) => {
  try {
    const body = await response.json();
    return body.error || fallback;
  } catch {
    return fallback;
  }
};

const requestJson = async (url, options = {}) => {
  const response = await fetch(url, options);
  if (!response.ok) {
    const error = new Error(
      await errorFromResponse(response, `Request failed (${response.status})`)
    );
    error.status = response.status;
    throw error;
  }

  const text = await response.text();
  return text ? JSON.parse(text) : null;
};

const withBusyButton = async (button, busyHtml, idleHtml, work) => {
  button.disabled = true;
  button.innerHTML = busyHtml;
  try {
    return await work();
  } finally {
    button.disabled = false;
    button.innerHTML = idleHtml;
  }
};

document.addEventListener('DOMContentLoaded', async () => {
  const currentUser = await checkAuth('admin');
  if (!currentUser) return;

  const state = {
    installedPlugins: [],
    repositories: [],
    autoUpdateStatus: null,
  };

  let autoUpdateStatusPollTimer = null;

  const els = {
    pluginsList: document.getElementById('plugins-list'),
    repositoriesList: document.getElementById('repositories-list'),
    addRepoBtn: document.getElementById('add-repo-btn'),
    browsePluginsBtn: document.getElementById('browse-plugins-btn'),
    autoUpdateBtn: document.getElementById('auto-update-btn'),
    autoUpdateStatus: document.getElementById('auto-update-status'),
    reloadAllBtn: document.getElementById('reload-all-btn'),
    addRepoModal: document.getElementById('add-repo-modal'),
    addRepoForm: document.getElementById('add-repo-form'),
    repoUrl: document.getElementById('repo-url'),
    repoName: document.getElementById('repo-name'),
    repoDescription: document.getElementById('repo-description'),
    addRepoSubmit: document.getElementById('add-repo-submit'),
    browsePluginsModal: document.getElementById('browse-plugins-modal'),
    browsePluginsTitle: document.getElementById('browse-plugins-title'),
    pluginsGrid: document.getElementById('plugins-grid'),
    repositorySelect: document.getElementById('repository-select'),
  };

  const showBrowsePrompt = () => {
    els.pluginsGrid.innerHTML = emptyState('Select a repository to browse plugins');
  };

  const isBrowseModalOpen = () => els.browsePluginsModal.style.display !== 'none';

  const refreshBrowseModal = async () => {
    const repositoryId = els.repositorySelect.value;
    if (isBrowseModalOpen() && repositoryId) {
      await loadPluginsForRepository(Number(repositoryId));
    }
  };

  const setModalOpen = (modal, open) => {
    modal.style.display = open ? 'flex' : 'none';
  };

  // --- Rendering ---
  const renderAutoUpdateStatus = () => {
    const status = state.autoUpdateStatus;
    if (!status) {
      els.autoUpdateStatus.innerHTML = '';
      return;
    }

    const rows = [];
    if (status.running) {
      rows.push(statusRow('info', 'ph-spinner', 'Updating plugins to the latest version...'));
    }
    if (status.error) {
      rows.push(
        statusRow(
          'error',
          'ph-warning',
          `Could not check for plugin updates: ${escapeHtml(status.error)}`
        )
      );
    }
    if (status.failed.length > 0) {
      rows.push(
        statusRow(
          'error',
          'ph-warning',
          `<div>
            <strong>${pluralize(status.failed.length, 'plugin')} could not be updated:</strong>
            <ul class="status-failures">
              ${status.failed
                .map(
                  failure =>
                    `<li>${escapeHtml(failure.plugin_id)}: ${escapeHtml(failure.error)}</li>`
                )
                .join('')}
            </ul>
          </div>`
        )
      );
    }
    if (!status.running && !status.error && status.failed.length === 0) {
      const message = status.has_run
        ? `All plugins are up to date (last checked ${escapeHtml(new Date(status.last_run_at).toLocaleString())})`
        : 'Plugins are updated to the latest version automatically when the server starts.';
      rows.push(statusRow('ok', 'ph-check-circle', message));
    }

    els.autoUpdateStatus.innerHTML = rows.join('');
  };

  const pluginStatusBadge = plugin => {
    if (plugin.loaded) return '<span class="status-badge status-active">Loaded</span>';
    if (plugin.error) return '<span class="status-badge status-error">Failed</span>';
    return '<span class="status-badge status-pending">Ready</span>';
  };

  const renderInstalledPlugins = () => {
    if (state.installedPlugins.length === 0) {
      els.pluginsList.innerHTML = emptyState(
        'No plugins installed',
        'Click "Browse & Install Plugins" to install plugins from repositories'
      );
      return;
    }

    els.pluginsList.innerHTML = state.installedPlugins
      .map(
        plugin => `
        <div class="plugin-card ${plugin.error ? 'plugin-error' : ''}">
          <div class="plugin-header">
            <div>
              <h3>${escapeHtml(plugin.name || plugin.id)}</h3>
              ${plugin.version ? `<span class="plugin-version">v${escapeHtml(plugin.version)}</span>` : ''}
            </div>
            ${pluginStatusBadge(plugin)}
          </div>
          ${plugin.description ? `<p class="plugin-description">${escapeHtml(plugin.description)}</p>` : ''}
          ${pluginMeta(plugin)}
          ${plugin.error ? `<div class="plugin-error-msg"><i class="ph-bold ph-warning"></i> ${escapeHtml(plugin.error)}</div>` : ''}
          <div class="plugin-actions">
            ${
              plugin.loaded
                ? `<button class="btn btn-secondary reload-plugin-btn" data-plugin-id="${escapeHtml(plugin.id)}">
                    <i class="ph-bold ph-arrow-clockwise"></i>
                    Reload
                  </button>
                  <button class="btn btn-danger unload-plugin-btn" data-plugin-id="${escapeHtml(plugin.id)}">
                    <i class="ph-bold ph-x"></i>
                    Unload
                  </button>`
                : ''
            }
          </div>
        </div>
      `
      )
      .join('');
  };

  const renderRepositories = () => {
    if (state.repositories.length === 0) {
      els.repositoriesList.innerHTML = emptyState(
        'No repositories',
        'Add a repository to browse and install plugins'
      );
      return;
    }

    els.repositoriesList.innerHTML = state.repositories
      .map(repo => {
        const isDefault = repo.url === DEFAULT_REPOSITORY_URL;
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
  };

  const renderRepositorySelector = () => {
    if (state.repositories.length === 0) {
      els.repositorySelect.innerHTML = '<option value="">No repositories available</option>';
      return;
    }

    els.repositorySelect.innerHTML =
      '<option value="">Select a repository...</option>' +
      state.repositories
        .map(repo => `<option value="${repo.id}">${escapeHtml(repo.name || repo.url)}</option>`)
        .join('');
  };

  const renderAvailablePlugins = (plugins, repositoryId) => {
    els.pluginsGrid.innerHTML = plugins
      .map(plugin => {
        const installed = state.installedPlugins.find(p => p.id === plugin.id);
        return `
          <div class="plugin-card">
            <div class="plugin-header">
              <h4>${escapeHtml(plugin.name)}</h4>
              <span class="plugin-version">v${escapeHtml(plugin.version)}</span>
            </div>
            <p class="plugin-description">${escapeHtml(plugin.description || 'No description')}</p>
            ${pluginMeta(plugin)}
            <div class="plugin-actions">
              ${
                installed
                  ? `<span class="installed-badge">
                      <i class="ph-bold ph-check-circle"></i>
                      Installed ${installed.version ? `(v${escapeHtml(installed.version)})` : ''}
                    </span>`
                  : `<button class="btn btn-primary install-plugin-btn" data-plugin-id="${escapeHtml(plugin.id)}" data-repo-id="${repositoryId}">
                      <i class="ph-bold ph-download"></i>
                      Install
                    </button>`
              }
            </div>
          </div>
        `;
      })
      .join('');
  };

  // --- Data loading ---
  const fetchInstalledPlugins = async () => {
    try {
      state.installedPlugins = await requestJson('/api/plugins');
      renderInstalledPlugins();
    } catch (error) {
      console.error('Error fetching installed plugins:', error);
      notify.error('Failed to load installed plugins');
    }
  };

  const fetchRepositories = async () => {
    try {
      state.repositories = await requestJson('/api/plugin-repositories');
      renderRepositories();
      renderRepositorySelector();
    } catch (error) {
      console.error('Error fetching repositories:', error);
      notify.error('Failed to load repositories');
    }
  };

  const fetchAvailablePlugins = async repositoryId => {
    try {
      return await requestJson(`/api/plugin-repositories/${repositoryId}/plugins`);
    } catch (error) {
      console.error('Error fetching available plugins:', error);
      notify.error('Failed to load available plugins');
      return [];
    }
  };

  const fetchAutoUpdateStatus = async () => {
    try {
      state.autoUpdateStatus = await requestJson(
        '/api/admin/plugin-repositories/auto-update-status'
      );
      renderAutoUpdateStatus();

      if (!state.autoUpdateStatus.running) return;

      clearTimeout(autoUpdateStatusPollTimer);
      autoUpdateStatusPollTimer = setTimeout(async () => {
        await fetchAutoUpdateStatus();
        await fetchInstalledPlugins();
      }, AUTO_UPDATE_POLL_MS);
    } catch (error) {
      console.error('Error fetching plugin update status:', error);
    }
  };

  const loadPluginsForRepository = async repositoryId => {
    if (!repositoryId) {
      showBrowsePrompt();
      return;
    }

    if (!state.repositories.some(repo => repo.id == repositoryId)) return;

    els.pluginsGrid.innerHTML =
      '<div class="loading-state"><i class="ph-bold ph-spinner ph-spin"></i> Loading plugins...</div>';

    const plugins = await fetchAvailablePlugins(repositoryId);
    if (plugins.length === 0) {
      els.pluginsGrid.innerHTML = emptyState(
        'No plugins available',
        "This repository doesn't have any compatible plugins"
      );
      return;
    }

    renderAvailablePlugins(plugins, repositoryId);
  };

  const openBrowseForRepository = async (repositoryId, repositoryName) => {
    setModalOpen(els.browsePluginsModal, true);
    els.repositorySelect.value = repositoryId;
    els.browsePluginsTitle.textContent = `Browse Plugins - ${repositoryName}`;
    await loadPluginsForRepository(repositoryId);
  };

  // --- Actions ---
  const addRepository = async (url, name, description) => {
    try {
      await requestJson('/api/admin/plugin-repositories', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url, name, description }),
      });
      notify.success('Repository added successfully');
      setModalOpen(els.addRepoModal, false);
      els.addRepoForm.reset();
      await fetchRepositories();
    } catch (error) {
      console.error('Error adding repository:', error);
      notify.error(error.message || 'Failed to add repository');
    }
  };

  const deleteRepository = async repositoryId => {
    if (!confirm('Are you sure you want to delete this repository?')) return;

    try {
      await requestJson(`/api/admin/plugin-repositories/${repositoryId}`, { method: 'DELETE' });
      notify.success('Repository deleted successfully');
      await fetchRepositories();
    } catch (error) {
      console.error('Error deleting repository:', error);
      notify.error('Failed to delete repository');
    }
  };

  const installPlugin = async (pluginId, repositoryId) => {
    try {
      await requestJson('/api/admin/plugin-repositories/install', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ plugin_id: pluginId, repository_id: repositoryId }),
      });
      notify.success(`Plugin ${pluginId} installed successfully`);
      await fetchInstalledPlugins();
      await refreshBrowseModal();
    } catch (error) {
      console.error('Error installing plugin:', error);
      notify.error(error.message || 'Failed to install plugin');
    }
  };

  const autoUpdatePlugins = async (showUpToDateToast = true) => {
    const idleHtml = '<i class="ph-bold ph-arrow-clockwise"></i> Update All Plugins';
    return withBusyButton(
      els.autoUpdateBtn,
      '<i class="ph-bold ph-spinner ph-spin"></i> Updating...',
      idleHtml,
      async () => {
        try {
          const result = await requestJson('/api/admin/plugin-repositories/auto-update', {
            method: 'POST',
          });

          if (result.updated.length > 0) {
            notify.success(
              `Updated ${pluralize(result.updated.length, 'plugin')} to the latest version`
            );
          }
          if (result.failed.length > 0) {
            notify.error(`Could not update: ${result.failed.map(f => f.plugin_id).join(', ')}`);
          } else if (result.updated.length === 0 && showUpToDateToast) {
            notify.success('All plugins are up to date');
          }

          return result;
        } catch (error) {
          if (error.status === 409) {
            notify.info('A plugin update is already in progress');
            return null;
          }
          console.error('Error automatically updating plugins:', error);
          notify.error(error.message || 'Failed to update plugins');
          return null;
        }
      }
    );
  };

  const reloadPlugin = async pluginId => {
    try {
      await requestJson(`/api/admin/plugins/${pluginId}/reload`, { method: 'POST' });
      notify.success(`Plugin ${pluginId} reloaded successfully`);
      await fetchInstalledPlugins();
    } catch (error) {
      console.error('Error reloading plugin:', error);
      notify.error(error.message || 'Failed to reload plugin');
    }
  };

  const unloadPlugin = async pluginId => {
    if (!confirm(`Are you sure you want to unload plugin "${pluginId}"?`)) return;

    try {
      await requestJson(`/api/admin/plugins/${pluginId}`, { method: 'DELETE' });
      notify.success(`Plugin ${pluginId} unloaded successfully`);
      await fetchInstalledPlugins();
    } catch (error) {
      console.error('Error unloading plugin:', error);
      notify.error(error.message || 'Failed to unload plugin');
    }
  };

  const reloadAllPlugins = async () => {
    if (!confirm('Are you sure you want to reload all plugins?')) return;

    const idleHtml = '<i class="ph-bold ph-arrow-clockwise"></i> Reload All Plugins';
    await withBusyButton(
      els.reloadAllBtn,
      '<i class="ph-bold ph-spinner ph-spin"></i> Reloading...',
      idleHtml,
      async () => {
        try {
          await requestJson('/api/admin/plugins/reload', { method: 'POST' });
          notify.success('All plugins reloaded successfully');
          await fetchInstalledPlugins();
        } catch (error) {
          console.error('Error reloading plugins:', error);
          notify.error(error.message || 'Failed to reload plugins');
        }
      }
    );
  };

  // --- Events (delegated so re-renders do not rebind listeners) ---
  document.querySelectorAll('.tab-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      const tabName = btn.dataset.tab;
      document.querySelectorAll('.tab-btn').forEach(tab => {
        tab.classList.toggle('active', tab.dataset.tab === tabName);
      });
      document.querySelectorAll('.tab-content').forEach(content => {
        content.classList.toggle('active', content.id === `tab-${tabName}`);
      });
    });
  });

  els.pluginsList.addEventListener('click', event => {
    const reload = closestDataset(event, '.reload-plugin-btn');
    if (reload) {
      reloadPlugin(reload.pluginId);
      return;
    }
    const unload = closestDataset(event, '.unload-plugin-btn');
    if (unload) unloadPlugin(unload.pluginId);
  });

  els.repositoriesList.addEventListener('click', event => {
    const browse = closestDataset(event, '.browse-plugins-btn');
    if (browse) {
      openBrowseForRepository(Number(browse.repoId), browse.repoName);
      return;
    }
    const remove = closestDataset(event, '.delete-repo-btn');
    if (remove) deleteRepository(Number(remove.repoId));
  });

  els.pluginsGrid.addEventListener('click', async event => {
    const install = event.target.closest('.install-plugin-btn');
    if (!install) return;

    const idleHtml = install.innerHTML;
    await withBusyButton(
      install,
      '<i class="ph-bold ph-spinner ph-spin"></i> Installing...',
      idleHtml,
      () => installPlugin(install.dataset.pluginId, Number(install.dataset.repoId))
    );
  });

  els.addRepoBtn.addEventListener('click', () => {
    els.addRepoForm.reset();
    setModalOpen(els.addRepoModal, true);
  });
  document.getElementById('add-repo-modal-close').addEventListener('click', () => {
    setModalOpen(els.addRepoModal, false);
    els.addRepoForm.reset();
  });
  document.getElementById('add-repo-cancel').addEventListener('click', () => {
    setModalOpen(els.addRepoModal, false);
    els.addRepoForm.reset();
  });
  els.addRepoSubmit.addEventListener('click', async event => {
    event.preventDefault();
    const url = els.repoUrl.value.trim();
    if (!url) {
      notify.error('Repository URL is required');
      return;
    }

    await withBusyButton(
      els.addRepoSubmit,
      '<i class="ph-bold ph-spinner ph-spin"></i> Adding...',
      'Add Repository',
      () =>
        addRepository(
          url,
          els.repoName.value.trim() || null,
          els.repoDescription.value.trim() || null
        )
    );
  });

  els.browsePluginsBtn.addEventListener('click', () => {
    els.repositorySelect.value = '';
    els.browsePluginsTitle.textContent = 'Browse & Install Plugins';
    showBrowsePrompt();
    setModalOpen(els.browsePluginsModal, true);
  });
  document.getElementById('browse-plugins-modal-close').addEventListener('click', () => {
    setModalOpen(els.browsePluginsModal, false);
  });

  els.autoUpdateBtn.addEventListener('click', async () => {
    await autoUpdatePlugins();
    await fetchAutoUpdateStatus();
    await fetchInstalledPlugins();
  });
  els.reloadAllBtn.addEventListener('click', reloadAllPlugins);

  els.repositorySelect.addEventListener('change', async event => {
    const repoId = event.target.value;
    if (!repoId) {
      showBrowsePrompt();
      return;
    }

    const repo = state.repositories.find(r => r.id == repoId);
    if (repo) {
      els.browsePluginsTitle.textContent = `Browse Plugins - ${repo.name || 'Repository'}`;
    }
    await loadPluginsForRepository(Number(repoId));
  });

  els.addRepoModal.addEventListener('click', event => {
    if (event.target === els.addRepoModal) {
      setModalOpen(els.addRepoModal, false);
      els.addRepoForm.reset();
    }
  });
  els.browsePluginsModal.addEventListener('click', event => {
    if (event.target === els.browsePluginsModal) setModalOpen(els.browsePluginsModal, false);
  });

  await fetchInstalledPlugins();
  await fetchRepositories();
  await fetchAutoUpdateStatus();
});
