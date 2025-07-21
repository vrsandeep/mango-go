// --- Interface for library_grid.js ---
let currentFolderId;

// --- Folder Settings Management ---
let folderSettingsLoaded = false;

// Fetch folder settings and update state
const loadFolderSettings = async () => {
    if (!currentFolderId || folderSettingsLoaded) return;

    try {
        const response = await fetch(`/api/folders/${currentFolderId}/settings`);
        if (response.ok) {
            const settings = await response.json();
            state.sortBy = settings.sort_by || 'auto';
            state.sortDir = settings.sort_dir || 'asc';

            // Update UI elements to reflect the loaded settings
            const sortBySelect = document.getElementById('sort-by');
            const sortDirBtn = document.getElementById('sort-dir-btn');
            if (sortBySelect) sortBySelect.value = state.sortBy;
            if (sortDirBtn) sortDirBtn.textContent = state.sortDir === 'asc' ? '▲' : '▼';

            folderSettingsLoaded = true;
        }
    } catch (error) {
        console.error('Failed to load folder settings:', error);
    }
};

// Update current folder ID and reload settings
const updateCurrentFolder = (newFolderId) => {
    if (currentFolderId !== newFolderId) {
        currentFolderId = newFolderId;
        folderSettingsLoaded = false; // Reset flag to allow loading new folder settings
        loadFolderSettings(); // Load settings for the new folder
    }
};

// This function tells the generic grid how to get its data.
const getCardsLoadingUrl = () => {
    const params = new URLSearchParams({
        page: state.currentPage,
        per_page: state.perPage,
        search: state.search,
        sort_by: state.sortBy,
        sort_dir: state.sortDir
    });
    if (currentFolderId) {
        params.set('folderId', currentFolderId);
    }
    return `/api/browse?${params.toString()}`;
};

// This function runs after data is fetched, to update page-specific elements.
const postCardsFetchAction = (data) => {
    const pageTitleEl = document.getElementById('page-title');
    const editFolderBtn = document.getElementById('edit-folder-btn');

    pageTitleEl.textContent = data.current_folder ? data.current_folder.name : 'Library';
    document.title = `${pageTitleEl.textContent} - Mango`;

    // Show the "Edit" button only when viewing a specific folder.
    editFolderBtn.style.display = data.current_folder ? 'block' : 'none';

    renderBreadcrumb(currentFolderId);
    renderTags(data.current_folder ? data.current_folder.tags : []);
};

// This function tells the generic grid how to render its cards.
const renderCards = (data, cardsGrid) => {
    if ((!data.subfolders || data.subfolders.length === 0) && (!data.chapters || data.chapters.length === 0)) {
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
};

const resetState = (cardsGrid) => {
    // Clear page-specific elements before a new load.
    document.getElementById('breadcrumb-container').innerHTML = '';
    document.getElementById('page-title').textContent = 'Library';
    document.getElementById('tags-container').innerHTML = '';
    document.getElementById('edit-folder-modal').style.display = 'none';

    // Reset folder settings flag to allow reloading when navigating to different folders
    folderSettingsLoaded = false;
};


// --- Page-Specific Helper Functions ---
const createFolderCard = (folder) => {
    const card = document.createElement('a');
    card.href = `/library/folder/${folder.id}`;
    card.className = 'item-card folder';
    card.innerHTML = `
        <div class="thumbnail-container">
            <img class="thumbnail" src="${folder.thumbnail || '/static/images/logo.svg'}" loading="lazy" alt="Cover for ${folder.name}">
        </div>
        <div class="item-title" title="${folder.name}">${folder.name}</div>
        <div class="progress-bar-container">
            <div class="progress-bar" style="width: ${folder.read_chapters / folder.total_chapters * 100 || 0}%;"></div>
        </div>
    `;
    return card;
};

const createChapterCard = (chapter) => {
    const card = document.createElement('a');
    const progressPercent = chapter.progress_percent || 0;
    card.href = `/reader/series/${chapter.folder_id}/chapters/${chapter.id}`;
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

const renderBreadcrumb = async (folderId) => {
    const breadcrumbEl = document.getElementById('breadcrumb-container');
    if (!folderId) {
        breadcrumbEl.innerHTML = '';
        return;
    }
    const response = await fetch(`/api/browse/breadcrumb?folderId=${folderId}`);
    const path = await response.json();

    let html = '<a href="/library">Library</a>';
    path.forEach(folder => {
        html += ` / <a href="/library/folder/${folder.id}">${folder.name}</a>`;
    });
    breadcrumbEl.innerHTML = html;
};

// --- Modal and Tagging Logic ---
let allTags = [];
let currentFolderTags = [];

const loadAllTags = async () => {
    try {
        const response = await fetch('/api/tags');
        allTags = await response.json() || [];
    } catch (e) {
        console.error("Failed to load all tags:", e);
    }
};
const renderTags = (tags) => {
    const tagsContainer = document.getElementById('tags-container');
    currentFolderTags = tags || [];
    tagsContainer.innerHTML = '';
    currentFolderTags.forEach(tag => {
        const tagEl = document.createElement('div');
        tagEl.className = 'tag';
        tagEl.innerHTML = `<span>${tag.name}</span><span class="tag-remove-btn" data-tag-id="${tag.id}">&times;</span>`;
        tagsContainer.appendChild(tagEl);
    });
};

const addTag = async (tagName) => {
    const normalizedTagName = tagName.trim().toLowerCase();
    if (normalizedTagName === '' || !currentFolderId || currentFolderTags.some(t => t.name === normalizedTagName)) {
        document.getElementById('tag-input').value = '';
        return;
    }
    const response = await fetch(`/api/folders/${currentFolderId}/tags`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: normalizedTagName })
    });
    if (response.ok) {
        document.getElementById('tag-input').value = '';
        document.getElementById('autocomplete-suggestions').style.display = 'none';
        const newTag = await response.json();
        currentFolderTags.push(newTag);
        renderTags(currentFolderTags);
    }
};

