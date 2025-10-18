import { checkAuth } from './auth.js';

document.addEventListener('DOMContentLoaded', async () => {
  const currentUser = await checkAuth('admin');
  if (!currentUser) return;

  // DOM elements
  const loadingEl = document.getElementById('loading');
  const noFilesEl = document.getElementById('no-files');
  const filesTableEl = document.getElementById('files-table');
  const filesTbodyEl = document.getElementById('files-tbody');
  const totalCountEl = document.getElementById('total-count');
  const lastUpdatedEl = document.getElementById('last-updated');
  const refreshBtn = document.getElementById('refresh-btn');
  const downloadCsvBtn = document.getElementById('download-csv-btn');
  const searchInput = document.getElementById('search-input');
  const errorFilter = document.getElementById('error-filter');

  let allBadFiles = [];
  let filteredBadFiles = [];

  // Initialize the page
  init();

  function init() {
    loadBadFiles();
    setupEventListeners();
  }

  function setupEventListeners() {
    refreshBtn.addEventListener('click', loadBadFiles);
    downloadCsvBtn.addEventListener('click', downloadCSV);
    searchInput.addEventListener('input', filterFiles);
    errorFilter.addEventListener('change', filterFiles);
  }

  async function loadBadFiles() {
    try {
      showLoading();

      // Get bad files count first
      const countResponse = await fetch('/api/admin/bad-files/count');
      const countData = await countResponse.json();

      if (countResponse.ok) {
        totalCountEl.textContent = countData.count;

        // Replace CSV download button with download from server button
        if (countData.show_download) {
          showError(
            'Only the first 50 files are shown. <a href="/api/admin/bad-files/download">Download Entire List</a>.'
          );
          downloadCsvBtn.removeEventListener('click', downloadCSV);
          downloadCsvBtn.addEventListener('click', downloadFromServer);
        }
      }

      if (countData.count === 0) {
        allBadFiles = [];
        filteredBadFiles = [];
        downloadCsvBtn.style.display = 'none';
      } else {
        downloadCsvBtn.style.display = 'inline-flex';
        // Get all bad files
        const response = await fetch('/api/admin/bad-files');
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }
        allBadFiles = await response.json();
        filteredBadFiles = [...allBadFiles];
      }

      updateLastUpdated();
      renderFiles();
    } catch (error) {
      console.error('Error loading bad files:', error);
      showError('Failed to load bad files. Please try again.');
    }
  }

  function showLoading() {
    loadingEl.style.display = 'flex';
    noFilesEl.style.display = 'none';
    filesTableEl.style.display = 'none';
  }

  function showError(message) {
    loadingEl.style.display = 'none';
    noFilesEl.style.display = 'flex';
    noFilesEl.innerHTML = `
            <i class="ph-bold ph-warning"></i>
            <h3>Error</h3>
            <p>${message}</p>
        `;
  }

  function updateLastUpdated() {
    const now = new Date();
    lastUpdatedEl.textContent = now.toLocaleString();
  }

  function renderFiles() {
    loadingEl.style.display = 'none';

    if (filteredBadFiles.length === 0) {
      noFilesEl.style.display = 'flex';
      filesTableEl.style.display = 'none';
      return;
    }

    noFilesEl.style.display = 'none';
    filesTableEl.style.display = 'block';

    // Clear existing rows
    filesTbodyEl.innerHTML = '';

    // Add file rows
    filteredBadFiles.forEach(file => {
      const row = createFileRow(file);
      filesTbodyEl.appendChild(row);
    });
  }

  function createFileRow(file) {
    const row = document.createElement('tr');

    const fileName = document.createElement('td');
    fileName.className = 'file-name';
    fileName.textContent = file.file_name;

    const filePath = document.createElement('td');
    filePath.className = 'file-path';
    filePath.title = file.path; // Show full path on hover
    filePath.textContent = file.path;

    const error = document.createElement('td');
    const errorBadge = document.createElement('span');
    errorBadge.className = `error-badge ${file.error}`;
    errorBadge.textContent = getErrorDisplayName(file.error);
    error.appendChild(errorBadge);

    const size = document.createElement('td');
    size.className = 'file-size';
    size.textContent = formatFileSize(file.file_size);

    const detected = document.createElement('td');
    detected.className = 'detected-date';
    detected.textContent = new Date(file.detected_at).toLocaleDateString();

    const actions = document.createElement('td');
    actions.className = 'action-buttons';

    const deleteBtn = document.createElement('button');
    deleteBtn.className = 'action-btn delete';
    deleteBtn.textContent = 'Dismiss';
    deleteBtn.title = 'Remove this entry from the bad files list';
    deleteBtn.addEventListener('click', () => deleteBadFile(file.id));

    actions.appendChild(deleteBtn);

    row.appendChild(fileName);
    row.appendChild(filePath);
    row.appendChild(error);
    row.appendChild(size);
    row.appendChild(detected);
    row.appendChild(actions);

    return row;
  }

  function getErrorDisplayName(errorType) {
    const errorNames = {
      corrupted_archive: 'Corrupted',
      invalid_format: 'Invalid Format',
      password_protected: 'Password Protected',
      empty_archive: 'Empty',
      unsupported_format: 'Unsupported',
      io_error: 'I/O Error',
    };
    return errorNames[errorType] || errorType;
  }

  function formatFileSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }

  function filterFiles() {
    const searchTerm = searchInput.value.toLowerCase();
    const selectedError = errorFilter.value;

    filteredBadFiles = allBadFiles.filter(file => {
      const matchesSearch =
        file.file_name.toLowerCase().includes(searchTerm) ||
        file.path.toLowerCase().includes(searchTerm);

      const matchesError = !selectedError || file.error === selectedError;

      return matchesSearch && matchesError;
    });

    renderFiles();
  }

  async function deleteBadFile(id) {
    if (
      !confirm('Are you sure you want to remove this entry? This will not delete the actual file.')
    ) {
      return;
    }

    try {
      const response = await fetch(`/api/admin/bad-files?id=${id}`, {
        method: 'DELETE',
      });

      if (response.ok) {
        // Remove from local arrays
        allBadFiles = allBadFiles.filter(file => file.id !== id);
        filteredBadFiles = filteredBadFiles.filter(file => file.id !== id);

        // Update count
        totalCountEl.textContent = allBadFiles.length;

        // Re-render
        renderFiles();

        // Hide download button if count is now <= 50
        if (allBadFiles.length <= 50) {
          downloadCsvBtn.style.display = 'none';
        }
      } else {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
    } catch (error) {
      console.error('Error deleting bad file:', error);
      alert('Failed to delete bad file entry. Please try again.');
    }
  }

  function downloadCSV() {
    if (filteredBadFiles.length === 0) {
      alert('No files to download.');
      return;
    }

    // Create CSV content
    const headers = [
      'File Name',
      'Path',
      'Error',
      'File Size (bytes)',
      'Detected At',
      'Last Checked',
    ];
    const csvContent = [
      headers.join(','),
      ...filteredBadFiles.map(file =>
        [
          `"${file.file_name}"`,
          `"${file.path}"`,
          `"${getErrorDisplayName(file.error)}"`,
          file.file_size,
          `"${new Date(file.detected_at).toLocaleString()}"`,
          `"${new Date(file.last_checked).toLocaleString()}"`,
        ].join(',')
      ),
    ].join('\n');

    // Create and download file
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const link = document.createElement('a');
    const url = URL.createObjectURL(blob);
    link.setAttribute('href', url);
    link.setAttribute('download', `bad_files_${new Date().toISOString().slice(0, 10)}.csv`);
    link.style.visibility = 'hidden';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  }

  async function downloadFromServer() {
    const response = await fetch('/api/admin/bad-files/download');
    const blob = await response.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `bad_files_${new Date().toISOString().slice(0, 10)}.csv`;
    a.click();
  }
});
