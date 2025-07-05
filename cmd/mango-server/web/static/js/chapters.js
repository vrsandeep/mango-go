const GET_CHAPTERS_URL = `/api/series/SERIES_ID?page=STATE_CURRENT_PAGE&per_page=PER_PAGE&search=STATE_SEARCH&sort_by=STATE_SORT_BY&sort_dir=STATE_SORT_DIR`;

let seriesId;
let seriesTitleEl;
let breadCrumbSeriesTitleEl;
let seriesThumbEl;
let renderTags;
let allTags = []; // To store the master list of all tags
let currentSeriesTags = [];
const getCardsLoadingUrl = () => {
  return GET_CHAPTERS_URL
    .replace("STATE_CURRENT_PAGE", state.currentPage)
    .replace("PER_PAGE", state.perPage)
    .replace("STATE_SEARCH", state.search)
    .replace("STATE_SORT_BY", state.sortBy)
    .replace("STATE_SORT_DIR", state.sortDir)
    .replace("SERIES_ID", seriesId);
};

const postCardsFetchAction = async (seriesData) => {
  document.title = seriesData.title;
  seriesTitleEl.textContent = seriesData.title;
  breadCrumbSeriesTitleEl.textContent = seriesData.title;
  seriesThumbEl.src = seriesData.custom_cover_url || seriesData.thumbnail || '';
  if (seriesData.settings) {
    state.sortBy = seriesData.settings.sort_by;
    state.sortDir = seriesData.settings.sort_dir;
    sortBySelect.value = state.sortBy;
    sortDirBtn.textContent = state.sortDir === 'asc' ? '▲' : '▼';
  }
}

const resetState = (cardsGrid) => {
  renderTags([]);
}

const renderCards = (seriesData, cardsGrid) => {
  if (!seriesData.chapters) {
    return;
  }
  seriesData.chapters.forEach(chapter => {
    const card = document.createElement('a');
    const progressPercent = chapter.progress_percent;
    card.href = `/reader/series/${seriesId}/chapters/${chapter.id}`;
    card.classList.add('item-card');
    card.innerHTML = `
        <div class="thumbnail-container">
            <img class="thumbnail" src="${chapter.thumbnail || ''}" alt="Cover for Chapter ${chapter.id}">
        </div>
        <div class="item-title">${chapter.path.split('/').pop()}</div>
        <div class="progress-bar-container">
            <div class="progress-bar" style="width: ${progressPercent}%;"></div>
        </div>
      `;
    cardsGrid.appendChild(card);
  });

  // Load tags
  if (seriesData.tags) {
    renderTags(seriesData.tags);
  }
}
// const updateSettings = async () => {
//   await fetch(`/api/series/${seriesId}/settings`, {
//     method: 'POST',
//     headers: {'Content-Type': 'application/json'},
//     body: JSON.stringify({sort_by: state.sortBy, sort_dir: state.sortDir})
//   });
// };

