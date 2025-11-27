import { checkAuth } from './auth.js';

document.addEventListener('DOMContentLoaded', async () => {
  const user = await checkAuth();
  if (!user) return;

  const homeContainer = document.getElementById('home-container');

  const createCard = item => {
    const card = document.createElement('a');
    card.className = 'item-card';

    const isChapter = item.chapter_id && item.chapter_id > 0;

    // Link to the chapter reader if it's a chapter, otherwise to the series page.
    card.href = isChapter
      ? `/reader/series/${item.series_id}/chapters/${item.chapter_id}`
      : `/library/folder/${item.series_id}`;

    const coverSrc = item.cover_art || '';
    const title = isChapter ? item.chapter_title.split(/[\\/]/).pop() : item.series_title;
    const subTitle = isChapter ? item.series_title : '';

    let progressBarHTML = '';
    // Only show progress for chapter cards, not series cards
    // Always show progress bar container for chapter cards (like library), even if progress is 0
    if (isChapter) {
      const progressPercent = item.progress_percent || 0;
      progressBarHTML = `
        <div class="progress-bar-container">
          <div class="progress-bar" style="width: ${progressPercent}%;"></div>
        </div>
      `;
    }

    let badgeHTML = '';
    if (item.new_chapter_count > 1) {
      badgeHTML = `<div class="badge">+${item.new_chapter_count}</div>`;
    }

    card.innerHTML = `
        <div class="thumbnail-container">
          <img class="thumbnail" src="${coverSrc}" alt="Cover for ${title}" loading="lazy">
          ${badgeHTML}
        </div>
        <div class="item-title" title="${title}">${title}</div>
        ${subTitle ? `<div class="item-subtitle">${subTitle}</div>` : ''}
        ${progressBarHTML}
        `;
    return card;
  };

  const createSection = (title, items) => {
    if (!items || items.length === 0) {
      return null; // Don't create the section if there are no items
    }

    const section = document.createElement('div');
    section.className = 'home-section';

    const sectionTitle = document.createElement('h2');
    sectionTitle.textContent = title;
    section.appendChild(sectionTitle);

    const scrollContainer = document.createElement('div');
    scrollContainer.className = 'horizontal-scroll-container';
    items.forEach(item => {
      scrollContainer.appendChild(createCard(item));
    });
    section.appendChild(scrollContainer);

    return section;
  };

  const loadHomePageData = async () => {
    try {
      const response = await fetch('/api/home');
      if (!response.ok) throw new Error('Failed to fetch home page data');

      const data = await response.json();
      homeContainer.innerHTML = ''; // Clear loading message

      const sections = [
        createSection('Continue Reading', data.continue_reading),
        createSection('Next Up', data.next_up),
        createSection('Recently Added', data.recently_added),
        createSection('Start Reading', data.start_reading),
      ].filter(Boolean); // Filter out null sections

      if (sections.length > 0) {
        sections.forEach(section => homeContainer.appendChild(section));
      } else {
        homeContainer.innerHTML = '<p>Your library is empty. Add some series to get started!</p>';
      }
    } catch (error) {
      console.error('Error loading home page:', error);
      homeContainer.innerHTML = '<p>Could not load your home page. Please try again later.</p>';
    }
  };

  loadHomePageData();
});
