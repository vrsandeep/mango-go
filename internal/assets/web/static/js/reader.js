import { checkAuth } from './auth.js';

document.addEventListener('DOMContentLoaded', async () => {
  const currentUser = await checkAuth();
  if (!currentUser) return;

  // Apply theme from localStorage
  const savedTheme = localStorage.getItem('theme');
  if (savedTheme === 'light') {
    document.body.classList.add('light-theme');
  } else {
    document.body.classList.remove('light-theme');
  }

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
  };

  let nextChapterId = null;
  let prevChapterId = null;

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

  const footerPrevBtn = document.getElementById('footer-prev-chapter-btn');
  const footerNextBtn = document.getElementById('footer-next-chapter-btn');
  const footerExitBtn = document.getElementById('footer-exit-chapter-btn');

  // --- Core Functions ---
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

  const waitForImagesToLoad = () => {
    return new Promise(resolve => {
      // update to select those that do not have display:none
      const images = document.querySelectorAll('.page-image:not([style*="display: none"])');
      if (images.length === 0) {
        resolve();
        return;
      }

      let loadedCount = 0;
      const totalImages = images.length;

      const checkAllLoaded = () => {
        loadedCount++;
        if (loadedCount === totalImages) {
          resolve();
        }
      };

      images.forEach(img => {
        if (img.complete) {
          checkAllLoaded();
        } else {
          img.addEventListener('load', checkAllLoaded);
          img.addEventListener('error', checkAllLoaded); // Handle errors too
        }
      });
    });
  };

  const updateProgress = (progressPercent, isRead) => {
    fetch(`/api/chapters/${chapterId}/progress`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ progress_percent: progressPercent, read: isRead }),
    });
  };
  const updateProgressText = progressPercent => {
    let progress = state.chapterData.progress_percent;
    if (state.readingMode === 'single_page') {
      progress = (state.currentPage / state.chapterData.page_count) * 100;
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

  const renderPages = () => {
    imageContainer.innerHTML = '';
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

  const applyReadingMode = () => {
    localStorage.setItem('readingMode', state.readingMode);
    if (state.readingMode === 'single_page') {
      imageContainer.classList.add('single-page');
      singlePageViewer.style.display = 'flex';
      updateSinglePageView();
    } else {
      // continuous
      imageContainer.classList.remove('single-page');
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
    // Scroll to top of page
    window.scrollTo(0, 0);

    updateProgressText();
    updateJumpToPageSelect();
  };

  const updateJumpToPageSelect = progressPercent => {
    let progress = state.chapterData.progress_percent;
    if (state.readingMode === 'single_page') {
      progress = (state.currentPage / state.chapterData.page_count) * 100;
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
      // Reset all styles first to ensure clean state
      img.style.width = '';
      img.style.height = '';
      img.style.maxWidth = '';
      img.style.maxHeight = '';
      img.style.objectFit = '';

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
          img.style.maxHeight = 'auto';
          img.style.objectFit = 'contain';
          break;
      }
    });
  };

  const populateModal = () => {
    const chapter = state.chapterData;
    const folder = state.folderData;
    const lastPart = chapter.path
      .split(/[\\/]/)
      .pop()
      .replace(/\.[^/.]+$/, '');
    modalTitle.textContent = folder.name + ' - ' + lastPart;
    modalPath.textContent = chapter.path;

    // Populate Jump to Page dropdown
    jumpToPageSelect.innerHTML = '';
    for (let i = 1; i <= chapter.page_count; i++) {
      const option = document.createElement('option');
      option.value = i;
      option.textContent = `Page ${i}`;
      jumpToPageSelect.appendChild(option);
    }

    // Populate Jump to Entry dropdown
    jumpToEntrySelect.innerHTML = '';
    state.allChapters.forEach(ch => {
      const option = document.createElement('option');
      option.value = ch.id;
      option.textContent = ch.path
        .split(/[\\/]/)
        .pop()
        .replace(/\.[^/.]+$/, '');
      if (ch.id == chapterId) {
        option.selected = true;
      }
      jumpToEntrySelect.appendChild(option);
    });

    modeSelect.value = state.readingMode;
    marginSlider.value = state.pageMargin;
    fitModeSelect.value = state.fitMode;
  };

  const calculateAndUpdateProgress = () => {
    let progress = 0;
    if (state.readingMode === 'single_page') {
      progress = (state.currentPage / state.chapterData.page_count) * 100;
    } else {
      const scrollableHeight = document.documentElement.scrollHeight - window.innerHeight;
      progress = Math.round((window.scrollY / scrollableHeight) * 100);
      if (scrollableHeight <= 0) {
        progressBar.style.width = '100%';
        return;
      }
    }
    progressBar.style.width = `${progress}%`;

    // Persist progress (debounced)
    clearTimeout(window.progressTimeout);
    window.progressTimeout = setTimeout(() => {
      const isRead = progress >= 99;
      updateProgress(progress, isRead);
    }, 500);
    updateProgressText(progress);
    updateJumpToPageSelect(progress);
  };

  // --- Event Listeners ---
  window.addEventListener('scroll', calculateAndUpdateProgress);
  imageContainer.addEventListener('click', () => (modal.style.display = 'flex'));
  modalCloseBtn.addEventListener('click', () => (modal.style.display = 'none'));
  // Close modal on overlay click
  modal.addEventListener('click', e => {
    if (e.target === modal) modal.style.display = 'none';
  });

  document.addEventListener('keydown', e => {
    // apply the following if continuous mode
    if (state.readingMode === 'continuous') {
      if (e.key === 'ArrowRight' || e.key === 'd') {
        if (
          nextChapterId &&
          window.scrollY + window.innerHeight >= document.documentElement.scrollHeight - 10
        ) {
          window.location.href = `/reader/series/${folderId}/chapters/${nextChapterId}`;
        } else {
          window.scrollBy({ top: window.innerHeight, behavior: 'smooth' });
        }
      } else if (e.key === 'ArrowLeft' || e.key === 'a') {
        if (prevChapterId && window.scrollY <= 10) {
          window.location.href = `/reader/series/${folderId}/chapters/${prevChapterId}`;
        } else {
          window.scrollBy({ top: -window.innerHeight, behavior: 'smooth' });
        }
      }
    }

    // apply the following if single page mode
    if (state.readingMode === 'single_page') {
      if (e.key === 'ArrowRight' || e.key === 'd') {
        if (state.currentPage < state.chapterData.page_count) {
          state.currentPage++;
          updateSinglePageView();
        }
      } else if (e.key === 'ArrowLeft' || e.key === 'a') {
        if (state.currentPage > 1) {
          state.currentPage--;
          updateSinglePageView();
        }
      }
    }
    if (e.key === 'Escape') {
      modal.style.display = 'none';
    }
  });

  singlePrevBtn.addEventListener('click', () => {
    if (state.currentPage > 1) {
      state.currentPage--;
      updateSinglePageView();
    }
  });
  singleNextBtn.addEventListener('click', () => {
    if (state.currentPage < state.chapterData.page_count) {
      state.currentPage++;
      updateSinglePageView();
    }
  });
  modeSelect.addEventListener('change', e => {
    state.readingMode = e.target.value;
    applyReadingMode();
  });
  marginSlider.addEventListener('input', e => {
    state.pageMargin = e.target.value;
    applyPageMargin();
  });
  fitModeSelect.addEventListener('change', e => {
    state.fitMode = e.target.value;
    applyFitMode();
  });
  jumpToPageSelect.addEventListener('change', e => {
    const pageNum = parseInt(e.target.value, 10);
    if (state.readingMode === 'single_page') {
      state.currentPage = pageNum || 1;
      updateSinglePageView();
    } else {
      document.getElementById(`page-${pageNum}`).scrollIntoView({ behavior: 'smooth' });
    }
    modal.style.display = 'none';
  });
  jumpToEntrySelect.addEventListener('change', e => {
    const newChapterId = e.target.value;
    if (newChapterId !== chapterId) {
      window.location.href = `/reader/series/${folderId}/chapters/${newChapterId}`;
    }
  });

  var genPrevChapterId = () => {
    const currentOption = jumpToEntrySelect.options[jumpToEntrySelect.selectedIndex];
    const prevOption = currentOption.previousElementSibling;
    var prevChapterId = null;
    if (prevOption) {
      prevChapterId = prevOption.value;
    } else {
      prevChapterId = state.allChapters[state.allChapters.length - 1].id;
    }
    return prevChapterId;
  };
  var genNextChapterId = () => {
    const currentOption = jumpToEntrySelect.options[jumpToEntrySelect.selectedIndex];
    const nextOption = currentOption.nextElementSibling;
    var nextChapterId = null;
    if (nextOption) {
      nextChapterId = nextOption.value;
    } else {
      nextChapterId = state.allChapters[0].id;
    }
    return nextChapterId;
  };
  var jumpToChapter = newChapterId => {
    if (newChapterId && newChapterId !== chapterId) {
      window.location.href = `/reader/series/${folderId}/chapters/${newChapterId}`;
    }
  };
  const exitToLibrary = () => (window.location.href = `/library/folder/${folderId}`);

  modalExitBtn.addEventListener('click', exitToLibrary);
  modalPrevBtn.addEventListener('click', () => {
    const newChapterId = genPrevChapterId();
    jumpToChapter(newChapterId);
  });
  modalNextBtn.addEventListener('click', () => {
    const newChapterId = genNextChapterId();
    jumpToChapter(newChapterId);
  });

  footerPrevBtn.addEventListener('click', () => {
    const newChapterId = genPrevChapterId();
    jumpToChapter(newChapterId);
  });
  footerNextBtn.addEventListener('click', () => {
    const newChapterId = genNextChapterId();
    jumpToChapter(newChapterId);
  });
  footerExitBtn.addEventListener('click', exitToLibrary);

  // markReadBtn.addEventListener('click', () => {
  //   updateProgress(100, true);
  //   progressBar.style.width = '100%';
  //   modal.style.display = 'none';
  // });

  // --- Initialization ---
  const init = async () => {
    await fetchInitialData();
    document.title = `${state.folderData.name} - Mango Reader`;
    await findNeighboringChapters();
    applyPageMargin();
    renderPages();
    applyFitMode();
    populateModal();

    // Wait for all images to load before restoring scroll position
    await waitForImagesToLoad();

    const savedProgress = state.chapterData.progress_percent || 0;
    if (state.readingMode === 'continuous') {
      const scrollableHeight = document.documentElement.scrollHeight - window.innerHeight;
      window.scrollTo(0, (scrollableHeight * savedProgress) / 100);
    } else {
      var pageNum = Math.ceil((savedProgress / 100) * state.chapterData.page_count) || 1;
      jumpToPageSelect.value = pageNum;
      state.currentPage = pageNum;
      updateSinglePageView();
    }
    calculateAndUpdateProgress(); // Initial calculation

    updateProgressText();
  };

  init();
});
