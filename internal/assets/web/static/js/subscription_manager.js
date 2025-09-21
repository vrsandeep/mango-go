import { checkAuth } from './auth.js';

document.addEventListener('DOMContentLoaded', async () => {
  const currentUser = await checkAuth();
  if (!currentUser) return;

  const providerSelect = document.getElementById('provider-select');
  const subTableBody = document.getElementById('sub-table-body');
  let availableFolders = [];

  const timeAgo = (date) => {
    if (!date) return 'Never';
    const seconds = Math.floor((new Date() - date) / 1000);
    let interval = seconds / 31536000;
    if (interval > 1) return Math.floor(interval) + " years ago";
    interval = seconds / 2592000;
    if (interval > 1) return Math.floor(interval) + " months ago";
    interval = seconds / 86400;
    if (interval > 1) return Math.floor(interval) + " days ago";
    interval = seconds / 3600;
    if (interval > 1) return Math.floor(interval) + " hours ago";
    interval = seconds / 60;
    if (interval > 1) return Math.floor(interval) + " minutes ago";
    return Math.floor(seconds) + " seconds ago";
  };

  const renderTable = (subs) => {
    subTableBody.innerHTML = '';
    if (!subs || subs.length === 0) {
      subTableBody.innerHTML = '<tr><td colspan="6">No subscriptions found.</td></tr>';
      return;
    }
    subs.forEach(sub => {
      const row = document.createElement('tr');
      const folderPath = sub.folder_path || 'Default (series name)';
      row.innerHTML = `
                        <td title="${sub.series_title}">${sub.series_title}</td>
                        <td>${sub.provider_id}</td>
                        <td class="folder-path-cell">
                            <span class="folder-path-display">${folderPath}</span>
                            <div class="folder-path-edit" style="display: none;" data-sub-id="${sub.id}">
                                <select class="folder-path-select" style="margin-bottom: 5px;">
                                    <option value="">Default (series name)</option>
                                    <option value="__manual__">Custom path...</option>
                                    ${availableFolders.map(folder =>
                                      `<option value="${folder.path}" ${folder.path === sub.folder_path ? 'selected' : ''}>${folder.name}</option>`
                                    ).join('')}
                                </select>
                                <input type="text" class="folder-path-manual" placeholder="Enter custom folder path..." style="display: none; width: 100%; padding: 3px; border: 1px solid #ccc; border-radius: 3px;" value="${sub.folder_path && !availableFolders.some(f => f.path === sub.folder_path) ? sub.folder_path : ''}">
                            </div>
                        </td>
                        <td>${timeAgo(new Date(sub.created_at))}</td>
                        <td>${timeAgo(sub.last_checked_at ? new Date(sub.last_checked_at) : null)}</td>
                        <td class="actions-cell">
                            <button data-action="edit-folder" data-id="${sub.id}" title="Edit folder path"><i class="ph-bold ph-pencil-simple"></i></button>
                            <button data-action="save-folder" data-id="${sub.id}" title="Save folder path" style="display: none;"><i class="ph-bold ph-check"></i></button>
                            <button data-action="cancel-folder" data-id="${sub.id}" title="Cancel editing" style="display: none;"><i class="ph-bold ph-x"></i></button>
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
      renderTable(subs);
    } catch (e) {
      console.error("Failed to load subscriptions", e);
    }
  };

  const loadFolders = async () => {
    try {
      const response = await fetch('/api/folders');
      availableFolders = await response.json() || [];
    } catch (e) {
      console.error("Failed to load folders", e);
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
    } catch (e) { console.error("Failed to load providers", e); }
  };

  const handleEditFolder = (subId) => {
    const row = document.querySelector(`[data-sub-id="${subId}"]`).closest('tr');
    const display = row.querySelector('.folder-path-display');
    const editDiv = row.querySelector('.folder-path-edit');
    const select = row.querySelector('.folder-path-select');
    const manualInput = row.querySelector('.folder-path-manual');
    const editBtn = row.querySelector('[data-action="edit-folder"]');
    const saveBtn = row.querySelector('[data-action="save-folder"]');
    const cancelBtn = row.querySelector('[data-action="cancel-folder"]');

    display.style.display = 'none';
    editDiv.style.display = 'block';
    editBtn.style.display = 'none';
    saveBtn.style.display = 'inline-block';
    cancelBtn.style.display = 'inline-block';

    // Set up select change handler for this row
    select.addEventListener('change', () => handleFolderSelectChangeInRow(subId));
  };

  const handleFolderSelectChangeInRow = (subId) => {
    const row = document.querySelector(`[data-sub-id="${subId}"]`).closest('tr');
    const select = row.querySelector('.folder-path-select');
    const manualInput = row.querySelector('.folder-path-manual');

    if (select.value === '__manual__') {
      manualInput.style.display = 'block';
      // Pre-fill with library path if not already set
      window.PathUtils.prefillCustomPath(manualInput);
      manualInput.focus();
    } else {
      manualInput.style.display = 'none';
      manualInput.value = '';
    }
  };


  const handleSaveFolder = async (subId) => {
    const row = document.querySelector(`[data-sub-id="${subId}"]`).closest('tr');
    const select = row.querySelector('.folder-path-select');
    const manualInput = row.querySelector('.folder-path-manual');
    const editDiv = row.querySelector('.folder-path-edit');

    let folderPath = null;

    if (select.value === '__manual__') {
      // Use manual input if custom option is selected
      const customPath = manualInput.value.trim();
      if (customPath) {
        folderPath = window.PathUtils.sanitizePath(customPath);
        if (!folderPath) {
          toast.error('Invalid folder path. Please check for invalid characters.');
          return;
        }
      }
    } else if (select.value) {
      // Use selected folder path
      folderPath = select.value;
    }

    try {
      await fetch(`/api/subscriptions/${subId}/folder-path`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ folder_path: folderPath })
      });

      // Update display and hide editing controls
      const display = row.querySelector('.folder-path-display');
      const editBtn = row.querySelector('[data-action="edit-folder"]');
      const saveBtn = row.querySelector('[data-action="save-folder"]');
      const cancelBtn = row.querySelector('[data-action="cancel-folder"]');

      display.textContent = folderPath || 'Default (series name)';
      display.style.display = 'block';
      editDiv.style.display = 'none';
      editBtn.style.display = 'inline-block';
      saveBtn.style.display = 'none';
      cancelBtn.style.display = 'none';

      toast.success('Folder path updated successfully');
    } catch (error) {
      console.error('Failed to update folder path:', error);
      toast.error('Failed to update folder path');
    }
  };

  const handleCancelFolder = (subId) => {
    const row = document.querySelector(`[data-sub-id="${subId}"]`).closest('tr');
    const display = row.querySelector('.folder-path-display');
    const editDiv = row.querySelector('.folder-path-edit');
    const select = row.querySelector('.folder-path-select');
    const manualInput = row.querySelector('.folder-path-manual');
    const editBtn = row.querySelector('[data-action="edit-folder"]');
    const saveBtn = row.querySelector('[data-action="save-folder"]');
    const cancelBtn = row.querySelector('[data-action="cancel-folder"]');

    display.style.display = 'block';
    editDiv.style.display = 'none';
    editBtn.style.display = 'inline-block';
    saveBtn.style.display = 'none';
    cancelBtn.style.display = 'none';

    // Reset form
    select.value = '';
    manualInput.style.display = 'none';
    manualInput.value = '';
  };

  subTableBody.addEventListener('click', async (e) => {
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
        toast.success('Re-check initiated. New chapters will be added to the download queue if found.');
      } else if (action === 'edit-folder') {
        handleEditFolder(id);
      } else if (action === 'save-folder') {
        await handleSaveFolder(id);
      } else if (action === 'cancel-folder') {
        handleCancelFolder(id);
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