let state = {
  currentPage: 1,
  search: '',
  sortBy: 'title',
  sortDir: 'asc',
  isLoading: false,
  hasMore: true
};
let loadCards;
let sortBySelect;
let sortDirBtn;
document.addEventListener('DOMContentLoaded', () => {
  const cardsGrid = document.getElementById('cards-grid');
  const loadMoreBtn = document.getElementById('load-more-btn');
  const themeToggleBtn = document.getElementById('theme-toggle-btn');
  const menuToggleBtn = document.getElementById('menu-toggle-btn');
  const navLinks = document.getElementById('nav-links');
  const searchInput = document.getElementById('search-input');
  sortBySelect = document.getElementById('sort-by');
  sortDirBtn = document.getElementById('sort-dir-btn');
  const totalCountEl = document.getElementById('total-count');

  // --- Theme Logic ---
  const applyTheme = (theme) => {
    document.body.classList.toggle('light-theme', theme === 'light');
  };
  themeToggleBtn.addEventListener('click', () => {
    const newTheme = document.body.classList.contains('light-theme') ? 'dark' : 'light';
    localStorage.setItem('theme', newTheme);
    applyTheme(newTheme);
  });

  // --- Mobile Menu Logic ---
  menuToggleBtn.addEventListener('click', () => navLinks.classList.toggle('active'));

  // --- Data Loading ---
  loadCards = async (reset = false) => {
    if (state.isLoading || !state.hasMore && !reset) return;
    state.isLoading = true;
    if (reset) {
      resetState(cardsGrid);
    }

    const url = getCardsLoadingUrl();
    const response = await fetch(url);
    const cardsList = await response.json();

    if (reset) {
      const total = response.headers.get('X-Total-Count');
      totalCountEl.textContent = `${total}`;
    }
    postCardsFetchAction(cardsList);
    if (areMoreCardsAvailable(cardsList)) {
      state.hasMore = false;
      loadMoreBtn.style.display = 'none';
    } else {
      loadMoreBtn.style.display = 'block';
    }
    renderCards(cardsList, cardsGrid)
    state.isLoading = false;
  };

  // --- Event Listeners ---
  loadMoreBtn.addEventListener('click', () => {
    if (!state.isLoading) {
      state.currentPage++;
      loadCards(false);
    }
  });

  let searchTimeout;
  searchInput.addEventListener('input', () => {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
      state.search = searchInput.value;
      loadCards(true);
    }, 300); // Debounce
  });

  sortBySelect.addEventListener('change', () => {
    state.sortBy = sortBySelect.value;
    loadCards(true);
    updateSettings();
  });

  sortDirBtn.addEventListener('click', () => {
    state.sortDir = state.sortDir === 'asc' ? 'desc' : 'asc';
    sortDirBtn.textContent = state.sortDir === 'asc' ? '▲' : '▼';
    loadCards(true);
    updateSettings();
  });

  const loadVersion = async () => {
    const response = await fetch('/api/version');
    const data = await response.json();
    document.getElementById('version-footer').textContent = `Version: ${data.version}`;
  };

  // Initial load
  applyTheme(localStorage.getItem('theme'));
  loadCards(true); // Initial load
  loadVersion();
});