const removeTag = async (tagId) => {
    await fetch(`/api/folders/${currentFolderId}/tags/${tagId}`, { method: 'DELETE' });
    currentFolderTags = currentFolderTags.filter(t => t.id != tagId);
    renderTags(currentFolderTags);
};


// --- DOMContentLoaded Initialization ---
document.addEventListener('DOMContentLoaded', async () => {
    const pathParts = window.location.pathname.split('/folder/');
    currentFolderId = pathParts.length > 1 ? pathParts[1] : null;

    // Load folder settings before initial card load
    await loadFolderSettings();

    const editFolderBtn = document.getElementById('edit-folder-btn');
    const editFolderModal = document.getElementById('edit-folder-modal');
    const modalCancelBtn = document.getElementById('modal-cancel-btn');
    const modalSaveBtn = document.getElementById('modal-save-btn');
    const coverFileInput = document.getElementById('cover-file-input');
    const markAllReadBtn = document.getElementById('mark-all-read-btn');
    const markAllUnreadBtn = document.getElementById('mark-all-unread-btn');
    const tagsContainer = document.getElementById('tags-container');
    const tagInput = document.getElementById('tag-input');
    const autocompleteSuggestions = document.getElementById('autocomplete-suggestions');

    // --- Modal Event Listeners ---
    editFolderBtn.addEventListener('click', () => {
        if (currentFolderId) {
            editFolderModal.style.display = 'flex';
            coverFileInput.value = ''; // Clear previous selection
        }
    });
    modalCancelBtn.addEventListener('click', () => editFolderModal.style.display = 'none');
    editFolderModal.addEventListener('click', (e) => { if (e.target === editFolderModal) editFolderModal.style.display = 'none'; });

    modalSaveBtn.addEventListener('click', async () => {
        const file = coverFileInput.files[0];
        if (file) {
            const formData = new FormData();
            formData.append('cover_file', file);
            await fetch(`/api/folders/${currentFolderId}/cover`, { method: 'POST', body: formData });
        }
        editFolderModal.style.display = 'none';
        loadCards(true);
    });

    markAllReadBtn.addEventListener('click', async () => {
        await fetch(`/api/folders/${currentFolderId}/mark-all-as`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ read: true })
        });
        editFolderModal.style.display = 'none';
        loadCards(true);
    });

    markAllUnreadBtn.addEventListener('click', async () => {
        await fetch(`/api/folders/${currentFolderId}/mark-all-as`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ read: false })
        });
        editFolderModal.style.display = 'none';
        loadCards(true);
    });

    // --- Tagging Event Listeners ---
    tagsContainer.addEventListener('click', (e) => {
        if (e.target.classList.contains('tag-remove-btn')) {
            removeTag(e.target.dataset.tagId);
        }
    });

    tagInput.addEventListener('input', () => {
        const query = tagInput.value.toLowerCase();
        if (query.length === 0) {
            autocompleteSuggestions.style.display = 'none';
            return;
        }

        const suggestions = allTags.filter(tag => tag.name.toLowerCase().includes(query) && !currentFolderTags.some(st => st.name === tag.name));

        autocompleteSuggestions.innerHTML = '';
        if (suggestions.length > 0) {
            suggestions.slice(0, 5).forEach(suggestion => { // Limit to 5 suggestions
                const item = document.createElement('div');
                item.className = 'suggestion-item';
                item.textContent = suggestion.name;
                item.addEventListener('click', () => {
                    addTag(suggestion.name);
                });
                autocompleteSuggestions.appendChild(item);
            });
            autocompleteSuggestions.style.display = 'block';
        } else {
            autocompleteSuggestions.style.display = 'none';
        }
    });
    tagInput.addEventListener('keydown', (e) => { if (e.key === 'Enter') { e.preventDefault(); addTag(tagInput.value); } });

    // Hide suggestions when clicking outside
    document.addEventListener('click', (e) => {
        if (!e.target.closest('.tag-input-container')) {
            autocompleteSuggestions.style.display = 'none';
        }
    });

    loadAllTags();
});

