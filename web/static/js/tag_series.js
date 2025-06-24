const GET_TAG_SERIES_URL = `/api/tags/TAG_ID/series?page=STATE_CURRENT_PAGE&per_page=100&search=STATE_SEARCH&sort_by=STATE_SORT_BY&sort_dir=STATE_SORT_DIR`;


const getCardsLoadingUrl = () => {
  const tagId = window.location.pathname.split('/')[2];
  return GET_TAG_SERIES_URL
    .replace("TAG_ID", tagId)
    .replace("STATE_CURRENT_PAGE", state.currentPage)
    .replace("STATE_SEARCH", state.search)
    .replace("STATE_SORT_BY", state.sortBy)
    .replace("STATE_SORT_DIR", state.sortDir);
};

const areMoreCardsAvailable = (cardsList) => {
  return !cardsList || cardsList.length < 100;
}

const postCardsFetchAction = async (cardsList) => { }
const resetState = (cardsGrid) => {
  state.currentPage = 1;
  state.hasMore = true;
  cardsGrid.innerHTML = '';
}

// const updateSettings = async () => {
//   // do nothing for now
// }
const renderCards = (cardsList, cardsGrid) => {
  if (!cardsList) {
    return;
  }
  cardsList.forEach(cards => {
    const card = document.createElement('a');
    card.href = `/series/${cards.id}`;
    card.className = 'item-card';
    // Prioritize custom cover, fall back to generated thumbnail
    const coverSrc = cards.custom_cover_url || cards.thumbnail || '';
    const progressPercent = cards.total_chapters > 0 ? (cards.read_chapters / cards.total_chapters) * 100 : 0;

    card.innerHTML = `
          <div class="thumbnail-container">
            <img class="thumbnail" src="${coverSrc}" alt="Cover for ${cards.title}" loading="lazy">
          </div>
          <div class="item-title" title="${cards.title}">${cards.title}</div>
          <div class="progress-bar-container">
            <div class="progress-bar" style="width: ${progressPercent}%;"></div>
          </div>
        `;
    cardsGrid.appendChild(card);
  });
}

document.addEventListener('DOMContentLoaded', () => {
  const tagId = window.location.pathname.split('/')[2];
  const pageTitleEl = document.getElementById('page-title');
  const loadTagTitle = async () => {
    const response = await fetch(`/api/tags/${tagId}`);
    const tag = await response.json();
    const title = `Tag: ${tag.name}`;
    document.title = title;
    pageTitleEl.textContent = title;
    // pageTitleEl.prepend(document.createTextNode(title));
  };

  loadTagTitle();
});