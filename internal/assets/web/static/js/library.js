import { checkAuth } from './auth.js';

document.addEventListener('DOMContentLoaded', async () => {
  const currentUser = await checkAuth();
  if (!currentUser) return;

  const cardsGrid = document.getElementById('cards-grid');
  const pageTitleEl = document.getElementById('page-title');
  const breadcrumbEl = document.getElementById('breadcrumb-container');
  const folderThumb = document.getElementById('folder-thumb');
  const searchInput = document.getElementById('search-input');
  const sortBySelect = document.getElementById('sort-by');
  const sortDirBtn = document.getElementById('sort-dir-btn');
  const paginationContainer = document.getElementById('pagination-container');
  const editFolderBtn = document.getElementById('edit-folder-btn');
  const editFolderModal = document.getElementById('edit-folder-modal');
  const modalCloseBtn = document.getElementById('modal-close-btn');
  const tagsContainer = document.getElementById('tags-container');
  const tagInput = document.getElementById('tag-input');
  const autocompleteSuggestions = document.getElementById('autocomplete-suggestions');
  const modalSaveBtn = document.getElementById('modal-save-btn');
  const modalCancelBtn = document.getElementById('modal-cancel-btn');
  const coverFileInput = document.getElementById('cover-file-input');
  const selectedFileInfo = document.getElementById('selected-file-info');
  const fileName = document.getElementById('file-name');
  const removeFileBtn = document.getElementById('remove-file-btn');
  const totalCountEl = document.getElementById('total-count');
  const markAllReadBtn = document.getElementById('mark-all-read-btn');
  const markAllUnreadBtn = document.getElementById('mark-all-unread-btn');

  // --- State Management ---
  let state = {
    currentFolderId: null,
    currentTagId: null,
    currentPage: 1,
    search: '',
    sortBy: null,
    sortDir: null,
    isLoading: false,
    totalItems: 0,
    perPage: 100,
  };
  let allTags = [];
  let currentFolderTags = [];

  // --- Core Functions ---

  // AniList API integration
  const fetchAniListData = async title => {
    if (!title) return null;

    try {
      // Clean up the title for better search results
      const cleanTitle = title.replace(/[^\w\s-]/g, '').trim();
      if (!cleanTitle) return null;

      const query = `
        query ($search: String) {
          Media(search: $search, type: MANGA) {
            id
            title {
              romaji
              english
            }
            coverImage {
              large
            }
            siteUrl
          }
        }
      `;

      const variables = { search: cleanTitle };

      const response = await fetch('https://graphql.anilist.co', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Accept: 'application/json',
        },
        body: JSON.stringify({
          query: query,
          variables: variables,
        }),
      });

      if (!response.ok) {
        console.error(`AniList API error: ${response.status} ${response.statusText}`);
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();

      if (data.errors) {
        console.warn('AniList API returned errors:', data.errors);
        return null;
      }

      return data.data?.Media || null;
    } catch (error) {
      console.error('Error fetching AniList data for title:', title, error);
      return null;
    }
  };

  // Async AniList button loader - runs after page load without blocking
  const loadAniListButtonsAsync = () => {
    // Use setTimeout to ensure this runs after the main rendering is complete
    setTimeout(async () => {
      const pageTitle = document.getElementById('page-title');
      if (!pageTitle) return;

      // Skip if button already exists
      if (document.querySelector('.anilist-button')) return;

      const title = pageTitle.textContent.trim();
      if (!title || title === 'Library') return; // Skip for generic library page

      try {
        const anilistData = await fetchAniListData(title);
        if (anilistData?.siteUrl) {
          addAniListButtonToHeader(anilistData.siteUrl);
        }
      } catch (error) {
        console.warn('Failed to load AniList data for:', title, error);
      }
    }, 100); // Small delay to ensure DOM is ready
  };

  // Helper function to add AniList button to page header
  const addAniListButtonToHeader = anilistUrl => {
    const pageTitle = document.getElementById('page-title');
    if (!pageTitle) return;

    // Create a container div for the title and button
    const titleContainer = document.createElement('div');
    titleContainer.className = 'title-container';

    // Move the title into the container
    pageTitle.parentNode.insertBefore(titleContainer, pageTitle);
    titleContainer.appendChild(pageTitle);

    // Create the AniList button with icon
    const button = document.createElement('a');
    button.href = anilistUrl;
    button.target = '_blank';
    button.className = 'anilist-button';

    // Create icon element
    const icon = document.createElement('img');
    icon.src = '/static/images/anilist-icon.svg';
    icon.alt = 'AniList';
    icon.className = 'anilist-icon';

    button.appendChild(icon);

    // Add button to the title container
    titleContainer.appendChild(button);
  };

  // Get the current folder ID from the URL path.
  const getFolderIdFromUrl = () => {
    const parts = window.location.pathname.split('/folder/');
    return parts.length > 1 ? parts[1] : null;
  };
  const getTagIdFromUrl = () => {
    const parts = window.location.pathname.split('/tags/');
    return parts.length > 1 ? parts[1] : null;
  };
  const getTagNameFromId = async id => {
    return allTags.find(tag => tag.id === parseInt(id)).name;
  };

  // Fetches and renders the breadcrumb navigation.
  const renderBreadcrumb = async () => {
    if (!state.currentFolderId) {
      breadcrumbEl.innerHTML = '';
      return;
    }
    if (breadcrumbEl.innerHTML != '') {
      return;
    }
    const url = `/api/browse/breadcrumb?folderId=${state.currentFolderId}`;
    const response = await fetch(url);
    const path = await response.json();

    let html = '<a href="/library">Library</a>';
    path.forEach(folder => {
      html += ` / <a href="/library/folder/${folder.id}">${folder.name}</a>`;
    });
    breadcrumbEl.innerHTML = html;
  };

  // Renders the grid with folders first, then chapters.
  const renderGrid = data => {
    cardsGrid.innerHTML = '';
    if (
      (!data.subfolders || data.subfolders.length === 0) &&
      (!data.chapters || data.chapters.length === 0)
    ) {
      cardsGrid.innerHTML = '<p>This folder is empty.</p>';
      return;
    }

    if (data.subfolders && data.subfolders.length > 0) {
      cardsGrid.insertAdjacentHTML('beforeend', '<h3 class="grid-section-header">Folders</h3>');
      data.subfolders.forEach(folder => cardsGrid.appendChild(createFolderCard(folder)));
    }
    if (data.chapters && data.chapters.length > 0) {
      cardsGrid.insertAdjacentHTML('beforeend', '<h3 class="grid-section-header">Chapters</h3>');
      data.chapters.forEach(chapter => cardsGrid.appendChild(createChapterCard(chapter)));
    }

    // Load AniList buttons asynchronously after rendering
    loadAniListButtonsAsync();
  };

  // Creates an HTML card for a folder.
  const createFolderCard = folder => {
    const card = document.createElement('a');
    card.href = `/library/folder/${folder.id}`;
    card.className = 'item-card folder';
    // Calculate progress for the folder
    const progressPercent =
      folder.total_chapters > 0 ? (folder.read_chapters / folder.total_chapters) * 100 : 0;
    card.innerHTML = `
            <div class="thumbnail-container">
                <img class="thumbnail" src="${folder.thumbnail || '/static/images/logo.svg'}" loading="lazy" alt="Cover for ${folder.name}">
            </div>
            <div class="item-title" title="${folder.name}">${folder.name}</div>
            <div class="progress-bar-container">
              <div class="progress-bar" style="width: ${progressPercent}%;"></div>
            </div>
        `;
    return card;
  };

  // Creates an HTML card for a chapter.
  const createChapterCard = chapter => {
    const card = document.createElement('a');
    const progressPercent = chapter.progress_percent || 0;
    card.href = `/reader/series/${chapter.folder_id}/chapters/${chapter.id}`; // Note: Reader URL might need adjustment
    card.className = 'item-card';
    const title = chapter.path.split(/[\\\\/]/).pop();
    card.innerHTML = `
            <div class="thumbnail-container">
                <img class="thumbnail" src="${chapter.thumbnail || ''}" loading="lazy" alt="Cover for ${title}">
            </div>
            <div class="item-title" title="${title}">${title}</div>
            <div class="progress-bar-container">
                <div class="progress-bar" style="width: ${progressPercent}%;"></div>
            </div>
        `;
    return card;
  };

  const saveFolderSettings = async () => {
    await fetch(`/api/folders/${state.currentFolderId}/settings`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ sort_by: state.sortBy, sort_dir: state.sortDir }),
    });
  };

  // Fetch folder settings and update state
  const loadFolderSettings = async () => {
    if (!state.currentFolderId) return;
    if (state.sortBy != null && state.sortDir != null) return;

    try {
      const response = await fetch(`/api/folders/${state.currentFolderId}/settings`);
      if (response.ok) {
        const settings = await response.json();
        state.sortBy = settings.sort_by || 'auto';
        state.sortDir = settings.sort_dir || 'asc';

        // Update UI elements to reflect the loaded settings
        sortBySelect.value = state.sortBy;
        sortDirBtn.textContent = state.sortDir === 'asc' ? '▲' : '▼';
      }
    } catch (error) {
      console.error('Failed to load folder settings:', error);
    }
  };

  // Main function to fetch all data and render the page.
  const loadFolderContents = async () => {
    if (state.isLoading) return;
    state.isLoading = true;
    cardsGrid.innerHTML = '<p>Loading...</p>';

    try {
      await renderBreadcrumb();
      await loadFolderSettings();

      const params = new URLSearchParams({
        page: state.currentPage,
        per_page: state.perPage,
        search: state.search,
        sort_by: state.sortBy,
        sort_dir: state.sortDir,
      });
      if (state.currentFolderId) {
        params.set('folderId', state.currentFolderId);
      }
      if (state.currentTagId) {
        params.set('tagId', state.currentTagId);
      }

      const response = await fetch(`/api/browse?${params.toString()}`);

      if (!response.ok) {
        console.error(`Browse API error: ${response.status} ${response.statusText}`);
        throw new Error(`Browse API error: ${response.status}`);
      }

      const data = await response.json();

      if (data.current_folder) {
        pageTitleEl.textContent = data.current_folder.name;
      } else if (state.currentTagId) {
        const tagName = await getTagNameFromId(state.currentTagId);
        pageTitleEl.textContent = `Tag: ${tagName}`;
        document.title = `Tag: ${tagName} - Mango`;
      } else {
        pageTitleEl.textContent = 'Library';
      }
      document.title = `${pageTitleEl.textContent} - Mango`;
      folderThumb.src = data.current_folder ? data.current_folder.thumbnail : '';
      folderThumb.style.display = data.current_folder ? 'block' : 'none';

      // Show the Edit button only when viewing a specific folder
      editFolderBtn.style.display = data.current_folder ? 'block' : 'none';
      renderTags(data.current_folder ? data.current_folder.tags : []);

      renderGrid(data);

      state.totalItems = parseInt(response.headers.get('X-Total-Count') || '0', 10);
      totalCountEl.textContent = `${state.totalItems}`;
      renderPagination();
    } catch (error) {
      console.error('Error loading folder contents:', error);
      cardsGrid.innerHTML = '<p>Error loading content. Please try again.</p>';
    } finally {
      state.isLoading = false;
    }
  };

  const renderPagination = () => {
    paginationContainer.innerHTML = '';
    const totalPages = Math.ceil(state.totalItems / state.perPage);
    if (totalPages <= 1) return;

    const createButton = (text, page, isDisabled = false, isActive = false) => {
      const btn = document.createElement('button');
      btn.className = 'pagination-btn';
      btn.innerHTML = text;
      if (isDisabled) btn.classList.add('disabled');
      if (isActive) btn.classList.add('active');
      btn.addEventListener('click', () => {
        state.currentPage = page;
        loadFolderContents();
        btn.classList.add('active');
        const siblings = Array.from(paginationContainer.children);
        siblings.forEach(sibling => {
          if (sibling !== btn) sibling.classList.remove('active');
        });
      });
      return btn;
    };

    paginationContainer.appendChild(createButton('&laquo;', 1, state.currentPage === 1));
    paginationContainer.appendChild(
      createButton('&lsaquo;', state.currentPage - 1, state.currentPage === 1)
    );

    const pageNumbers = [];
    // Always show first page
    pageNumbers.push(1);

    // Ellipsis logic
    if (state.currentPage > 4) {
      pageNumbers.push('...');
    }

    // Window of pages around current page
    for (
      let i = Math.max(2, state.currentPage - 2);
      i <= Math.min(totalPages - 1, state.currentPage + 2);
      i++
    ) {
      pageNumbers.push(i);
    }

    if (state.currentPage < totalPages - 3) {
      pageNumbers.push('...');
    }

    // Always show last page
    if (totalPages > 1) pageNumbers.push(totalPages);

    // Render buttons from unique page numbers
    [...new Set(pageNumbers)].forEach(num => {
      if (num === '...') {
        const ellipsis = document.createElement('span');
        ellipsis.className = 'pagination-ellipsis';
        ellipsis.textContent = '...';
        paginationContainer.appendChild(ellipsis);
      } else {
        paginationContainer.appendChild(createButton(num, num, false, state.currentPage === num));
      }
    });

    paginationContainer.appendChild(
      createButton('&rsaquo;', state.currentPage + 1, state.currentPage === totalPages)
    );
    paginationContainer.appendChild(
      createButton('&raquo;', totalPages, state.currentPage === totalPages)
    );
  };

  // --- Tagging Logic ---
  const loadAllTags = async () => {
    try {
      const response = await fetch('/api/tags');
      allTags = (await response.json()) || [];
    } catch (e) {
      console.error('Failed to load all tags:', e);
    }
  };

  const renderTags = tags => {
    currentFolderTags = tags || [];
    tagsContainer.innerHTML = '';
    currentFolderTags.forEach(tag => {
      const tagEl = document.createElement('div');
      tagEl.className = 'tag';
      tagEl.innerHTML = `<span>${tag.name}</span><span class="tag-remove-btn" data-tag-id="${tag.id}">&times;</span>`;
      tagsContainer.appendChild(tagEl);
    });

    // Update icon visibility based on whether there are tags
    const tagsDisplay = document.querySelector('.tags-display');
    if (tagsDisplay) {
      if (currentFolderTags.length === 0) {
        tagsDisplay.classList.add('no-tags');
      } else {
        tagsDisplay.classList.remove('no-tags');
      }
    }
  };

  const addTag = async tagName => {
    const normalizedTagName = tagName.trim().toLowerCase();
    if (
      normalizedTagName === '' ||
      !state.currentFolderId ||
      currentFolderTags.some(t => t.name === normalizedTagName)
    ) {
      tagInput.value = '';
      return;
    }
    const response = await fetch(`/api/folders/${state.currentFolderId}/tags`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: normalizedTagName }),
    });
    if (response.ok) {
      tagInput.value = '';
      autocompleteSuggestions.style.display = 'none';
      const newTag = await response.json();
      currentFolderTags.push(newTag);
      renderTags(currentFolderTags);
    }
  };

  const removeTag = async tagId => {
    await fetch(`/api/folders/${state.currentFolderId}/tags/${tagId}`, {
      method: 'DELETE',
    });
    currentFolderTags = currentFolderTags.filter(t => t.id != tagId);
    renderTags(currentFolderTags);
  };

  const handleSearch = () => {
    state.search = searchInput.value.trim();
    state.currentPage = 1;
    loadFolderContents();
  };

  const handleSaveChanges = async () => {
    const file = coverFileInput.files[0];
    if (!file) {
      // In the future, you could handle other fields here.
      // For now, if no file, just close the modal.
      editFolderModal.style.display = 'none';
      return;
    }

    const formData = new FormData();
    formData.append('cover_file', file);

    try {
      const response = await fetch(`/api/folders/${state.currentFolderId}/cover`, {
        method: 'POST',
        body: formData, // The browser will set the Content-Type header automatically
      });

      if (response.ok) {
        editFolderModal.style.display = 'none';
        toast.success('Cover uploaded successfully!');
        loadFolderContents(); // Reload to show the new cover
      } else {
        const errorData = await response.json();
        toast.error(`Error uploading cover: ${errorData.error}`);
      }
    } catch (err) {
      toast.error('An unexpected error occurred during upload.');
    }
  };

  // Mark all chapters as read or unread.
  const markAllAs = async read => {
    const response = await fetch(`/api/folders/${state.currentFolderId}/mark-all-as`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ read }),
    });
    if (response.ok) {
      // Dismiss the modal
      editFolderModal.style.display = 'none';
      // Show success toast
      toast.success(`All chapters marked as ${read ? 'read' : 'unread'}`);
      // Reload folder contents
      loadFolderContents();
    } else {
      toast.error(`Failed to mark all chapters as ${read ? 'read' : 'unread'}`);
    }
  };
  // --- Event Listeners ---
  editFolderBtn.addEventListener('click', () => {
    if (state.currentFolderId) {
      editFolderModal.style.display = 'flex';
      coverFileInput.value = ''; // Clear previous selection
      selectedFileInfo.style.display = 'none'; // Hide file info
      // Update save button text to be more intuitive
      modalSaveBtn.innerHTML = '<i class="ph-bold ph-upload"></i> Upload Cover';
    }
  });

  // Handle file selection
  coverFileInput.addEventListener('change', e => {
    const file = e.target.files[0];
    if (file) {
      fileName.textContent = file.name;
      selectedFileInfo.style.display = 'flex';
    } else {
      selectedFileInfo.style.display = 'none';
    }
  });

  // Handle remove file button
  removeFileBtn.addEventListener('click', () => {
    coverFileInput.value = '';
    selectedFileInfo.style.display = 'none';
  });
  modalCancelBtn.addEventListener('click', () => (editFolderModal.style.display = 'none'));
  modalSaveBtn.addEventListener('click', handleSaveChanges);
  modalCloseBtn.addEventListener('click', () => (editFolderModal.style.display = 'none'));

  // Close modal when clicking outside
  editFolderModal.addEventListener('click', e => {
    if (e.target === editFolderModal) {
      editFolderModal.style.display = 'none';
    }
  });

  // Close modal when pressing ESC key
  document.addEventListener('keydown', e => {
    if (e.key === 'Escape' && editFolderModal.style.display === 'flex') {
      editFolderModal.style.display = 'none';
    }
  });
  tagsContainer.addEventListener('click', e => {
    if (e.target.classList.contains('tag-remove-btn')) {
      removeTag(e.target.dataset.tagId);
    }
  });
  tagInput.addEventListener('keydown', e => {
    if (e.key === 'Enter') {
      e.preventDefault();
      addTag(tagInput.value);
    }
  });
  tagInput.addEventListener('input', () => {
    const query = tagInput.value.trim().toLowerCase();
    if (query === '') {
      autocompleteSuggestions.style.display = 'none';
      return;
    }
    const suggestions = allTags.filter(
      tag =>
        tag.name.toLowerCase().includes(query) && !currentFolderTags.some(t => t.name === tag.name)
    );
    autocompleteSuggestions.innerHTML = '';
    suggestions.slice(0, 5).forEach(suggestion => {
      const suggestionEl = document.createElement('div');
      suggestionEl.className = 'autocomplete-suggestion';
      suggestionEl.textContent = suggestion.name;
      suggestionEl.addEventListener('click', () => {
        addTag(suggestion.name);
        autocompleteSuggestions.style.display = 'none';
      });
      autocompleteSuggestions.appendChild(suggestionEl);
    });
    autocompleteSuggestions.style.display = suggestions.length > 0 ? 'block' : 'none';
  });

  let searchTimeout;
  searchInput.addEventListener('input', () => {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(handleSearch, 300);
  });

  sortBySelect.addEventListener('change', () => {
    state.sortBy = sortBySelect.value;
    state.sortDir = state.sortDir === 'asc' ? 'desc' : 'asc';
    saveFolderSettings();
    loadFolderContents();
  });

  sortDirBtn.addEventListener('click', () => {
    state.sortBy = sortBySelect.value;
    state.sortDir = state.sortDir === 'asc' ? 'desc' : 'asc';
    sortDirBtn.textContent = state.sortDir === 'asc' ? '▲' : '▼';
    saveFolderSettings();
    loadFolderContents();
  });

  markAllReadBtn.addEventListener('click', async () => {
    markAllAs(true);
  });
  markAllUnreadBtn.addEventListener('click', async () => {
    markAllAs(false);
  });

  const init = async () => {
    state.currentFolderId = getFolderIdFromUrl();
    state.currentTagId = getTagIdFromUrl();
    await loadAllTags(); // Load tags for autocomplete
    await loadFolderContents();
  };

  init();
});
