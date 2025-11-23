document.addEventListener('DOMContentLoaded', () => {
  const searchBtn = document.getElementById('search-btn');
  const searchModal = document.getElementById('search-modal');
  const searchInput = document.getElementById('search-input-modal');
  const searchCloseBtn = document.getElementById('search-close-btn');
  const searchResults = document.getElementById('search-results');

  let searchTimeout = null;

  // Open search modal
  searchBtn?.addEventListener('click', () => {
    searchModal.style.display = 'flex';
    searchInput?.focus();
  });

  // Close search modal
  const closeSearch = () => {
    searchModal.style.display = 'none';
    searchInput.value = '';
    searchResults.innerHTML = '';
  };

  searchCloseBtn?.addEventListener('click', closeSearch);

  // Close on overlay click
  searchModal?.addEventListener('click', (e) => {
    if (e.target === searchModal) {
      closeSearch();
    }
  });

  // Close on Escape key
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && searchModal.style.display === 'flex') {
      closeSearch();
    }
  });

  // Perform search with debounce
  searchInput?.addEventListener('input', (e) => {
    const query = e.target.value.trim();

    // Clear previous timeout
    if (searchTimeout) {
      clearTimeout(searchTimeout);
    }

    // If query is empty, clear results
    if (query.length === 0) {
      searchResults.innerHTML = '';
      return;
    }

    // Wait 300ms before searching
    searchTimeout = setTimeout(async () => {
      try {
        const response = await fetch(`/api/folders/search?q=${encodeURIComponent(query)}`);
        if (!response.ok) {
          throw new Error('Search failed');
        }
        const folders = await response.json();
        renderSearchResults(folders);
      } catch (error) {
        console.error('Search error:', error);
        searchResults.innerHTML = '<div class="search-error">Error performing search. Please try again.</div>';
      }
    }, 300);
  });

  // Render search results
  function renderSearchResults(folders) {
    if (folders.length === 0) {
      searchResults.innerHTML = '<div class="search-empty">No folders found matching your search.</div>';
      return;
    }

    const resultsHTML = folders.map(folder => `
      <div class="search-result-item" data-folder-id="${folder.id}">
        <div class="search-result-name">${escapeHtml(folder.name)}</div>
        <div class="search-result-path">${escapeHtml(folder.path || '')}</div>
      </div>
    `).join('');

    searchResults.innerHTML = resultsHTML;

    // Add click handlers to results
    searchResults.querySelectorAll('.search-result-item').forEach(item => {
      item.addEventListener('click', () => {
        const folderId = item.dataset.folderId;
        window.location.href = `/library/folder/${folderId}`;
      });
    });
  }

  // Escape HTML to prevent XSS
  function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }
});

