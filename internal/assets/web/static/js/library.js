document.addEventListener('DOMContentLoaded', async () => {
  const currentUser = await checkAuth();
  if (!currentUser) return;

  const cardsGrid = document.getElementById('cards-grid');
  const pageTitleEl = document.getElementById('page-title');
  const breadcrumbEl = document.getElementById('breadcrumb-container');
  const searchInput = document.getElementById('search-input');
  const sortBySelect = document.getElementById('sort-by');
  const sortDirBtn = document.getElementById('sort-dir-btn');
  const paginationContainer = document.getElementById('pagination-container');

  // --- State Management ---
  let state = {
    currentFolderId: null,
    currentPage: 1,
    search: '',
    sortBy: 'auto',
    sortDir: 'asc',
    isLoading: false,
    totalItems: 0,
    perPage: 100
  };

  // --- Core Functions ---

  // Get the current folder ID from the URL path.
  const getFolderIdFromUrl = () => {
    const parts = window.location.pathname.split('/folder/');
    return parts.length > 1 ? parts[1] : null;
  };

  // Fetches and renders the breadcrumb navigation.
  const renderBreadcrumb = async () => {
    if (!state.currentFolderId) {
      breadcrumbEl.innerHTML = '';
      return;
    }
    const url = `/api/browse/breadcrumb?folderId=${state.currentFolderId}`;
    const response = await fetch(url);
    const path = await response.json();

    breadcrumbEl.innerHTML = '<a href="/library">Home</a>';
    path.forEach(folder => {
      breadcrumbEl.innerHTML += ` / <a href="/library/folder/${folder.id}">${folder.name}</a>`;
    });
  };

  // Renders the grid with folders first, then chapters.
  const renderGrid = (data) => {
    cardsGrid.innerHTML = '';
    if ((!data.subfolders || data.subfolders.length === 0) && (!data.chapters || data.chapters.length === 0)) {
      cardsGrid.innerHTML = '<p>This folder is empty.</p>';
      return;
    }

    if (data.subfolders && data.subfolders.length > 0) {
      const header = document.createElement('h3');
      header.className = 'grid-section-header';
      header.textContent = 'Folders';
      cardsGrid.appendChild(header);
      data.subfolders.forEach(folder => cardsGrid.appendChild(createFolderCard(folder)));
    }
    if (data.chapters && data.chapters.length > 0) {
      const header = document.createElement('h3');
      header.className = 'grid-section-header';
      header.textContent = 'Chapters';
      cardsGrid.appendChild(header);
      data.chapters.forEach(chapter => cardsGrid.appendChild(createChapterCard(chapter)));
    }
  };

  // Creates an HTML card for a folder.
  const createFolderCard = (folder) => {
    const card = document.createElement('a');
    card.href = `/library/folder/${folder.id}`;
    card.className = 'item-card folder';
    card.innerHTML = `
            <div class="thumbnail-container">
                <img class="thumbnail" src="${folder.thumbnail || '/static/images/logo.svg'}"></div>
            <div class="item-title" title="${folder.name}">${folder.name}</div>
        `;
    return card;
  };

  // Creates an HTML card for a chapter.
  const createChapterCard = (chapter) => {
    const card = document.createElement('a');
    const progressPercent = chapter.progress_percent || 0;
    card.href = `/reader/series/${chapter.folder_id}/chapters/${chapter.id}`; // Note: Reader URL might need adjustment
    card.className = 'item-card';
    card.innerHTML = `
            <div class="thumbnail-container">
                <img class="thumbnail" src="${chapter.thumbnail || ''}" loading="lazy">
            </div>
            <div class="item-title" title="${chapter.path.split(/[\\\\/]/).pop()}">${chapter.path.split(/[\\\\/]/).pop()}</div>
            <div class="progress-bar-container">
                <div class="progress-bar" style="width: ${progressPercent}%;"></div>
            </div>
        `;
    return card;
  };

  // Main function to fetch all data and render the page.
  const loadFolderContents = async () => {
    if (state.isLoading) return;
    state.isLoading = true;
    cardsGrid.innerHTML = '<p>Loading...</p>';

    await renderBreadcrumb();

    const params = new URLSearchParams({
      page: state.currentPage,
      per_page: state.perPage,
      search: state.search,
      sort_by: state.sortBy,
      sort_dir: state.sortDir
    });
    if (state.currentFolderId) {
      params.set('folderId', state.currentFolderId);
    }

    const response = await fetch(`/api/browse?${params.toString()}`);
    const data = await response.json();

    pageTitleEl.textContent = data.current_folder ? data.current_folder.name : 'Library';
    document.title = `${pageTitleEl.textContent} - Mango`;

    renderGrid(data);

    state.totalItems = parseInt(response.headers.get('X-Total-Count') || '0', 10);
    renderPagination();

    state.isLoading = false;
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
        loadCards(true);
        btn.classList.add('active');
        const siblings = Array.from(paginationContainer.children);
        siblings.forEach(sibling => {
          if (sibling !== btn) sibling.classList.remove('active');
        });
      });
      return btn;
    };

    paginationContainer.appendChild(createButton('&laquo;', 1, state.currentPage === 1));
    paginationContainer.appendChild(createButton('&lsaquo;', state.currentPage - 1, state.currentPage === 1));

    const pageNumbers = [];
    // Always show first page
    pageNumbers.push(1);

    // Ellipsis logic
    if (state.currentPage > 4) {
      pageNumbers.push('...');
    }

    // Window of pages around current page
    for (let i = Math.max(2, state.currentPage - 2); i <= Math.min(totalPages - 1, state.currentPage + 2); i++) {
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

    paginationContainer.appendChild(createButton('&rsaquo;', state.currentPage + 1, state.currentPage === totalPages));
    paginationContainer.appendChild(createButton('&raquo;', totalPages, state.currentPage === totalPages));
  };

  const handleSearch = () => {
    state.search = searchInput.value.trim();
    state.currentPage = 1;
    loadFolderContents();
  };

  // --- Event Listeners ---
  let searchTimeout;
  searchInput.addEventListener('input', () => {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(handleSearch, 300);
  });

  sortBySelect.addEventListener('change', () => {
    state.sortBy = sortBySelect.value;
    loadFolderContents();
  });

  sortDirBtn.addEventListener('click', () => {
    state.sortDir = state.sortDir === 'asc' ? 'desc' : 'asc';
    sortDirBtn.textContent = state.sortDir === 'asc' ? '▲' : '▼';
    loadFolderContents();
  });
  const init = () => {
    state.currentFolderId = getFolderIdFromUrl();
    loadFolderContents();
  };

  init();
});
