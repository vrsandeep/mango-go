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
    sort: { by: 'chapter', dir: 'desc' },
  };

  // --- DOM Elements ---
  const providerSelect = document.getElementById('provider-select');
  const searchInput = document.getElementById('search-input');
  const clearSearchBtn = document.getElementById('clear-search-btn');
  const searchResultsSection = document.getElementById('results-section');
  const searchResultsGrid = document.getElementById('search-results-grid');
  const searchSummary = document.getElementById('search-summary');
  const searchCountEl = document.getElementById('search-count');
  const searchToggleBtn = document.getElementById('search-toggle-btn');
  const searchToggleIcon = document.getElementById('search-toggle-icon');
  const chaptersSection = document.getElementById('chapters-section');
  const chapterSeriesTitleEl = document.getElementById('chapter-series-title');
  const chapterCountEl = document.getElementById('chapter-count');
  const providerBadge = document.getElementById('provider-badge');
  const chapterTableBody = document.querySelector('#chapter-table tbody');
  const chapterTableHeaders = document.querySelectorAll('#chapter-table th');
  const downloadSelectedBtn = document.getElementById('download-selected-btn');
  const selectionCount = document.getElementById('selection-count');
  const subscribeBtn = document.getElementById('subscribe-btn');
  const selectAllBtn = document.getElementById('select-all-btn');
  const clearSelectionsBtn = document.getElementById('clear-selections-btn');

  // Floating panels
  const filtersBtn = document.getElementById('filters-btn');
  const settingsBtn = document.getElementById('settings-btn');
  const filtersPanel = document.getElementById('filters-panel');
  const settingsPanel = document.getElementById('settings-panel');
  const helpPanel = document.getElementById('help-panel');
  const filtersClose = document.getElementById('filters-close');
  const settingsClose = document.getElementById('settings-close');
  const helpClose = document.getElementById('help-close');
  const panelOverlay = document.getElementById('panel-overlay');
  const filterBadge = document.getElementById('filter-badge');

  // Filter elements
  const filterTitle = document.getElementById('filter-title');
  const filterLanguage = document.getElementById('filter-language');
  const filterVolumeMin = document.getElementById('filter-volume-min');
  const filterVolumeMax = document.getElementById('filter-volume-max');
  const filterChapterMin = document.getElementById('filter-chapter-min');
  const filterChapterMax = document.getElementById('filter-chapter-max');
  const applyFiltersBtn = document.getElementById('apply-filters-btn');
  const clearFiltersBtn = document.getElementById('clear-filters-btn');

  // Settings elements
  const folderPathRadios = document.querySelectorAll('input[name="folder-path"]');
  const customFolderPath = document.getElementById('custom-folder-path');
  const pathPreview = document.getElementById('path-preview');
  const previewText = document.getElementById('preview-text');

  // --- Core Functions ---
  const loadProviders = async () => {
    try {
      const response = await fetch('/api/providers');
      const providers = await response.json();
      if (providers) {
        providerSelect.innerHTML = providers
          .map(p => `<option value="${p.id}">${p.name}</option>`)
          .join('');
        state.selectedProvider = providerSelect.value;
      }
    } catch (error) {
      console.error('Failed to load providers:', error);
    }
  };

  let searchTimeout;

  const renderSearchResults = results => {
    searchResultsSection.style.display = 'block';
    chaptersSection.style.display = 'none';
    searchCountEl.textContent = results.length;
    searchSummary.style.display = 'block';
    searchResultsGrid.innerHTML = '';

    if (!results || results.length === 0) {
      searchResultsGrid.innerHTML = `
        <div class="empty-state">
          <i class="ph-bold ph-magnifying-glass"></i>
          <h3>No results found</h3>
          <p>Try adjusting your search terms or provider</p>
        </div>
      `;
      return;
    }

    results.forEach(series => {
      const card = document.createElement('div');
      card.className = 'item-card';
      card.innerHTML = `
        <div class="thumbnail-container">
          <img class="thumbnail" src="${series.cover_url}" loading="lazy" alt="${series.title}">
        </div>
        <div class="item-title">${series.title}</div>
      `;
      card.addEventListener('click', () => handleSeriesSelect(series));
      searchResultsGrid.appendChild(card);
    });
  };

  const handleSeriesSelect = async series => {
    state.selectedSeries = series;
    searchResultsSection.style.display = 'none';
    chaptersSection.style.display = 'block';
    chapterSeriesTitleEl.textContent = series.title;
    providerBadge.textContent = state.selectedProvider;
    chapterTableBody.innerHTML = '<tr><td colspan="6">Loading chapters...</td></tr>';

    // Clear previous selections when selecting a new series
    clearSelections();

    const response = await fetch(
      `/api/providers/${state.selectedProvider}/series/${encodeURIComponent(series.identifier)}`
    );
    state.chapters = await response.json();
    state.filteredChapters = [...state.chapters];
    populateLanguageFilter();
    applyFiltersAndSort();
    // Enable filters button after chapters are loaded
    updateFiltersButtonState();
  };

  const renderChapterTable = () => {
    chapterTableBody.innerHTML = '';
    chapterCountEl.textContent = `${state.filteredChapters.length} chapters`;

    if (state.filteredChapters.length === 0) {
      chapterTableBody.innerHTML = `
        <tr>
          <td colspan="6" class="empty-state">
            <i class="ph-bold ph-funnel"></i>
            <h3>No chapters match your filters</h3>
            <p>Try adjusting your filter criteria</p>
          </td>
        </tr>
      `;
      return;
    }

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

  const populateLanguageFilter = () => {
    // Get unique languages from chapters
    const languages = [...new Set(state.chapters.map(ch => ch.language))].sort();

    // Clear existing options except "All Languages"
    const languageSelect = document.getElementById('filter-language');
    languageSelect.innerHTML = '<option value="">All Languages</option>';

    // Add language options
    languages.forEach(lang => {
      const option = document.createElement('option');
      option.value = lang;
      option.textContent = lang;
      languageSelect.appendChild(option);
    });
  };

  const applyFiltersAndSort = () => {
    // Apply Filters
    const titleFilter = document.querySelector('[data-filter="title"]').value;
    const langFilter = document.querySelector('[data-filter="language"]').value;
    const volumeMin = document.querySelector('[data-filter="volume-min"]').value;
    const volumeMax = document.querySelector('[data-filter="volume-max"]').value;
    const chapterMin = document.querySelector('[data-filter="chapter-min"]').value;
    const chapterMax = document.querySelector('[data-filter="chapter-max"]').value;

    state.filteredChapters = state.chapters.filter(ch => {
      const fullTitle = (ch.title || `Vol. ${ch.volume} Ch. ${ch.chapter}`).toLowerCase();

      // Title filter
      if (titleFilter && !fullTitle.includes(titleFilter.toLowerCase())) {
        return false;
      }

      // Language filter
      if (langFilter && ch.language !== langFilter) {
        return false;
      }

      // Volume range filter
      const volume = parseFloat(ch.volume) || 0;
      if (volumeMin && volume < parseFloat(volumeMin)) {
        return false;
      }
      if (volumeMax && volume > parseFloat(volumeMax)) {
        return false;
      }

      // Chapter range filter
      const chapter = parseFloat(ch.chapter) || 0;
      if (chapterMin && chapter < parseFloat(chapterMin)) {
        return false;
      }
      if (chapterMax && chapter > parseFloat(chapterMax)) {
        return false;
      }

      return true;
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
    updateFilterBadge();
    closePanel(filtersPanel);
  };

  const loadFolders = async () => {
    try {
      const response = await fetch('/api/folders');
      const folders = await response.json();

      // Store folders for later use
      state.availableFolders = folders;
    } catch (error) {
      console.error('Failed to load folders:', error);
      state.availableFolders = [];
    }
  };

  // Search functionality
  const performSearch = async () => {
    const query = searchInput.value.trim();
    if (query.length < 3) {
      searchResultsSection.style.display = 'none';
      return;
    }

    if (!state.selectedProvider) {
      toast.error('Please select a provider first');
      return;
    }

    // Clear preview box when new search is performed
    clearPathPreview();
    // Clear previous selections when performing a new search
    clearSelections();
    // Clear selected series when performing a new search
    state.selectedSeries = null;
    state.chapters = [];
    // Disable filters button when performing a new search
    updateFiltersButtonState();

    try {
      const response = await fetch(
        `/api/providers/${state.selectedProvider}/search?q=${encodeURIComponent(query)}`
      );
      const results = await response.json();
      renderSearchResults(results);
    } catch (error) {
      console.error('Search failed:', error);
      toast.error('Search failed. Please try again.');
    }
  };

  const clearSearch = () => {
    searchInput.value = '';
    clearSearchBtn.style.display = 'none';
    searchResultsSection.style.display = 'none';
    chaptersSection.style.display = 'none';
    // Clear preview box in Download settings modal
    clearPathPreview();
    // Clear previous selections when clearing search
    clearSelections();
    // Disable filters button when search is cleared
    updateFiltersButtonState();
  };

  // Download functionality
  const downloadSelectedChapters = async () => {
    const selectedChapters = state.chapters.filter(chapter =>
      state.selectedChapterRows.has(chapter.identifier)
    );

    if (selectedChapters.length === 0) {
      toast.error('Please select chapters to download');
      return;
    }

    try {
      const response = await fetch('/api/downloads/queue', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          series_title: state.selectedSeries.title,
          provider_id: state.selectedProvider,
          chapters: selectedChapters,
        }),
      });

      if (response.ok) {
        toast.success(`${selectedChapters.length} chapters added to download queue`);
        clearSelections();
      } else {
        throw new Error('Failed to add chapters to queue');
      }
    } catch (error) {
      console.error('Download failed:', error);
      toast.error('Failed to add chapters to download queue');
    }
  };

  // --- Floating Panel Functions ---
  const openPanel = panel => {
    panel.classList.add('open');
    panelOverlay.classList.add('show');
    document.body.style.overflow = 'hidden';
  };

  const closePanel = panel => {
    panel.classList.remove('open');
    panelOverlay.classList.remove('show');
    document.body.style.overflow = '';
  };

  const closeAllPanels = () => {
    closePanel(filtersPanel);
    closePanel(settingsPanel);
    closePanel(helpPanel);
  };

  // --- Filter Functions ---
  const updateFilterBadge = () => {
    const activeFilters = [
      filterTitle.value,
      filterLanguage.value,
      filterVolumeMin.value,
      filterVolumeMax.value,
      filterChapterMin.value,
      filterChapterMax.value,
    ].filter(value => value.trim() !== '').length;

    if (activeFilters > 0) {
      filterBadge.textContent = activeFilters;
      filterBadge.style.display = 'inline';
      filtersBtn.classList.add('active');
    } else {
      filterBadge.style.display = 'none';
      filtersBtn.classList.remove('active');
    }
  };

  // Removed duplicate function - using applyFiltersAndSort instead

  const clearFilters = () => {
    filterTitle.value = '';
    filterLanguage.value = '';
    filterVolumeMin.value = '';
    filterVolumeMax.value = '';
    filterChapterMin.value = '';
    filterChapterMax.value = '';

    state.filteredChapters = [...state.chapters];
    renderChapterTable();
    updateFilterBadge();
  };

  // --- Folder Path Functions ---
  const handleFolderPathChange = () => {
    const isCustom = document.querySelector('input[name="folder-path"]:checked').value === 'custom';

    if (isCustom) {
      customFolderPath.style.display = 'block';
      pathPreview.style.display = 'block';
      // Don't prefill - let user enter custom path
      customFolderPath.value = '';
      customFolderPath.focus();
      updatePathPreview();
    } else {
      customFolderPath.style.display = 'none';
      pathPreview.style.display = 'none';
      customFolderPath.value = '';
    }
  };

  const updatePathPreview = () => {
    if (customFolderPath.value.trim()) {
      const libraryPath = window.PathUtils.getLibraryPath();
      const customPath = customFolderPath.value.trim();
      const fullPath = `${libraryPath}${customPath}`;
      previewText.textContent = fullPath;
    } else {
      previewText.textContent = 'Library Path + Series Name';
    }
  };

  const clearPathPreview = () => {
    pathPreview.style.display = 'none';
    previewText.textContent = '';
    customFolderPath.value = '';
    // Reset to default folder path option
    const defaultRadio = document.querySelector('input[name="folder-path"][value="default"]');
    if (defaultRadio) {
      defaultRadio.checked = true;
      customFolderPath.style.display = 'none';
    }
  };

  const handleSubscribe = async () => {
    if (!state.selectedSeries) return;

    // Check if subscription already exists
    try {
      const existingSubsResponse = await fetch(`/api/subscriptions?provider_id=${encodeURIComponent(state.selectedProvider)}`);
      const existingSubs = await existingSubsResponse.json();
      const alreadyExists = existingSubs.some(
        sub => sub.series_identifier === state.selectedSeries.identifier && sub.provider_id === state.selectedProvider
      );

      if (alreadyExists) {
        toast.error(`Subscription to "${state.selectedSeries.title}" already exists.`);
        return;
      }
    } catch (error) {
      console.error('Failed to check existing subscriptions:', error);
      // Continue with subscription attempt even if check fails
    }

    let folderPath = null;
    const selectedFolderPath = document.querySelector('input[name="folder-path"]:checked').value;

    if (selectedFolderPath === 'custom') {
      // Use custom path if custom option is selected and input has value
      const customPath = customFolderPath.value.trim();
      if (customPath) {
        folderPath = window.PathUtils.sanitizePath(customPath);
        if (!folderPath) {
          toast.error('Invalid folder path. Please check for invalid characters.');
          return;
        }
      }
    } else if (selectedFolderPath === 'default') {
      // Use default path (series name)
      folderPath = null;
    }

    try {
      const response = await fetch('/api/subscriptions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          series_title: state.selectedSeries.title,
          series_identifier: state.selectedSeries.identifier,
          provider_id: state.selectedProvider,
          folder_path: folderPath,
        }),
      });

      if (response.ok) {
        const folderText = folderPath ? ` in folder "${folderPath}"` : '';
        toast.success(`Subscribed to ${state.selectedSeries.title}${folderText}.`);

        // Reset form
        document.querySelector('input[name="folder-path"][value="default"]').checked = true;
        customFolderPath.style.display = 'none';
        customFolderPath.value = '';
      } else {
        const errorData = await response.json().catch(() => ({}));
        toast.error(errorData.error || 'Failed to create subscription.');
      }
    } catch (error) {
      console.error('Subscription failed:', error);
      toast.error('Failed to create subscription.');
    }
  };

  // --- UI Logic & Event Listeners ---
  const updateDownloadButtonState = () => {
    downloadSelectedBtn.disabled = state.selectedChapterRows.size === 0;
  };

  const updateFiltersButtonState = () => {
    // Enable filters button only when a manga is selected (has chapters)
    const hasChapters = state.selectedSeries && state.chapters && state.chapters.length > 0;
    filtersBtn.disabled = !hasChapters;
    if (hasChapters) {
      filtersBtn.classList.remove('disabled');
    } else {
      filtersBtn.classList.add('disabled');
    }
  };

  const clearSelections = () => {
    state.selectedChapterRows.clear();
    document
      .querySelectorAll('#chapter-table tbody tr.selected')
      .forEach(row => row.classList.remove('selected'));
    updateDownloadButtonState();
    updateSelectionCount();
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
  providerSelect.addEventListener('change', () => (state.selectedProvider = providerSelect.value));
  searchSummary.addEventListener('click', () => {
    const grid = searchResultsGrid;
    const isHidden = grid.style.display === 'none' || !grid.style.display;
    // const isHidden = searchResultsGrid.style.display === 'none';
    grid.style.display = isHidden ? 'grid' : 'none';
    // searchResultsGrid.style.display = isHidden ? 'grid' : 'none';
    searchToggleIcon.textContent = isHidden ? '▼' : '▶';
  });

  // This line is handled by the floating panel system - filtersBtn opens the filters panel
  applyFiltersBtn.addEventListener('click', applyFiltersAndSort);
  clearFiltersBtn.addEventListener('click', () => {
    document.querySelectorAll('#filters-panel input').forEach(input => (input.value = ''));
    applyFiltersAndSort();
  });

  selectAllBtn.addEventListener('click', () => {
    state.filteredChapters.forEach(ch => state.selectedChapterRows.add(ch.identifier));
    document
      .querySelectorAll('#chapter-table tbody tr')
      .forEach(row => row.classList.add('selected'));
    updateDownloadButtonState();
    updateSelectionCount();
  });
  clearSelectionsBtn.addEventListener('click', clearSelections);
  downloadSelectedBtn.addEventListener('click', downloadSelectedChapters);
  subscribeBtn.addEventListener('click', handleSubscribe);

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
  chapterTableBody.addEventListener('click', e => {
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
      updateSelectionCount();
    } else if (e.ctrlKey || e.metaKey) {
      toggleChapterSelection(identifier, row);
    } else {
      const wasSelected = state.selectedChapterRows.has(identifier);
      clearSelections();
      if (!wasSelected) {
        toggleChapterSelection(identifier, row);
      }
    }
    lastSelectedRowIndex = currentIndex;
    updateDownloadButtonState();
  });

  // --- New Event Listeners ---

  // Clear search button
  clearSearchBtn.addEventListener('click', clearSearch);

  // Search input changes
  searchInput.addEventListener('input', () => {
    clearSearchBtn.style.display = searchInput.value ? 'block' : 'none';
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(performSearch, 500);
  });

  // Floating panel buttons
  filtersBtn.addEventListener('click', () => {
    // Only open panel if filters button is enabled (manga is selected)
    if (!filtersBtn.disabled) {
      openPanel(filtersPanel);
    }
  });
  settingsBtn.addEventListener('click', () => openPanel(settingsPanel));

  // Panel close buttons
  filtersClose.addEventListener('click', () => closePanel(filtersPanel));
  settingsClose.addEventListener('click', () => closePanel(settingsPanel));

  // Help panel close button
  if (helpClose) {
    helpClose.addEventListener('click', () => closePanel(document.getElementById('help-panel')));
  }

  // Panel overlay
  panelOverlay.addEventListener('click', closeAllPanels);

  // Filter inputs
  [
    filterTitle,
    filterLanguage,
    filterVolumeMin,
    filterVolumeMax,
    filterChapterMin,
    filterChapterMax,
  ].forEach(input => {
    input.addEventListener('input', updateFilterBadge);
  });

  // Filter buttons (applyFilters handled by applyFiltersAndSort above)
  clearFiltersBtn.addEventListener('click', clearFilters);

  // Folder path radio buttons
  folderPathRadios.forEach(radio => {
    radio.addEventListener('change', handleFolderPathChange);
  });

  // Custom folder path input
  customFolderPath.addEventListener('input', updatePathPreview);

  // Selection count update
  const updateSelectionCount = () => {
    const count = state.selectedChapterRows.size;
    selectionCount.textContent = count;
    downloadSelectedBtn.disabled = count === 0;
  };

  // Chapter selection function
  const toggleChapterSelection = (identifier, row) => {
    if (state.selectedChapterRows.has(identifier)) {
      state.selectedChapterRows.delete(identifier);
      row.classList.remove('selected');
    } else {
      state.selectedChapterRows.add(identifier);
      row.classList.add('selected');
    }
    updateSelectionCount();
  };

  // --- Keyboard Shortcuts ---
  const keyboardShortcuts = {
    // Search shortcuts
    '/': () => {
      searchInput.focus();
      searchInput.select();
    },
    Escape: () => {
      if (searchInput.value) {
        searchInput.value = '';
        clearSearch();
      } else if (document.querySelector('.floating-panel.open')) {
        closeAllPanels();
      }
    },

    // Panel shortcuts
    f: () => {
      // Only open filters panel if enabled (manga is selected)
      if (!filtersBtn.disabled) {
        openPanel(filtersPanel);
      }
    },
    s: () => openPanel(settingsPanel),

    // Action shortcuts (handled in the main Enter key below)

    // Selection shortcuts
    a: e => {
      e.preventDefault();
      selectAllChapters();
    },
    d: e => {
      e.preventDefault();
      deselectAllChapters();
    },

    // Download shortcut
    Enter: e => {
      if (e.ctrlKey || e.metaKey) {
        if (state.selectedChapterRows.size > 0) {
          e.preventDefault();
          downloadSelectedChapters();
        }
      } else if (searchInput === document.activeElement) {
        searchInput.blur();
        performSearch();
      }
    },
  };

  // Handle keyboard shortcuts
  document.addEventListener('keydown', e => {
    // Don't trigger shortcuts when typing in inputs, textareas, or contenteditable elements
    if (
      e.target.tagName === 'INPUT' ||
      e.target.tagName === 'TEXTAREA' ||
      e.target.contentEditable === 'true' ||
      e.target.closest('.floating-panel')
    ) {
      return;
    }

    // For Cmd/Ctrl shortcuts, only proceed if modifier is pressed
    const isModifierPressed = e.ctrlKey || e.metaKey;
    const key = e.key.toLowerCase();

    // Handle Cmd/Ctrl shortcuts (A, D, Enter)
    if (isModifierPressed && (key === 'a' || key === 'd' || key === 'enter')) {
      const shortcut = keyboardShortcuts[key];
      if (shortcut) {
        shortcut(e);
      }
      return;
    }

    // Handle other shortcuts (non-modifier keys)
    if (!isModifierPressed) {
      const shortcut = keyboardShortcuts[key];
      if (shortcut) {
        shortcut(e);
      }
    }
  });

  // Helper functions for keyboard shortcuts
  function selectAllChapters() {
    // Use the same logic as the Select All button
    if (!state.filteredChapters || state.filteredChapters.length === 0) {
      return;
    }
    state.filteredChapters.forEach(ch => state.selectedChapterRows.add(ch.identifier));
    document
      .querySelectorAll('#chapter-table tbody tr')
      .forEach(row => row.classList.add('selected'));
    updateDownloadButtonState();
    updateSelectionCount();
  }

  function deselectAllChapters() {
    // Use the same logic as the Clear Selections button
    clearSelections();
  }

  // Add keyboard shortcut hints to UI
  function addKeyboardHints() {
    // Add tooltips or hints for keyboard shortcuts
    const searchInput = document.getElementById('search-input');
    if (searchInput) {
      searchInput.placeholder = 'Search manga... (Press / to focus)';
    }

    // Add keyboard shortcut indicators to buttons
    if (filtersBtn) {
      filtersBtn.title = 'Open Filters (F)';
    }

    if (settingsBtn) {
      settingsBtn.title = 'Open Settings (S)';
    }
  }

  // Show keyboard shortcuts help
  function showKeyboardHelp() {
    const helpPanel = document.getElementById('help-panel');
    openPanel(helpPanel);
  }

  // Add help button to show keyboard shortcuts
  function addHelpButton() {
    const quickActions = document.querySelector('.quick-actions');
    if (quickActions) {
      const helpBtn = document.createElement('button');
      helpBtn.className = 'action-btn';
      helpBtn.innerHTML = '❓ Help';
      helpBtn.title = 'Show keyboard shortcuts (?)';
      helpBtn.addEventListener('click', showKeyboardHelp);
      quickActions.appendChild(helpBtn);
    }
  }

  // Add keyboard shortcut for help (?)
  document.addEventListener('keydown', e => {
    if (e.key === '?' && !e.target.matches('input, textarea, [contenteditable]')) {
      e.preventDefault();
      showKeyboardHelp();
    }
  });

  // --- Initialization ---
  loadProviders();
  window.PathUtils.loadLibraryPath();
  loadFolders();
  addKeyboardHints();
  addHelpButton();
  // Initialize filters button as disabled
  updateFiltersButtonState();
});
