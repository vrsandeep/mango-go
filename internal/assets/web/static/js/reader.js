// reader2.js — enhanced reader: RTL direction, double-page spread, zoom + drag-to-pan.
// To revert to the original reader, change reader.html to load reader.js instead.
import { checkAuth } from './auth.js';

document.addEventListener('DOMContentLoaded', async () => {
  const currentUser = await checkAuth();
  if (!currentUser) return;

  const savedTheme = localStorage.getItem('theme');
  if (savedTheme === 'light') document.body.classList.add('light-theme');
  else document.body.classList.remove('light-theme');

  const pathParts = window.location.pathname.split('/');
  const folderId = pathParts[3];
  const chapterId = pathParts[5];

  let state = {
    folderData: null,
    chapterData: null,
    allChapters: [],
    currentPage: 1,
    readingMode: localStorage.getItem('readingMode') || 'continuous',
    pageMargin: localStorage.getItem('pageMargin') || '10',
    fitMode: localStorage.getItem('fitMode') || 'fit-original',
    // 'ltr' = left-to-right (default); 'rtl' flips arrow key mapping and page order in spreads.
    readingDirection: localStorage.getItem('readingDirection') || 'ltr',
    zoomLevel: 1.0,
    panX: 0,
    panY: 0,
  };

  // Transient panning state — not persisted in localStorage.
  let isPanning = false;
  let panMoved = false;
  let panStartX = 0;
  let panStartY = 0;

  // Active spread index for double-page mode.
  let currentSpread = 0;

  let nextChapterId = null;
  let prevChapterId = null;

  // --- DOM References ---
  const imageContainer = document.getElementById('image-container');
  const progressBar = document.getElementById('progress-bar');
  const singlePageViewer = document.getElementById('single-page-viewer');
  const singlePrevBtn = document.getElementById('single-prev-btn');
  const singleNextBtn = document.getElementById('single-next-btn');
  const modal = document.getElementById('reader-modal');
  const modalTitle = document.getElementById('modal-title');
  const modalPath = document.getElementById('modal-path');
  const modalProgress = document.getElementById('modal-progress');
  const jumpToPageSelect = document.getElementById('jump-to-page');
  const modeSelect = document.getElementById('mode-select');
  const marginSlider = document.getElementById('margin-slider');
  const jumpToEntrySelect = document.getElementById('jump-to-entry');
  const modalPrevBtn = document.getElementById('modal-prev-btn');
  const modalNextBtn = document.getElementById('modal-next-btn');
  const modalExitBtn = document.getElementById('modal-exit-btn');
  const modalCloseBtn = document.getElementById('modal-close-btn');
  const fitModeSelect = document.getElementById('fit-mode-select');
  const directionSelect = document.getElementById('direction-select');
  const zoomIndicator = document.getElementById('zoom-indicator');
  const footerPrevBtn = document.getElementById('footer-prev-chapter-btn');
  const footerNextBtn = document.getElementById('footer-next-chapter-btn');
  const footerExitBtn = document.getElementById('footer-exit-chapter-btn');

  // --- Data Fetching ---
  const fetchInitialData = async () => {
    const [chapterRes, folderRes] = await Promise.all([
      fetch(`/api/chapters/${chapterId}`),
      fetch(`/api/browse?folderId=${folderId}&page=1&per_page=9999&sort_by=auto&sort_dir=asc`),
    ]);
    state.chapterData = await chapterRes.json();
    const folderContents = await folderRes.json();
    state.folderData = folderContents.current_folder;
    state.allChapters = folderContents.chapters;
  };

  const waitForImagesToLoad = () =>
    new Promise(resolve => {
      const images = document.querySelectorAll('.page-image:not([style*="display: none"])');
      if (images.length === 0) { resolve(); return; }
      let loaded = 0;
      const done = () => { if (++loaded === images.length) resolve(); };
      images.forEach(img => {
        if (img.complete) done();
        else { img.addEventListener('load', done); img.addEventListener('error', done); }
      });
    });

  // --- Progress ---
  const updateProgress = (progressPercent, isRead) => {
    fetch(`/api/chapters/${chapterId}/progress`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ progress_percent: progressPercent, read: isRead }),
    });
  };

  const updateProgressText = progressPercent => {
    let progress;
    if (state.readingMode === 'single_page') {
      progress = (state.currentPage / state.chapterData.page_count) * 100;
    } else if (state.readingMode === 'double_page') {
      const pages = getPagesForSpread(currentSpread);
      progress = (pages[pages.length - 1] / state.chapterData.page_count) * 100;
    } else {
      progress = progressPercent || state.chapterData.progress_percent;
    }
    const page = Math.ceil((progress / 100) * state.chapterData.page_count) || 1;
    modalProgress.textContent = `Progress: ${page}/${state.chapterData.page_count} (${progress.toFixed(1)}%)`;
  };

  const findNeighboringChapters = async () => {
    const response = await fetch(`/api/folders/${folderId}/chapters/${chapterId}/neighbors`);
    const neighbors = await response.json();
    if (neighbors.prev) {
      prevChapterId = neighbors.prev;
      footerPrevBtn.style.display = 'inline-block';
      modalPrevBtn.disabled = false;
    } else {
      modalPrevBtn.disabled = true;
    }
    if (neighbors.next) {
      nextChapterId = neighbors.next;
      footerNextBtn.style.display = 'inline-block';
      modalNextBtn.disabled = false;
    } else {
      footerExitBtn.style.display = 'inline-block';
      modalNextBtn.disabled = true;
    }
  };

  // --- Direction ---
  const isRTL = () => state.readingDirection === 'rtl';

  // --- Spread Helpers (double-page mode) ---
  // Spread layout: spread 0 is the cover (page 1 alone), then pairs: [2,3], [4,5], ...
  const getSpreadCount = () => {
    const n = state.chapterData.page_count;
    if (n <= 0) return 0;
    if (n === 1) return 1;
    return 1 + Math.ceil((n - 1) / 2);
  };

  const getPagesForSpread = spread => {
    if (spread === 0) return [1];
    const left = 2 + (spread - 1) * 2;
    const right = left + 1;
    return right <= state.chapterData.page_count ? [left, right] : [left];
  };

  // Returns the spread index that contains the given page number.
  const getSpreadForPage = page => {
    if (page <= 1) return 0;
    return 1 + Math.floor((page - 2) / 2);
  };

  // --- Rendering ---
  const renderPages = () => {
    imageContainer.innerHTML = '';
    currentSpread = 0;

    if (state.readingMode === 'double_page') {
      renderDoublePageMode();
      return;
    }

    for (let i = 1; i <= state.chapterData.page_count; i++) {
      const img = document.createElement('img');
      img.src = `/api/chapters/${chapterId}/pages/${i}`;
      img.classList.add('page-image');
      img.id = `page-${i}`;
      img.loading = 'lazy';
      imageContainer.appendChild(img);
    }
    applyReadingMode();
  };

  const renderDoublePageMode = () => {
    const total = getSpreadCount();
    for (let s = 0; s < total; s++) {
      const pages = getPagesForSpread(s);
      const spreadEl = document.createElement('div');
      // page-spread--single marks a spread with only one image (cover or last odd page)
      spreadEl.className = 'page-spread' + (pages.length === 1 ? ' page-spread--single' : '');
      spreadEl.dataset.spread = s;

      // In RTL, swap the pair so the higher-numbered page is on the left (manga convention)
      const displayPages = isRTL() && pages.length > 1 ? [pages[1], pages[0]] : pages;
      displayPages.forEach(pageNum => {
        const img = document.createElement('img');
        img.src = `/api/chapters/${chapterId}/pages/${pageNum}`;
        img.classList.add('page-image', 'spread-page');
        img.id = `page-${pageNum}`;
        img.loading = 'lazy';
        spreadEl.appendChild(img);
      });
      imageContainer.appendChild(spreadEl);
    }
    applyReadingMode();
  };

  const updateDoublePageView = () => {
    document.querySelectorAll('.page-spread').forEach((el, idx) => {
      el.style.display = idx === currentSpread ? 'flex' : 'none';
    });
    singlePrevBtn.disabled = currentSpread === 0;
    singleNextBtn.disabled = currentSpread === getSpreadCount() - 1;
    window.scrollTo(0, 0);
    updateProgressText();
    updateJumpToPageSelect();
  };

  const applyReadingMode = () => {
    localStorage.setItem('readingMode', state.readingMode);
    if (state.readingMode === 'single_page') {
      imageContainer.classList.remove('double-page');
      imageContainer.classList.add('single-page');
      singlePageViewer.style.display = 'flex';
      updateSinglePageView();
    } else if (state.readingMode === 'double_page') {
      imageContainer.classList.remove('single-page');
      imageContainer.classList.add('double-page');
      singlePageViewer.style.display = 'flex';
      updateDoublePageView();
    } else {
      // continuous
      imageContainer.classList.remove('single-page', 'double-page');
      singlePageViewer.style.display = 'none';
      document.querySelectorAll('.page-image').forEach(img => (img.style.display = 'block'));
    }
  };

  const updateSinglePageView = () => {
    document.querySelectorAll('.page-image').forEach((img, index) => {
      img.style.display = index + 1 === state.currentPage ? 'block' : 'none';
    });
    singlePrevBtn.disabled = state.currentPage === 1;
    singleNextBtn.disabled = state.currentPage === state.chapterData.page_count;
    window.scrollTo(0, 0);
    updateProgressText();
    updateJumpToPageSelect();
  };

  const updateJumpToPageSelect = progressPercent => {
    let progress;
    if (state.readingMode === 'single_page') {
      progress = (state.currentPage / state.chapterData.page_count) * 100;
    } else if (state.readingMode === 'double_page') {
      const pages = getPagesForSpread(currentSpread);
      progress = (pages[pages.length - 1] / state.chapterData.page_count) * 100;
    } else {
      progress = progressPercent || state.chapterData.progress_percent;
    }
    const page = Math.ceil((progress / 100) * state.chapterData.page_count) || 1;
    jumpToPageSelect.value = page;
  };

  const applyPageMargin = () => {
    localStorage.setItem('pageMargin', state.pageMargin);
    document.documentElement.style.setProperty('--page-margin', `${state.pageMargin}px`);
  };

  const applyFitMode = () => {
    localStorage.setItem('fitMode', state.fitMode);
    const images = document.querySelectorAll('.page-image');
    images.forEach(img => {
      img.style.width = '';
      img.style.height = '';
      img.style.maxWidth = '';
      img.style.maxHeight = '';
      img.style.objectFit = '';

      // Spread pages fill their flex cell; override normal fit modes.
      if (img.classList.contains('spread-page')) {
        img.style.width = '100%';
        img.style.height = 'auto';
        img.style.objectFit = 'contain';
        return;
      }

      switch (state.fitMode) {
        case 'fit-width':
          img.style.width = '100%';
          img.style.maxWidth = 'none';
          img.style.height = 'auto';
          break;
        case 'fit-height':
          img.style.width = 'auto';
          img.style.maxWidth = 'none';
          img.style.height = '100vh';
          img.style.maxHeight = '100vh';
          img.style.objectFit = 'contain';
          break;
        case 'fit-original':
          img.style.width = 'auto';
          img.style.height = 'auto';
          img.style.maxWidth = 'none';
          img.style.maxHeight = 'none';
          img.style.objectFit = 'contain';
          break;
        case 'fit-specific':
          img.style.width = '100%';
          img.style.height = 'auto';
          img.style.maxWidth = '850px';
          img.style.objectFit = 'contain';
          break;
      }
    });
  };

  // --- Zoom & Pan ---
  // Note: CSS transform does not expand layout, so in continuous mode the scrollable
  // height reflects unzoomed content. Zoom works best in single/double-page modes.
  const applyZoom = () => {
    const active = state.zoomLevel !== 1.0 || state.panX !== 0 || state.panY !== 0;
    imageContainer.style.transform = active
      ? `translate(${state.panX}px, ${state.panY}px) scale(${state.zoomLevel})`
      : '';
    imageContainer.style.cursor =
      state.zoomLevel > 1 ? (isPanning ? 'grabbing' : 'grab') : '';

    if (zoomIndicator) {
      if (state.zoomLevel !== 1.0) {
        zoomIndicator.textContent = `${Math.round(state.zoomLevel * 100)}%`;
        zoomIndicator.style.display = 'block';
      } else {
        zoomIndicator.style.display = 'none';
      }
    }
  };

  const zoomIn = () => {
    state.zoomLevel = Math.min(4.0, +(state.zoomLevel + 0.25).toFixed(2));
    applyZoom();
  };

  const zoomOut = () => {
    state.zoomLevel = Math.max(0.5, +(state.zoomLevel - 0.25).toFixed(2));
    if (state.zoomLevel === 1.0) { state.panX = 0; state.panY = 0; }
    applyZoom();
  };

  const zoomReset = () => {
    state.zoomLevel = 1.0;
    state.panX = 0;
    state.panY = 0;
    applyZoom();
  };

  // --- Modal ---
  const populateModal = () => {
    const chapter = state.chapterData;
    const folder = state.folderData;
    const lastPart = chapter.path.split(/[\\/]/).pop().replace(/\.[^/.]+$/, '');
    modalTitle.textContent = `${folder.name} - ${lastPart}`;
    modalPath.textContent = chapter.path;

    jumpToPageSelect.innerHTML = '';
    for (let i = 1; i <= chapter.page_count; i++) {
      const option = document.createElement('option');
      option.value = i;
      option.textContent = `Page ${i}`;
      jumpToPageSelect.appendChild(option);
    }

    jumpToEntrySelect.innerHTML = '';
    state.allChapters.forEach(ch => {
      const option = document.createElement('option');
      option.value = ch.id;
      option.textContent = ch.path.split(/[\\/]/).pop().replace(/\.[^/.]+$/, '');
      if (ch.id == chapterId) option.selected = true;
      jumpToEntrySelect.appendChild(option);
    });

    modeSelect.value = state.readingMode;
    marginSlider.value = state.pageMargin;
    fitModeSelect.value = state.fitMode;
    if (directionSelect) directionSelect.value = state.readingDirection;
  };

  // --- Progress Calculation ---
  const calculateAndUpdateProgress = () => {
    let progress = 0;
    if (state.readingMode === 'single_page') {
      progress = (state.currentPage / state.chapterData.page_count) * 100;
    } else if (state.readingMode === 'double_page') {
      const pages = getPagesForSpread(currentSpread);
      progress = (pages[pages.length - 1] / state.chapterData.page_count) * 100;
    } else {
      const scrollableHeight = document.documentElement.scrollHeight - window.innerHeight;
      if (scrollableHeight <= 0) { progressBar.style.width = '100%'; return; }
      progress = Math.round((window.scrollY / scrollableHeight) * 100);
    }
    progressBar.style.width = `${progress}%`;

    clearTimeout(window.progressTimeout);
    window.progressTimeout = setTimeout(() => updateProgress(progress, progress >= 99), 500);
    updateProgressText(progress);
    updateJumpToPageSelect(progress);
  };

  // --- Chapter Navigation Helpers ---
  const genPrevChapterId = () => {
    const cur = jumpToEntrySelect.options[jumpToEntrySelect.selectedIndex];
    return cur.previousElementSibling
      ? cur.previousElementSibling.value
      : state.allChapters[state.allChapters.length - 1].id;
  };

  const genNextChapterId = () => {
    const cur = jumpToEntrySelect.options[jumpToEntrySelect.selectedIndex];
    return cur.nextElementSibling
      ? cur.nextElementSibling.value
      : state.allChapters[0].id;
  };

  const jumpToChapter = newId => {
    if (newId && newId !== chapterId)
      window.location.href = `/reader/series/${folderId}/chapters/${newId}`;
  };

  const exitToLibrary = () => (window.location.href = `/library/folder/${folderId}`);

  // Advance forward / back respecting RTL arrow mapping.
  const advanceForward = () => {
    if (state.readingMode === 'single_page') {
      if (state.currentPage < state.chapterData.page_count) {
        state.currentPage++;
        updateSinglePageView();
      }
    } else if (state.readingMode === 'double_page') {
      if (currentSpread < getSpreadCount() - 1) {
        currentSpread++;
        updateDoublePageView();
      }
    }
  };

  const advanceBack = () => {
    if (state.readingMode === 'single_page') {
      if (state.currentPage > 1) {
        state.currentPage--;
        updateSinglePageView();
      }
    } else if (state.readingMode === 'double_page') {
      if (currentSpread > 0) {
        currentSpread--;
        updateDoublePageView();
      }
    }
  };

  // --- Event Listeners ---
  window.addEventListener('scroll', calculateAndUpdateProgress);

  // Click on image area opens settings modal, unless the user is panning while zoomed.
  imageContainer.addEventListener('click', () => {
    if (panMoved) { panMoved = false; return; }
    if (state.zoomLevel > 1) return;
    modal.style.display = 'flex';
  });

  modalCloseBtn.addEventListener('click', () => (modal.style.display = 'none'));
  modal.addEventListener('click', e => { if (e.target === modal) modal.style.display = 'none'; });

  // Keyboard shortcuts
  document.addEventListener('keydown', e => {
    // Zoom: Ctrl/Cmd + = (plus), - (minus), 0 (reset)
    if (e.ctrlKey || e.metaKey) {
      if (e.key === '=' || e.key === '+') { e.preventDefault(); zoomIn(); return; }
      if (e.key === '-') { e.preventDefault(); zoomOut(); return; }
      if (e.key === '0') { e.preventDefault(); zoomReset(); return; }
    }

    if (e.key === 'Escape') { modal.style.display = 'none'; return; }

    // In RTL mode, the "next page" key is ArrowLeft and "prev page" is ArrowRight.
    const nextKey = isRTL() ? 'ArrowLeft' : 'ArrowRight';
    const prevKey = isRTL() ? 'ArrowRight' : 'ArrowLeft';
    const nextAlpha = isRTL() ? 'a' : 'd';
    const prevAlpha = isRTL() ? 'd' : 'a';

    if (state.readingMode === 'continuous') {
      const atBottom = window.scrollY + window.innerHeight >= document.documentElement.scrollHeight - 10;
      const atTop = window.scrollY <= 10;
      if (e.key === nextKey || e.key === nextAlpha) {
        if (nextChapterId && atBottom) window.location.href = `/reader/series/${folderId}/chapters/${nextChapterId}`;
        else window.scrollBy({ top: window.innerHeight, behavior: 'smooth' });
      } else if (e.key === prevKey || e.key === prevAlpha) {
        if (prevChapterId && atTop) window.location.href = `/reader/series/${folderId}/chapters/${prevChapterId}`;
        else window.scrollBy({ top: -window.innerHeight, behavior: 'smooth' });
      }
    }

    if (state.readingMode === 'single_page' || state.readingMode === 'double_page') {
      if (e.key === nextKey || e.key === nextAlpha) advanceForward();
      else if (e.key === prevKey || e.key === prevAlpha) advanceBack();
    }
  });

  // Ctrl+scroll to zoom
  window.addEventListener('wheel', e => {
    if (!e.ctrlKey && !e.metaKey) return;
    e.preventDefault();
    e.deltaY < 0 ? zoomIn() : zoomOut();
  }, { passive: false });

  // Drag to pan when zoomed in
  imageContainer.addEventListener('pointerdown', e => {
    if (state.zoomLevel <= 1 || e.button !== 0) return;
    isPanning = true;
    panMoved = false;
    panStartX = e.clientX - state.panX;
    panStartY = e.clientY - state.panY;
    imageContainer.setPointerCapture(e.pointerId);
  });

  imageContainer.addEventListener('pointermove', e => {
    if (!isPanning) return;
    const newX = e.clientX - panStartX;
    const newY = e.clientY - panStartY;
    if (Math.abs(newX - state.panX) > 2 || Math.abs(newY - state.panY) > 2) panMoved = true;
    state.panX = newX;
    state.panY = newY;
    applyZoom();
  });

  imageContainer.addEventListener('pointerup', () => {
    isPanning = false;
    applyZoom();
  });

  // Page navigation buttons
  singlePrevBtn.addEventListener('click', () => {
    if (state.readingMode === 'continuous') jumpToChapter(genPrevChapterId());
    else advanceBack();
  });

  singleNextBtn.addEventListener('click', () => {
    if (state.readingMode === 'continuous') jumpToChapter(genNextChapterId());
    else advanceForward();
  });

  // Settings modal controls
  modeSelect.addEventListener('change', e => {
    const prev = state.readingMode;
    state.readingMode = e.target.value;
    // Re-render DOM when entering or leaving double_page (different node structure)
    if (prev === 'double_page' || state.readingMode === 'double_page') {
      renderPages();
    } else {
      applyReadingMode();
    }
    applyFitMode();
  });

  marginSlider.addEventListener('input', e => {
    state.pageMargin = e.target.value;
    applyPageMargin();
  });

  fitModeSelect.addEventListener('change', e => {
    state.fitMode = e.target.value;
    applyFitMode();
  });

  if (directionSelect) {
    directionSelect.addEventListener('change', e => {
      state.readingDirection = e.target.value;
      localStorage.setItem('readingDirection', state.readingDirection);
      // Re-render to apply new direction (spread page order, etc.)
      renderPages();
      applyFitMode();
    });
  }

  jumpToPageSelect.addEventListener('change', e => {
    const pageNum = parseInt(e.target.value, 10);
    if (state.readingMode === 'single_page') {
      state.currentPage = pageNum || 1;
      updateSinglePageView();
    } else if (state.readingMode === 'double_page') {
      currentSpread = getSpreadForPage(pageNum);
      updateDoublePageView();
    } else {
      document.getElementById(`page-${pageNum}`)?.scrollIntoView({ behavior: 'smooth' });
    }
    modal.style.display = 'none';
  });

  jumpToEntrySelect.addEventListener('change', e => {
    const newId = e.target.value;
    if (newId !== chapterId) window.location.href = `/reader/series/${folderId}/chapters/${newId}`;
  });

  modalExitBtn.addEventListener('click', exitToLibrary);
  modalPrevBtn.addEventListener('click', () => jumpToChapter(genPrevChapterId()));
  modalNextBtn.addEventListener('click', () => jumpToChapter(genNextChapterId()));
  footerPrevBtn.addEventListener('click', () => jumpToChapter(genPrevChapterId()));
  footerNextBtn.addEventListener('click', () => jumpToChapter(genNextChapterId()));
  footerExitBtn.addEventListener('click', exitToLibrary);

  // --- Initialization ---
  const init = async () => {
    await fetchInitialData();
    document.title = `${state.folderData.name} - Mango Reader`;
    await findNeighboringChapters();
    applyPageMargin();
    renderPages();
    applyFitMode();
    populateModal();

    await waitForImagesToLoad();

    const savedProgress = state.chapterData.progress_percent || 0;
    if (state.readingMode === 'continuous') {
      const scrollableHeight = document.documentElement.scrollHeight - window.innerHeight;
      window.scrollTo(0, (scrollableHeight * savedProgress) / 100);
    } else if (state.readingMode === 'double_page') {
      const page = Math.ceil((savedProgress / 100) * state.chapterData.page_count) || 1;
      currentSpread = getSpreadForPage(page);
      updateDoublePageView();
    } else {
      const pageNum = Math.ceil((savedProgress / 100) * state.chapterData.page_count) || 1;
      jumpToPageSelect.value = pageNum;
      state.currentPage = pageNum;
      updateSinglePageView();
    }

    calculateAndUpdateProgress();
    updateProgressText();
  };

  init();
});