document.addEventListener('DOMContentLoaded', () => {
  seriesId = window.location.pathname.split('/')[2];
  seriesTitleEl = document.getElementById('series-title');
  seriesThumbEl = document.getElementById('series-thumb');
  breadCrumbSeriesTitleEl = document.getElementById('breadcrumb-series-title');

  const tagsContainer = document.getElementById('tags-container');
  const tagInput = document.getElementById('tag-input');
  const autocompleteSuggestions = document.getElementById('autocomplete-suggestions');

  const editSeriesBtn = document.getElementById('edit-series-btn');
  const editModal = document.getElementById('edit-modal');
  const modalCancelBtn = document.getElementById('modal-cancel-btn');
  const modalSaveBtn = document.getElementById('modal-save-btn');
  const coverUrlInput = document.getElementById('cover-url-input');
  const markAllReadBtn = document.getElementById('mark-all-read-btn');
  const markAllUnreadBtn = document.getElementById('mark-all-unread-btn');

  const updateCover = async (url) => {
    await fetch(`/api/series/${seriesId}/cover`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ url: url })
    });
    seriesThumbEl.src = url; // Update image on the page immediately
  };

  // seriesThumbEl.addEventListener('click', () => {
  //   const newUrl = prompt("Enter new cover image URL:", seriesThumbEl.src);
  //   if (newUrl && newUrl.trim() !== '') {
  //     updateCover(newUrl.trim());
  //   }
  // });

  // --- Event Listeners ---

  // --- Modal Logic ---
  editSeriesBtn.addEventListener('click', () => {
    fetch(`/api/series/${seriesId}?page=1&per_page=1`)
      .then(res => res.json())
      .then(seriesData => {
        coverUrlInput.value = seriesData.custom_cover_url || '';
        editModal.style.display = 'flex';
      });
  });
  modalCancelBtn.addEventListener('click', () => editModal.style.display = 'none');
  editModal.addEventListener('click', (e) => {
    if (e.target === editModal) editModal.style.display = 'none';
  });

  modalSaveBtn.addEventListener('click', async () => {
    await fetch(`/api/series/${seriesId}/cover`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ url: coverUrlInput.value })
    });
    editModal.style.display = 'none';
    // Reload the page to reflect changes
    window.location.reload();
  });

  markAllReadBtn.addEventListener('click', async () => {
    await fetch(`/api/series/${seriesId}/mark-all-as`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ read: true })
    });
    showToast('All chapters marked as read.');
    setTimeout(() => {
      window.location.reload();
    }, 1000);
  });

  markAllUnreadBtn.addEventListener('click', async () => {
    await fetch(`/api/series/${seriesId}/mark-all-as`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ read: false })
    });
    showToast('All chapters marked as unread.');
    setTimeout(() => {
      window.location.reload();
    }, 1000);
  });
  const showToast = (message) => {
    const toast = document.createElement('div');
    toast.textContent = message;
    toast.style.position = 'fixed';
    toast.style.bottom = '20px';
    toast.style.left = '50%';
    toast.style.transform = 'translateX(-50%)';
    toast.style.backgroundColor = '#333';
    toast.style.color = '#fff';
    toast.style.padding = '10px 20px';
    toast.style.borderRadius = '5px';
    toast.style.zIndex = '1000';
    document.body.appendChild(toast);
    // setTimeout(() => { document.body.removeChild(toast); }, 3000);
  };

  // --- Tag Logic ---
  const loadAllTags = async () => {
    try {
      const response = await fetch('/api/tags');
      allTags = await response.json() || [];
    } catch (e) {
      console.error("Failed to load all tags:", e);
    }
  };
  renderTags = (tags) => {
    currentSeriesTags = tags;
    tagsContainer.innerHTML = '';
    tags.forEach(tag => {
      const tagEl = document.createElement('div');
      tagEl.className = 'tag';
      tagEl.innerHTML = `<span>${tag.name}</span><span class="tag-remove-btn" data-tag-id="${tag.id}">&times;</span>`;
      tagsContainer.appendChild(tagEl);
    });
  };

  const addTag = async (tagName) => {
    const normalizedTagName = tagName.trim().toLowerCase();
    if (normalizedTagName === '' || currentSeriesTags.some(t => t.name === normalizedTagName)) {
      tagInput.value = '';
      return;
    }
    try {
      const response = await fetch(`/api/series/${seriesId}/tags`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: normalizedTagName })
      });
      if (response.ok) {
        tagInput.value = '';
        autocompleteSuggestions.style.display = 'none';
        const newTag = await response.json();
        currentSeriesTags.push(newTag);
        renderTags(currentSeriesTags);
        // loadCards(true); // Reload everything to get updated tags
      }
    } catch (e) {
      console.error("Failed to add tag:", e);
    }
  };

  const removeTag = async (tagId) => {
    await fetch(`/api/series/${seriesId}/tags/${tagId}`, { method: 'DELETE' });
    loadCards(true);
  };

  // tagInput.addEventListener('keyup', (e) => {
  //   if (e.key === 'Enter' && tagInput.value.trim() !== '') {
  //     addTag(tagInput.value.trim());
  //   }
  // });
  tagInput.addEventListener('input', () => {
    const query = tagInput.value.toLowerCase();
    if (query.length === 0) {
      autocompleteSuggestions.style.display = 'none';
      return;
    }

    const suggestions = allTags.filter(tag => tag.name.toLowerCase().includes(query) && !currentSeriesTags.some(st => st.name === tag.name));

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
  tagInput.addEventListener('keydown', (e) => {
    const suggestions = autocompleteSuggestions.querySelectorAll('.suggestion-item');
    if (e.key === 'Enter') {
      e.preventDefault();
      const activeSuggestion = autocompleteSuggestions.querySelector('.active');
      if (activeSuggestion) {
        addTag(activeSuggestion.textContent);
      } else {
        addTag(tagInput.value);
      }
    } else if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      // Keyboard navigation logic for suggestions
    }
  });
  // Hide suggestions when clicking outside
  document.addEventListener('click', (e) => {
    if (!e.target.closest('.tag-input-container')) {
      autocompleteSuggestions.style.display = 'none';
    }
  });


  tagsContainer.addEventListener('click', (e) => {
    if (e.target.classList.contains('tag-remove-btn')) {
      removeTag(e.target.dataset.tagId);
    }
  });

  loadAllTags();
});
