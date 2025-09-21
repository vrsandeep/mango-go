import { checkAuth } from './auth.js';

document.addEventListener('DOMContentLoaded', async () => {
  const currentUser = await checkAuth();
  if (!currentUser) return;

  // --- State Management ---
  let state = {
    chapters: [],
    filteredChapters: [],
    selectedSeries: null,
    selectedProvider: null,
    selectedChapterRows: new Set(),
    sort: { by: 'chapter', dir: 'desc' }
  };

  // --- DOM Elements ---
  const providerSelect = document.getElementById('provider-select');
  const searchInput = document.getElementById('search-input');
  const searchResultsContainer = document.getElementById('search-results-container');
  const searchResultsGrid = document.getElementById('search-results-grid');
  const searchSummary = document.getElementById('search-summary');
  const searchCountEl = document.getElementById('search-count');
  const searchToggleIcon = document.getElementById('search-toggle-icon');
  const chapterView = document.getElementById('chapter-view');
  const chapterSeriesTitleEl = document.getElementById('chapter-series-title');
  const chapterCountEl = document.getElementById('chapter-count');
  const chapterTableBody = document.querySelector('#chapter-table tbody');
  const chapterTableHeaders = document.querySelectorAll('#chapter-table th');
  const downloadSelectedBtn = document.getElementById('download-selected-btn');
  const folderSelect = document.getElementById('folder-select');
  const customFolderPath = document.getElementById('custom-folder-path');
  const subscribeBtn = document.getElementById('subscribe-btn');
  const filterToggleBtn = document.getElementById('filter-toggle-btn');
  const filterPanel = document.getElementById('filter-panel');
  const applyFiltersBtn = document.getElementById('apply-filters-btn');
  const clearFiltersBtn = document.getElementById('clear-filters-btn');
  const selectAllBtn = document.getElementById('select-all-btn');
  const clearSelectionsBtn = document.getElementById('clear-selections-btn');

  // --- Core Functions ---
  const loadProviders = async () => {
    try {
      const response = await fetch('/api/providers');
      const providers = await response.json();
      if (providers) {
        providerSelect.innerHTML = providers.map(p => `<option value="${p.id}">${p.name}</option>`).join('');
        state.selectedProvider = providerSelect.value;
      }
    } catch (error) {
      console.error("Failed to load providers:", error);
    }
  };

  let searchTimeout;
  const handleSearch = () => {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(async () => {
      const query = searchInput.value;
      if (query.length < 3) {
        searchResultsContainer.style.display = 'none';
        return;
      }
      const response = await fetch(`/api/providers/${state.selectedProvider}/search?q=${encodeURIComponent(query)}`);
      const results = await response.json();
      renderSearchResults(results);
    }, 500);
  };

  const renderSearchResults = (results) => {
    searchResultsContainer.style.display = 'block';
    chapterView.style.display = 'none';
    searchCountEl.textContent = `${results.length} manga found`;
    searchResultsGrid.innerHTML = '';
    results.forEach(series => {
      const card = document.createElement('div');
      card.className = 'item-card';
      card.innerHTML = `<div class="thumbnail-container"><img class="thumbnail" src="${series.cover_url}" loading="lazy"></div><div class="item-title">${series.title}</div>`;
      card.addEventListener('click', () => handleSeriesSelect(series));
      searchResultsGrid.appendChild(card);
    });
  };

  const handleSeriesSelect = async (series) => {
    state.selectedSeries = series;
    searchResultsGrid.style.display = 'none';
    searchToggleIcon.textContent = '▶';
    // searchResultsContainer.style.display = 'none';
    chapterView.style.display = 'block';
    chapterSeriesTitleEl.textContent = series.title;
    chapterTableBody.innerHTML = '<tr><td colspan="6">Loading chapters...</td></tr>';

    const response = await fetch(`/api/providers/${state.selectedProvider}/series/${encodeURIComponent(series.identifier)}`);
    state.chapters = await response.json();
    applyFiltersAndSort();
  };

  const renderChapterTable = () => {
    chapterTableBody.innerHTML = '';
    state.filteredChapters.forEach((chapter, index) => {
      const row = document.createElement('tr');
      row.dataset.chapterIdentifier = chapter.identifier;
      row.dataset.index = index;
      const title = chapter.title || `Vol. ${chapter.volume || '?'} Ch. ${chapter.chapter || '?'}`;
      row.innerHTML = `
                    <td title="${title}">${title}</td>
                    <td>${chapter.pages}</td>
                    <td>${chapter.volume}</td>
                    <td>${chapter.chapter}</td>
                    <td>${chapter.language}</td>
                    <td>${new Date(chapter.published_at).toLocaleDateString()}</td>
                `;
      if (state.selectedChapterRows.has(chapter.identifier)) {
        row.classList.add('selected');
      }
      chapterTableBody.appendChild(row);
    });
    updateDownloadButtonState();
    updateTableHeaderIcons();
  };

  const applyFiltersAndSort = () => {
    // Apply Filters
    const titleFilter = document.querySelector('[data-filter="title"]').value;
    const langFilter = document.querySelector('[data-filter="language"]').value;
    state.filteredChapters = state.chapters.filter(ch => {
      const fullTitle = (ch.title || `Vol. ${ch.volume} Ch. ${ch.chapter}`).toLowerCase();
      return fullTitle.includes(titleFilter.toLowerCase()) && ch.language.toLowerCase().includes(langFilter.toLowerCase());
    });

    // Apply Sort
    const { key, dir } = state.sort;
    const dirMultiplier = dir === 'asc' ? 1 : -1;
    state.filteredChapters.sort((a, b) => {
      let valA = a[key];
      let valB = b[key];

      // Special handling for numeric and date fields
      if (key === 'pages' || key === 'volume' || key === 'chapter') {
        valA = parseFloat(valA) || 0;
        valB = parseFloat(valB) || 0;
      } else if (key === 'published_at') {
        valA = new Date(valA).getTime();
        valB = new Date(valB).getTime();
      } else {
        valA = (valA || '').toString().toLowerCase();
        valB = (valB || '').toString().toLowerCase();
      }

      if (valA < valB) return -1 * dirMultiplier;
      if (valA > valB) return 1 * dirMultiplier;
      return 0;
    });

    chapterCountEl.textContent = `${state.filteredChapters.length} of ${state.chapters.length} chapters found`;
    renderChapterTable();
  };

  const handleDownloadSelected = async () => {
    const selectedChapters = state.chapters.filter(ch => state.selectedChapterRows.has(ch.identifier));
    if (selectedChapters.length === 0) return;

    await fetch('/api/downloads/queue', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        series_title: state.selectedSeries.title,
        provider_id: state.selectedProvider,
        chapters: selectedChapters
      })
    });
    toast.success(`${selectedChapters.length} chapters added to download queue.`);
    clearSelections();
  };


  const loadFolders = async () => {
    try {
      const response = await fetch('/api/folders');
      const folders = await response.json();

      // Clear existing options except the default and manual option
      folderSelect.innerHTML = '<option value="">Default folder (series name)</option><option value="__manual__">Custom path...</option>';

      // Add folder options
      folders.forEach(folder => {
        const option = document.createElement('option');
        option.value = folder.path;
        option.textContent = folder.name;
        folderSelect.appendChild(option);
      });
    } catch (error) {
      console.error('Failed to load folders:', error);
    }
  };

  const handleFolderSelectChange = () => {
    if (folderSelect.value === '__manual__') {
      customFolderPath.style.display = 'inline-block';
      // Pre-fill with library path if not already set
      window.PathUtils.prefillCustomPath(customFolderPath);
      customFolderPath.focus();
    } else {
      customFolderPath.style.display = 'none';
      customFolderPath.value = '';
    }
  };


  const handleSubscribe = async () => {
    if (!state.selectedSeries) return;

    let folderPath = null;

    if (folderSelect.value === '__manual__') {
      // Use custom path if manual option is selected and input has value
      const customPath = customFolderPath.value.trim();
      if (customPath) {
        folderPath = window.PathUtils.sanitizePath(customPath);
        if (!folderPath) {
          toast.error('Invalid folder path. Please check for invalid characters.');
          return;
        }
      }
    } else if (folderSelect.value) {
      // Use selected folder path
      folderPath = folderSelect.value;
    }

    await fetch('/api/subscriptions', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        series_title: state.selectedSeries.title,
        series_identifier: state.selectedSeries.identifier,
        provider_id: state.selectedProvider,
        folder_path: folderPath
      })
    });

    const folderText = folderPath ? ` in folder "${folderPath}"` : '';
    toast.success(`Subscribed to ${state.selectedSeries.title}${folderText}.`);

    // Reset form
    folderSelect.value = '';
    customFolderPath.style.display = 'none';
    customFolderPath.value = '';
  };

  // --- UI Logic & Event Listeners ---
  const updateDownloadButtonState = () => {
    downloadSelectedBtn.disabled = state.selectedChapterRows.size === 0;
  };

  const clearSelections = () => {
    state.selectedChapterRows.clear();
    document.querySelectorAll('#chapter-table tbody tr.selected').forEach(row => row.classList.remove('selected'));
    updateDownloadButtonState();
  };

  const updateTableHeaderIcons = () => {
    chapterTableHeaders.forEach(th => {
      const icon = th.querySelector('.sort-icon');
      if (icon) icon.remove();
      if (th.dataset.sort === state.sort.key) {
        const newIcon = document.createElement('span');
        newIcon.className = 'sort-icon';
        newIcon.textContent = state.sort.dir === 'asc' ? ' ▲' : ' ▼';
        th.appendChild(newIcon);
      }
    });
  };

  // Event listeners for search, providers, buttons
  providerSelect.addEventListener('change', () => state.selectedProvider = providerSelect.value);
  searchInput.addEventListener('input', handleSearch);
  searchSummary.addEventListener('click', () => {
    const grid = searchResultsGrid;
    const isHidden = grid.style.display === 'none' || !grid.style.display;
    // const isHidden = searchResultsGrid.style.display === 'none';
    grid.style.display = isHidden ? 'grid' : 'none';
    // searchResultsGrid.style.display = isHidden ? 'grid' : 'none';
    searchToggleIcon.textContent = isHidden ? '▼' : '▶';
  });

  filterToggleBtn.addEventListener('click', () => filterPanel.style.display = filterPanel.style.display === 'none' ? 'block' : 'none');
  applyFiltersBtn.addEventListener('click', applyFiltersAndSort);
  clearFiltersBtn.addEventListener('click', () => {
    document.querySelectorAll('#filter-panel input').forEach(input => input.value = '');
    applyFiltersAndSort();
  });

  selectAllBtn.addEventListener('click', () => {
    state.filteredChapters.forEach(ch => state.selectedChapterRows.add(ch.identifier));
    document.querySelectorAll('#chapter-table tbody tr').forEach(row => row.classList.add('selected'));
    updateDownloadButtonState();
  });
  clearSelectionsBtn.addEventListener('click', clearSelections);
  downloadSelectedBtn.addEventListener('click', handleDownloadSelected);
  subscribeBtn.addEventListener('click', handleSubscribe);
  folderSelect.addEventListener('change', handleFolderSelectChange);

  // Sorting listener
  chapterTableHeaders.forEach(th => {
    th.addEventListener('click', () => {
      const sortKey = th.dataset.sort;
      if (!sortKey) return;

      if (state.sort.key === sortKey) {
        state.sort.dir = state.sort.dir === 'asc' ? 'desc' : 'asc';
      } else {
        state.sort.key = sortKey;
        state.sort.dir = 'desc'; // Default to desc for new columns
      }
      applyFiltersAndSort();
    });
  });

  // Multi-select logic
  let lastSelectedRowIndex = null;
  chapterTableBody.addEventListener('click', (e) => {
    const row = e.target.closest('tr');
    if (!row) return;

    const identifier = row.dataset.chapterIdentifier;
    const currentIndex = parseInt(row.dataset.index, 10);

    if (e.shiftKey && lastSelectedRowIndex !== null) {
      const start = Math.min(lastSelectedRowIndex, currentIndex);
      const end = Math.max(lastSelectedRowIndex, currentIndex);
      for (let i = start; i <= end; i++) {
        const id = state.filteredChapters[i].identifier;
        state.selectedChapterRows.add(id);
        chapterTableBody.querySelector(`[data-index="${i}"]`).classList.add('selected');
      }
    } else if (e.ctrlKey || e.metaKey) {
      if (state.selectedChapterRows.has(identifier)) {
        state.selectedChapterRows.delete(identifier);
        row.classList.remove('selected');
      } else {
        state.selectedChapterRows.add(identifier);
        row.classList.add('selected');
      }
    } else {
      const wasSelected = state.selectedChapterRows.has(identifier);
      clearSelections();
      if (!wasSelected) {
        state.selectedChapterRows.add(identifier);
        row.classList.add('selected');
      }
    }
    lastSelectedRowIndex = currentIndex;
    updateDownloadButtonState();
  });

  // --- Initialization ---
  loadProviders();
  window.PathUtils.loadLibraryPath();
  loadFolders();
});