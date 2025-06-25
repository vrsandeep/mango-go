function initializeLibraryPage(getApiUrl, renderItemCard, loadExtraData = null) {
    const grid = document.getElementById('series-grid') || document.getElementById('chapters-grid');
    const searchInput = document.getElementById('search-input');
    const sortBySelect = document.getElementById('sort-by');
    const sortDirBtn = document.getElementById('sort-dir-btn');
    const totalCountEl = document.getElementById('total-count');
    const paginationContainer = document.getElementById('pagination-container');
    const perPage = 100;

    let state = {
        currentPage: 1,
        search: '',
        sortBy: sortBySelect ? sortBySelect.value : 'title',
        sortDir: 'asc',
        isLoading: false,
        totalItems: 0
    };

    const fetchData = async () => {
        if (state.isLoading) return;
        state.isLoading = true;
        grid.innerHTML = '<p>Loading...</p>';

        try {
            const apiUrl = getApiUrl(state);
            const response = await fetch(apiUrl);
            const items = await response.json();

            state.totalItems = parseInt(response.headers.get('X-Total-Count') || '0', 10);

            grid.innerHTML = '';
            if (items && items.length > 0) {
                items.forEach(item => {
                    const card = renderItemCard(item);
                    grid.appendChild(card);
                });
            } else {
                grid.innerHTML = '<p>No items found.</p>';
            }

            if (totalCountEl) {
                totalCountEl.textContent = `(${state.totalItems} found)`;
            }

            renderPagination();

        } catch (error) {
            console.error("Failed to fetch data:", error);
            grid.innerHTML = '<p>Error loading data. Please try again.</p>';
        } finally {
            state.isLoading = false;
        }
    };

    const renderPagination = () => {
        paginationContainer.innerHTML = '';
        const totalPages = Math.ceil(state.totalItems / perPage);
        if (totalPages <= 1) return;

        const createButton = (text, page, isDisabled = false, isActive = false) => {
            const btn = document.createElement('button');
            btn.className = 'pagination-btn';
            btn.innerHTML = text;
            if (isDisabled) btn.classList.add('disabled');
            if (isActive) btn.classList.add('active');
            if (!isDisabled && !isActive) {
                btn.addEventListener('click', () => {
                    state.currentPage = page;
                    fetchData();
                });
            }
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
        if(totalPages > 1) pageNumbers.push(totalPages);

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
    let searchTimeout;
    if (searchInput) {
        searchInput.addEventListener('input', () => {
            clearTimeout(searchTimeout);
            searchTimeout = setTimeout(() => {
                state.search = searchInput.value;
                state.currentPage = 1; // Reset to first page on new search
                fetchData();
            }, 300);
        });
    }

    if (sortBySelect) {
        sortBySelect.addEventListener('change', () => {
            state.sortBy = sortBySelect.value;
            fetchData();
        });
    }

    if (sortDirBtn) {
        sortDirBtn.addEventListener('click', () => {
            state.sortDir = state.sortDir === 'asc' ? 'desc' : 'asc';
            sortDirBtn.textContent = state.sortDir === 'asc' ? '▲' : '▼';
            fetchData();
        });
    }

    // --- Initial Load ---
    if (loadExtraData) {
        loadExtraData();
    }
    fetchData();
}