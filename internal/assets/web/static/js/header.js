document.addEventListener('DOMContentLoaded', () => {
  const menuToggleBtn = document.getElementById('menu-toggle-btn');
  const navLinks = document.getElementById('nav-links');
  const themeToggleBtn = document.getElementById('theme-toggle-btn');
  const logoutBtn = document.getElementById('logout-btn');

  // --- Mobile Menu Logic ---
  menuToggleBtn.addEventListener('click', () => navLinks.classList.toggle('active'));

  // --- Theme Logic ---
  const applyTheme = theme => {
    document.body.classList.toggle('light-theme', theme === 'light');
    themeToggleBtn.innerHTML =
      theme === 'light' ? '<i class="ph-bold ph-moon"></i>' : '<i class="ph-bold ph-sun"></i>';
  };

  themeToggleBtn.addEventListener('click', () => {
    const newTheme = document.body.classList.contains('light-theme') ? 'dark' : 'light';
    localStorage.setItem('theme', newTheme);
    applyTheme(newTheme);
  });

  const loadVersion = async () => {
    const response = await fetch('/api/version');
    const data = await response.json();
    document.getElementById('version-footer').textContent = `Version: ${data.version}`;
  };
  logoutBtn.addEventListener('click', () => {
    // Handle logout logic
  });

  // --- Dropdown Menu Logic ---
  document.querySelectorAll('.header-dropdown').forEach(dropdown => {
    const btn = dropdown.querySelector('.header-dropdown-btn');
    const content = dropdown.querySelector('.header-dropdown-content');
    let open = false;
    let hoverTimeout;

    // Helper to open/close
    function setOpen(state) {
      open = state;
      if (open) {
        content.style.display = 'block';
        btn.setAttribute('aria-expanded', 'true');
      } else {
        content.style.display = 'none';
        btn.setAttribute('aria-expanded', 'false');
      }
    }

    // Mouse enter/leave for button and content
    btn.addEventListener('mouseenter', () => {
      clearTimeout(hoverTimeout);
      setOpen(true);
    });
    btn.addEventListener('mouseleave', () => {
      hoverTimeout = setTimeout(() => setOpen(false), 120);
    });
    content.addEventListener('mouseenter', () => {
      clearTimeout(hoverTimeout);
      setOpen(true);
    });
    content.addEventListener('mouseleave', () => {
      hoverTimeout = setTimeout(() => setOpen(false), 120);
    });

    // Click toggles
    btn.addEventListener('click', e => {
      e.preventDefault();
      setOpen(!open);
    });
  });

  // Init
  applyTheme(localStorage.getItem('theme'));
  loadVersion();
  initNotifications();
});

function initNotifications() {
  const wrap = document.querySelector('.notifications-wrap');
  const btn = document.getElementById('notifications-btn');
  const panel = document.getElementById('notifications-panel');
  const listEl = document.getElementById('notifications-list');
  const emptyEl = document.getElementById('notifications-empty');
  const viewMoreEl = document.getElementById('notifications-view-more');
  const itemTemplate = document.getElementById('notification-item-template');
  const dot = wrap?.querySelector('.notifications-dot');
  if (!wrap || !btn || !panel || !listEl || !emptyEl || !viewMoreEl || !itemTemplate || !dot) return;

  let open = false;

  const formatRelative = iso => {
    const then = new Date(iso).getTime();
    if (Number.isNaN(then)) return '';
    const diffSec = Math.max(0, Math.round((Date.now() - then) / 1000));
    if (diffSec < 60) return 'just now';
    const mins = Math.round(diffSec / 60);
    if (mins < 60) return `${mins}m ago`;
    const hours = Math.round(mins / 60);
    if (hours < 24) return `${hours}h ago`;
    const days = Math.round(hours / 24);
    if (days === 1) return 'yesterday';
    return `${days}d ago`;
  };

  const itemHref = item => {
    if (item.local_folder_id != null && item.local_chapter_id != null) {
      return `/reader/series/${item.local_folder_id}/chapters/${item.local_chapter_id}`;
    }
    if (item.local_folder_id != null) {
      return `/library/folder/${item.local_folder_id}`;
    }
    return '/downloads/manager';
  };

  const renderList = (items, hasMore) => {
    listEl.replaceChildren();
    const hasItems = Boolean(items && items.length);
    emptyEl.hidden = hasItems;
    listEl.hidden = !hasItems;
    viewMoreEl.hidden = !hasMore;
    if (!hasItems) return;

    items.forEach(item => {
      const node = itemTemplate.content.firstElementChild.cloneNode(true);
      node.href = itemHref(item);
      node.classList.toggle('unread', !item.read);
      node.querySelector('.notifications-item-title').textContent = item.chapter_title ?? '';
      node.querySelector('.notifications-item-series').textContent = item.series_title ?? '';
      node.querySelector('.notifications-item-time').textContent = formatRelative(item.created_at);
      listEl.appendChild(node);
    });
  };

  const setDot = hasUnread => {
    if (hasUnread) {
      dot.removeAttribute('hidden');
    } else {
      dot.setAttribute('hidden', '');
    }
  };

  const fetchNotifications = async () => {
    try {
      const response = await fetch('/api/notifications');
      if (!response.ok) return null;
      return await response.json();
    } catch (err) {
      console.error('Failed to load notifications:', err);
      return null;
    }
  };

  const refreshIndicator = async () => {
    const data = await fetchNotifications();
    if (!data) return;
    setDot(Boolean(data.has_unread));
    if (open) renderList(data.notifications, data.has_more);
  };

  const setOpen = async state => {
    open = state;
    panel.hidden = !open;
    btn.setAttribute('aria-expanded', open ? 'true' : 'false');
    if (!open) return;

    const data = await fetchNotifications();
    if (data) {
      renderList(data.notifications, data.has_more);
      setDot(Boolean(data.has_unread));
    }

    if (data && data.has_unread) {
      try {
        await fetch('/api/notifications/read', { method: 'POST' });
        setDot(false);
      } catch (err) {
        console.error('Failed to mark notifications as read:', err);
      }
    }
  };

  btn.addEventListener('click', e => {
    e.stopPropagation();
    setOpen(!open);
  });

  document.addEventListener('click', e => {
    if (open && !wrap.contains(e.target)) {
      setOpen(false);
    }
  });

  const markReadOnDownloadManager = async () => {
    if (window.location.pathname !== '/downloads/manager') return;
    try {
      const response = await fetch('/api/notifications/read', { method: 'POST' });
      if (response.ok) setDot(false);
    } catch (err) {
      console.error('Failed to mark notifications as read:', err);
    }
  };

  markReadOnDownloadManager().then(refreshIndicator);
  setInterval(refreshIndicator, 5 * 60 * 1000);
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible') refreshIndicator();
  });
}
