document.addEventListener('DOMContentLoaded', () => {
  const pathParts = window.location.pathname.split('/');
  const seriesId = pathParts[3];
  const chapterId = pathParts[5];

  let chapterData = null;
  let nextChapterId = null;
  let prevChapterId = null;

  const imageContainer = document.getElementById('image-container');
  const progressBar = document.getElementById('progress-bar');

  const footerPrevBtn = document.getElementById('footer-prev-chapter-btn');
  const footerNextBtn = document.getElementById('footer-next-chapter-btn');
  const footerExitBtn = document.getElementById('footer-exit-chapter-btn');
  // Modal elements
  const modal = document.getElementById('nav-modal');
  const prevChapterModalBtn = document.getElementById('prev-chapter-btn');
  const nextChapterModalBtn = document.getElementById('next-chapter-modal-btn');
  const exitReaderBtn = document.getElementById('exit-reader-btn');
  const markReadBtn = document.getElementById('mark-read-btn');

  // --- API Calls ---
  const fetchChapterDetails = async () => {
    const response = await fetch(`/api/series/${seriesId}/chapters/${chapterId}`);
    chapterData = await response.json();
  };

  const updateProgress = (progressPercent, isRead) => {
    fetch(`/api/chapters/${chapterId}/progress`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ progress_percent: progressPercent, read: isRead }),
    });
  };
  const findNeighboringChapters = async () => {
    const response = await fetch(`/api/series/${seriesId}/chapters/${chapterId}/neighbors`);
    const neighbors = await response.json();
    if (neighbors.prev) {
      prevChapterId = neighbors.prev;
      footerPrevBtn.style.display = 'inline-block';
      prevChapterModalBtn.disabled = false;
    } else {
      prevChapterModalBtn.disabled = true;
    }
    if (neighbors.next) {
      nextChapterId = neighbors.next;
      footerNextBtn.style.display = 'inline-block';
      nextChapterModalBtn.disabled = false;
    } else {
      nextChapterModalBtn.disabled = true;
      footerExitBtn.style.display = 'inline-block';
    }
  };

  const loadImages = () => {
    for (let i = 1; i <= chapterData.page_count; i++) {
      const img = document.createElement('img');
      img.src = `/api/series/${seriesId}/chapters/${chapterId}/pages/${i}`;
      img.classList.add('page-image');
      img.addEventListener('click', () => modal.style.display = 'flex');
      imageContainer.appendChild(img);
    }
  };

  const calculateAndUpdateProgress = () => {
    const scrollableHeight = document.documentElement.scrollHeight - window.innerHeight;
    if (scrollableHeight <= 0) {
      progressBar.style.width = '100%';
      return;
    }
    const progress = Math.round((window.scrollY / scrollableHeight) * 100);
    progressBar.style.width = `${progress}%`;

    // Persist progress (debounced)
    clearTimeout(window.progressTimeout);
    window.progressTimeout = setTimeout(() => {
      const isRead = progress >= 99;
      updateProgress(progress, isRead);
    }, 500);
  };

  // --- Event Listeners ---
  window.addEventListener('scroll', calculateAndUpdateProgress);
  document.addEventListener('keydown', (e) => {
    if (e.key === 'ArrowRight' || e.key === 'd') {
      if (nextChapterId && window.scrollY + window.innerHeight >= document.documentElement.scrollHeight - 10) {
        window.location.href = `/reader/series/${seriesId}/chapters/${nextChapterId}`;
      } else {
        window.scrollBy({ top: window.innerHeight, behavior: 'smooth' });
      }
    } else if (e.key === 'ArrowLeft' || e.key === 'a') {
      if (prevChapterId && window.scrollY <= 10) {
        window.location.href = `/reader/series/${seriesId}/chapters/${prevChapterId}`;
      } else {
        window.scrollBy({ top: -window.innerHeight, behavior: 'smooth' });
      }
    } else if (e.key === 'Escape') {
      modal.style.display = 'none';
    }
  });

  nextChapterModalBtn.addEventListener('click', () => {
    if (nextChapterId) window.location.href = `/reader/series/${seriesId}/chapters/${nextChapterId}`;
  });
  prevChapterModalBtn.addEventListener('click', () => {
    if (prevChapterId) window.location.href = `/reader/series/${seriesId}/chapters/${prevChapterId}`;
  });

  footerPrevBtn.addEventListener('click', () => {
    if (prevChapterId) window.location.href = `/reader/series/${seriesId}/chapters/${prevChapterId}`;
  });
  footerNextBtn.addEventListener('click', () => {
    if (nextChapterId) window.location.href = `/reader/series/${seriesId}/chapters/${nextChapterId}`;
  });
  footerExitBtn.addEventListener('click', () => {
    window.location.href = `/series/${seriesId}`;
  });

  markReadBtn.addEventListener('click', () => {
    updateProgress(100, true);
    progressBar.style.width = '100%';
    modal.style.display = 'none';
  });

  exitReaderBtn.addEventListener('click', () => {
    window.location.href = `/series/${seriesId}`;
  });

  // Close modal on overlay click
  modal.addEventListener('click', (e) => {
    if (e.target === modal) {
      modal.style.display = 'none';
    }
  });

  // --- Initialization ---
  const init = async () => {
    await fetchChapterDetails();
    // await fetchSeriesDetails();
    await findNeighboringChapters();
    document.title = `Chapter ${chapterData.id}`;
    loadImages();
    // Restore scroll position after images have loaded
    setTimeout(() => {
      const scrollableHeight = document.documentElement.scrollHeight - window.innerHeight;
      const savedProgress = chapterData.progress_percent || 0;
      window.scrollTo(0, (scrollableHeight * savedProgress) / 100);
      calculateAndUpdateProgress(); // Initial calculation
    }, 500); // Delay to allow images to start rendering
  };

  init();
});