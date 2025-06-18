const GET_SERIES_URL = `/api/series?page=STATE_CURRENT_PAGE&per_page=100&search=STATE_SEARCH&sort_by=STATE_SORT_BY&sort_dir=STATE_SORT_DIR`;

const getCardsLoadingUrl = () => {
  return GET_SERIES_URL
    .replace("STATE_CURRENT_PAGE", state.currentPage)
    .replace("STATE_SEARCH", state.search)
    .replace("STATE_SORT_BY", state.sortBy)
    .replace("STATE_SORT_DIR", state.sortDir);
};

const areMoreCardsAvailable = (cardsList) => {
  return !cardsList || cardsList.length < 100;
}

const postCardsFetchAction = async (cardsList) => {}

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
