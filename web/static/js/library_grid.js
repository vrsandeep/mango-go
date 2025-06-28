let state = {
  currentPage: 1,
  search: '',
  sortBy: '',
  sortDir: '',
  isLoading: false,
  totalItems: 0,
  perPage: 100
};
let loadCards;
let sortBySelect;
let sortDirBtn;
document.addEventListener('DOMContentLoaded', async () => {
  const currentUser = await checkAuth();
  if (!currentUser) return;

  const cardsGrid = document.getElementById('cards-grid');
  const searchInput = document.getElementById('search-input');
  sortBySelect = document.getElementById('sort-by');
  sortDirBtn = document.getElementById('sort-dir-btn');
  const totalCountEl = document.getElementById('total-count');
  const paginationContainer = document.getElementById('pagination-container');

  // --- Data Loading ---
  loadCards = async (reset = false) => {
    if (state.isLoading || !reset) return;
    state.isLoading = true;
    cardsGrid.innerHTML = '<p>Loading...</p>';
    if (reset) {
      resetState(cardsGrid);
    }

    const url = getCardsLoadingUrl();
    const response = await fetch(url);
    const cardsList = await response.json();

    const total = response.headers.get('X-Total-Count');
    state.totalItems = parseInt(total || '0', 10);
    totalCountEl.textContent = `${total}`;
    cardsGrid.innerHTML = '';
    postCardsFetchAction(cardsList);
    renderPagination();
    if (!cardsList || cardsList.length === 0) {
      cardsGrid.innerHTML = '<p>No items found.</p>';
    }
    renderCards(cardsList, cardsGrid)
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

  // --- Event Listeners ---
  // loadMoreBtn.addEventListener('click', () => {
  //   if (!state.isLoading) {
  //     state.currentPage++;
  //     loadCards(false);
  //   }
  // });

  let searchTimeout;
  searchInput.addEventListener('input', () => {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
      state.search = searchInput.value;
      state.currentPage = 1; // Reset to first page on new search
      loadCards(true);
    }, 300); // Debounce
  });

  sortBySelect.addEventListener('change', () => {
    state.sortBy = sortBySelect.value;
    loadCards(true);
  });

  sortDirBtn.addEventListener('click', () => {
    state.sortDir = state.sortDir === 'asc' ? 'desc' : 'asc';
    sortDirBtn.textContent = state.sortDir === 'asc' ? '▲' : '▼';
    loadCards(true);
  });

  // Initial load
  loadCards(true); // Initial load
});
