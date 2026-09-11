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
  const headerRight = document.querySelector('.header-right');
  if (!headerRight) return;

  const wrap = document.createElement('div');
  wrap.className = 'notifications-wrap';
  wrap.innerHTML = `
    <button type="button" id="notifications-btn" class="notifications-btn" title="Notifications" aria-haspopup="true" aria-expanded="false">
      <i class="ph-bold ph-bell"></i>
      <span class="notifications-dot" hidden></span>
    </button>
    <div class="notifications-panel" id="notifications-panel" hidden>
      <div class="notifications-panel-header">New chapters</div>
      <div class="notifications-list" id="notifications-list"></div>
    </div>
  `;

  const searchBtn = document.getElementById('search-btn');
  headerRight.insertBefore(wrap, searchBtn || headerRight.firstChild);

  const btn = wrap.querySelector('#notifications-btn');
  const panel = wrap.querySelector('#notifications-panel');
  const listEl = wrap.querySelector('#notifications-list');
  const dot = wrap.querySelector('.notifications-dot');
  let open = false;

  const escapeHtml = str =>
    String(str ?? '')
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/"/g, '&quot;');

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

  const renderList = items => {
    if (!items || items.length === 0) {
      listEl.innerHTML = '<p class="notifications-empty">No new chapters in the last few days.</p>';
      return;
    }
    listEl.innerHTML = items
      .map(item => {
        const unreadClass = item.read ? '' : ' unread';
        return `<a class="notifications-item${unreadClass}" href="${itemHref(item)}">
          <span class="notifications-item-title">${escapeHtml(item.chapter_title)}</span>
          <span class="notifications-item-series">${escapeHtml(item.series_title)}</span>
          <span class="notifications-item-time">${escapeHtml(formatRelative(item.created_at))}</span>
        </a>`;
      })
      .join('');
  };

  const setDot = hasUnread => {
    if (hasUnread) {
      dot.removeAttribute('hidden');
      btn.classList.add('has-unread');
    } else {
      dot.setAttribute('hidden', '');
      btn.classList.remove('has-unread');
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
    if (open) renderList(data.notifications);
  };

  const setOpen = async state => {
    open = state;
    panel.hidden = !open;
    btn.setAttribute('aria-expanded', open ? 'true' : 'false');
    if (!open) return;

    const data = await fetchNotifications();
    if (data) {
      renderList(data.notifications);
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
  setInterval(refreshIndicator, 30000);
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible') refreshIndicator();
  });
}
