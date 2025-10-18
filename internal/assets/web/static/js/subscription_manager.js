import { checkAuth } from './auth.js';

document.addEventListener('DOMContentLoaded', async () => {
  const currentUser = await checkAuth();
  if (!currentUser) return;

  const providerSelect = document.getElementById('provider-select');
  const subTableBody = document.getElementById('sub-table-body');
  let availableFolders = [];
  let currentSubscriptions = []; // Store current subscriptions data

  const timeAgo = date => {
    if (!date) return 'Never';
    const seconds = Math.floor((new Date() - date) / 1000);
    let interval = seconds / 31536000;
    if (interval > 1) return Math.floor(interval) + ' years ago';
    interval = seconds / 2592000;
    if (interval > 1) return Math.floor(interval) + ' months ago';
    interval = seconds / 86400;
    if (interval > 1) return Math.floor(interval) + ' days ago';
    interval = seconds / 3600;
    if (interval > 1) return Math.floor(interval) + ' hours ago';
    interval = seconds / 60;
    if (interval > 1) return Math.floor(interval) + ' minutes ago';
    return Math.floor(seconds) + ' seconds ago';
  };

  const getSubscriptionData = subId => {
    return currentSubscriptions.find(sub => sub.id == subId);
  };

  const renderTable = subs => {
    subTableBody.innerHTML = '';
    if (!subs || subs.length === 0) {
      subTableBody.innerHTML = '<tr><td colspan="6">No subscriptions found.</td></tr>';
      return;
    }
    subs.forEach(sub => {
      const row = document.createElement('tr');

      // Determine display text for folder path
      let folderPathDisplay = 'Default (series name)';
      if (sub.folder_path) {
        const defaultPath = window.PathUtils.getDefaultFolderPath(sub.series_title);
        if (sub.folder_path === defaultPath) {
          folderPathDisplay = 'Default (series name)';
        } else {
          folderPathDisplay = sub.folder_path;
        }
      }

      row.innerHTML = `
                        <td title="${sub.series_title}">${sub.series_title}</td>
                        <td>${sub.provider_id}</td>
                        <td class="folder-path-cell">
                            <span class="folder-path-display">${folderPathDisplay}</span>
                        </td>
                        <td>${timeAgo(new Date(sub.created_at))}</td>
                        <td>${timeAgo(sub.last_checked_at ? new Date(sub.last_checked_at) : null)}</td>
                        <td class="actions-cell">
                            <button data-action="edit-folder" data-id="${sub.id}" title="Edit folder path"><i class="ph-bold ph-pencil-simple"></i></button>
                            <button data-action="recheck" data-id="${sub.id}" title="Re-check for new chapters"><i class="ph-bold ph-arrow-clockwise"></i></button>
                            <button data-action="delete" data-id="${sub.id}" title="Delete subscription"><i class="ph-bold ph-trash"></i></button>
                        </td>
                    `;
      subTableBody.appendChild(row);
    });
  };

  const loadSubscriptions = async () => {
    const providerID = providerSelect.value;
    const url = `/api/subscriptions${providerID ? '?provider_id=' + providerID : ''}`;
    try {
      const response = await fetch(url);
      const subs = await response.json();
      currentSubscriptions = subs; // Store the data for later use
      renderTable(subs);
    } catch (e) {
      console.error('Failed to load subscriptions', e);
    }
  };

  const loadFolders = async () => {
    try {
      const response = await fetch('/api/folders');
      availableFolders = (await response.json()) || [];
    } catch (e) {
      console.error('Failed to load folders', e);
      availableFolders = [];
    }
  };

  const loadProviders = async () => {
    try {
      const response = await fetch('/api/providers');
      const providers = await response.json();
      if (providers) {
        providers.forEach(p => {
          const option = document.createElement('option');
          option.value = p.id;
          option.textContent = p.name;
          providerSelect.appendChild(option);
        });
      }
    } catch (e) {
      console.error('Failed to load providers', e);
    }
  };

  // Modal functionality
  let currentEditingSubId = null;

  const openFolderPathModal = subId => {
    const subData = getSubscriptionData(subId);
    if (!subData) return;

    currentEditingSubId = subId;

    // Set modal title
    document.getElementById('modal-title').textContent =
      `Edit Folder Path for "${subData.series_title}"`;

    // Set series name (read-only)
    document.getElementById('modal-series-name').value = subData.series_title;

    // Set current path display
    const currentPathDisplay = document.getElementById('modal-current-path-display');
    const defaultPath = window.PathUtils.getDefaultFolderPath(subData.series_title);
    const currentPath = subData.folder_path || defaultPath;
    currentPathDisplay.textContent = currentPath;

    // Set library path display
    document.getElementById('modal-library-path').textContent = window.PathUtils.getLibraryPath();

    // Populate folder select with available folders
    const folderSelect = document.getElementById('modal-folder-select');
    folderSelect.innerHTML = `
      <option value="">Default (series name)</option>
      <option value="__manual__">Custom path...</option>
      ${availableFolders
        .map(
          folder =>
            `<option value="${folder.path}" ${folder.path === subData.folder_path ? 'selected' : ''}>${folder.name}</option>`
        )
        .join('')}
    `;

    // Set initial selection
    if (subData.folder_path) {
      const defaultPath = window.PathUtils.getDefaultFolderPath(subData.series_title);
      if (subData.folder_path === defaultPath) {
        folderSelect.value = '';
      } else if (availableFolders.some(f => f.path === subData.folder_path)) {
        folderSelect.value = subData.folder_path;
      } else {
        folderSelect.value = '__manual__';
        // Extract relative path from the stored folder_path
        const libraryPath = window.PathUtils.getLibraryPath();
        const relativePath = subData.folder_path.startsWith(libraryPath)
          ? subData.folder_path.substring(libraryPath.length)
          : subData.folder_path;
        document.getElementById('modal-custom-path').value = relativePath;
      }
    } else {
      folderSelect.value = '';
    }

    // Show/hide custom path group and update preview
    updateModalCustomPathGroup();

    // Show modal
    document.getElementById('folder-path-modal').style.display = 'flex';
  };

  const updateModalCustomPathGroup = () => {
    const folderSelect = document.getElementById('modal-folder-select');
    const customPathGroup = document.getElementById('modal-custom-path-group');
    const customPathInput = document.getElementById('modal-custom-path');
    const pathPreview = document.getElementById('modal-path-preview');

    if (folderSelect.value === '__manual__') {
      customPathGroup.style.display = 'block';
      customPathInput.focus();
      updatePathPreview();
    } else {
      customPathGroup.style.display = 'none';
      customPathInput.value = '';
    }
  };

  const updatePathPreview = () => {
    const customPathInput = document.getElementById('modal-custom-path');
    const pathPreview = document.getElementById('modal-path-preview');
    const subData = getSubscriptionData(currentEditingSubId);

    if (!subData) return;

    let previewPath = '';
    const folderSelect = document.getElementById('modal-folder-select');

    if (folderSelect.value === '__manual__') {
      const customPath = customPathInput.value.trim();
      if (customPath) {
        // For custom paths, show the relative path + library path
        const sanitizedPath = window.PathUtils.sanitizePath(customPath) || customPath;
        const libraryPath = window.PathUtils.getLibraryPath();
        previewPath = `${libraryPath}${sanitizedPath}`;
      } else {
        previewPath = window.PathUtils.getDefaultFolderPath(subData.series_title);
      }
    } else if (folderSelect.value) {
      previewPath = folderSelect.value;
    } else {
      previewPath = window.PathUtils.getDefaultFolderPath(subData.series_title);
    }

    pathPreview.textContent = previewPath;
  };

  const closeModal = () => {
    document.getElementById('folder-path-modal').style.display = 'none';
    currentEditingSubId = null;

    // Reset form
    document.getElementById('modal-folder-select').value = '';
    document.getElementById('modal-custom-path').value = '';
    document.getElementById('modal-custom-path-group').style.display = 'none';
  };

  const saveFolderPathFromModal = async () => {
    if (!currentEditingSubId) return;

    const folderSelect = document.getElementById('modal-folder-select');
    const customPathInput = document.getElementById('modal-custom-path');

    let folderPath = null;

    if (folderSelect.value === '__manual__') {
      // Use manual input if custom option is selected
      const customPath = customPathInput.value.trim();
      if (customPath) {
        // Sanitize the relative path
        folderPath = window.PathUtils.sanitizePath(customPath);
        if (!folderPath) {
          toast.error('Invalid folder path. Please check for invalid characters.');
          return;
        }
        // The API expects the relative path, not the full path
        // The server will combine it with the library path
      }
    } else if (folderSelect.value) {
      // Use selected folder path
      folderPath = folderSelect.value;
    }

    try {
      await fetch(`/api/subscriptions/${currentEditingSubId}/folder-path`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ folder_path: folderPath }),
      });

      // Reload subscriptions to update the display
      await loadSubscriptions();

      closeModal();
      toast.success('Folder path updated successfully');
    } catch (error) {
      console.error('Failed to update folder path:', error);
      toast.error('Failed to update folder path');
    }
  };

  const resetToDefault = () => {
    document.getElementById('modal-folder-select').value = '';
    document.getElementById('modal-custom-path').value = '';
    updateModalCustomPathGroup();
    updatePathPreview();
  };

  subTableBody.addEventListener('click', async e => {
    const button = e.target.closest('button');
    if (!button) return;

    button.disabled = true; // Prevent double-clicks
    const action = button.dataset.action;
    const id = button.dataset.id;

    try {
      if (action === 'delete') {
        if (confirm('Are you sure you want to delete this subscription?')) {
          await fetch(`/api/subscriptions/${id}`, { method: 'DELETE' });
          loadSubscriptions();
        }
      } else if (action === 'recheck') {
        await fetch(`/api/subscriptions/${id}/recheck`, { method: 'POST' });

        // Update the last checked at value immediately
        const row = button.closest('tr');
        const lastCheckedCell = row.querySelector('td:nth-child(5)'); // Last Checked At column
        if (lastCheckedCell) {
          const now = new Date();
          lastCheckedCell.textContent = timeAgo(now);
        }

        toast.success(
          'Re-check initiated. New chapters will be added to the download queue if found.'
        );
      } else if (action === 'edit-folder') {
        openFolderPathModal(id);
      }
    } catch (e) {
      console.error(`Action ${action} failed:`, e);
      toast.error(`Action ${action} failed.`);
    } finally {
      button.disabled = false;
    }
  });

  providerSelect.addEventListener('change', () => {
    localStorage.setItem('sub_provider_filter', providerSelect.value);
    loadSubscriptions();
  });

  // Modal event listeners
  document.getElementById('modal-close-btn').addEventListener('click', closeModal);
  document.getElementById('modal-cancel-btn').addEventListener('click', closeModal);
  document.getElementById('modal-save-btn').addEventListener('click', saveFolderPathFromModal);
  document.getElementById('modal-reset-btn').addEventListener('click', resetToDefault);

  // Close modal when clicking outside
  document.getElementById('folder-path-modal').addEventListener('click', e => {
    if (e.target.id === 'folder-path-modal') {
      closeModal();
    }
  });

  // Close modal with Escape key
  document.addEventListener('keydown', e => {
    if (
      e.key === 'Escape' &&
      document.getElementById('folder-path-modal').style.display === 'flex'
    ) {
      closeModal();
    }
  });

  // Update preview when folder select changes
  document.getElementById('modal-folder-select').addEventListener('change', () => {
    updateModalCustomPathGroup();
    updatePathPreview();
  });

  // Update preview when custom path input changes
  document.getElementById('modal-custom-path').addEventListener('input', updatePathPreview);

  const init = async () => {
    await loadProviders();
    await window.PathUtils.loadLibraryPath();
    await loadFolders();
    const savedProvider = localStorage.getItem('sub_provider_filter');
    if (savedProvider) providerSelect.value = savedProvider;
    await loadSubscriptions();
  };
  init();
});
