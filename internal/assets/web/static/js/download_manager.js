import { checkAuth } from './auth.js';

document.addEventListener('DOMContentLoaded', async () => {
  const currentUser = await checkAuth();
  if (!currentUser) return;

  const queueTableBody = document.getElementById('queue-table-body');
  const headerActionButtons = document.querySelectorAll('.header-actions button');
  const pauseResumeBtn = document.getElementById('pause-resume-btn');
  let ws;

  const renderRow = item => {
    let row = document.getElementById(`item-${item.id}`);
    if (!row) {
      row = document.createElement('tr');
      row.id = `item-${item.id}`;
    }
    const statusClass = `status-${item.status.replace(' ', '_').toLowerCase()}`;

    let actionButtons = generateActionCellContent(item.id, item.status);

    row.innerHTML = `
      <td title="${item.chapter_title}">${item.chapter_title}</td>
      <td title="${item.series_title}">${item.series_title}</td>
      <td>
          <div class="progress-bar-container">
              <div class="progress-bar" style="width: ${item.progress}%;">${item.progress}%</div>
          </div>
      </td>
      <td>${new Date(item.created_at).toLocaleString()}</td>
      <td><span class="${statusClass}">${item.status}</span></td>
      <td>${item.provider_id}</td>
      <td class="actions-cell">${actionButtons}</td>
    `;
    return row;
  };

  const generateActionCellContent = (itemId, status) => {
    let actionButtons = '';
    if (status === 'queued' || status === 'completed') {
      actionButtons = `<button class="action-btn delete-btn" data-action="delete" data-id="${itemId}" title="Delete">🗑️</button>`;
    } else if (status === 'failed') {
      actionButtons = `<button class="action-btn retry-btn" data-action="retry" data-id="${itemId}" title="Retry"><i class="ph-bold ph-arrow-clockwise"></i></button> <button class="action-btn delete-btn" data-action="delete" data-id="${itemId}" title="Delete">🗑️</button>`;
    } else if (status === 'in_progress') {
      actionButtons = `<button class="action-btn pause-btn" data-action="pause" data-id="${itemId}" title="Pause">⏸️</button>`;
    } else if (status === 'paused') {
      actionButtons = `<button class="action-btn resume-btn" data-action="resume" data-id="${itemId}" title="Resume">▶️</button>`;
    }

    return actionButtons;
  };

  const loadQueue = async () => {
    try {
      const response = await fetch('/api/downloads/queue');
      const items = await response.json();
      queueTableBody.innerHTML = '';
      if (items) {
        items.forEach(item => queueTableBody.appendChild(renderRow(item)));
      }
    } catch (error) {
      console.error('Failed to load queue:', error);
      queueTableBody.innerHTML = '<tr><td colspan="7">Error loading queue.</td></tr>';
    }
  };

  const handleWebSocket = () => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${protocol}//${window.location.host}/ws/admin/progress`);

    ws.onopen = () => {
      console.log('WebSocket connected for download manager');
    };

    ws.onmessage = event => {
      const data = JSON.parse(event.data);
      if (data.jobId !== 'downloader' || !data.item_id) return;

      let row = document.getElementById(`item-${data.item_id}`);
      if (!row) {
        // The item is new to us, refresh the whole list to get all data
        loadQueue();
        return;
      }

      const progressBar = row.querySelector('.progress-bar');
      const statusEl = row.querySelector('[class^="status-"]');

      if (progressBar) {
        progressBar.style.width = `${data.progress}%`;
        progressBar.textContent = `${Math.round(data.progress)}%`;
      }

      if (statusEl) {
        statusEl.textContent = data.status;
        statusEl.className = `status-${data.status.replace(' ', '_').toLowerCase()}`;
      }

      // Update action buttons based on new status
      const actionsCell = row.querySelector('.actions-cell');
      if (actionsCell) {
        actionsCell.innerHTML = generateActionCellContent(data.item_id, data.status);
      }
    };

    ws.onclose = () => {
      console.log('WebSocket for downloads disconnected. Reconnecting in 5 seconds...');
      setTimeout(handleWebSocket, 5000);
    };
    ws.onerror = err => {
      console.error('WebSocket error:', err);
    };
  };

  // Helper function to ensure websocket is connected
  const ensureWebSocketConnected = () => {
    if (!ws || ws.readyState === WebSocket.CLOSED || ws.readyState === WebSocket.CLOSING) {
      handleWebSocket();
    } else if (ws.readyState === WebSocket.CONNECTING) {
      // Already connecting, wait for it to open
      ws.addEventListener(
        'open',
        () => {
          console.log('WebSocket connection established');
        },
        { once: true }
      );
    }
  };

  headerActionButtons.forEach(button => {
    button.addEventListener('click', async e => {
      const action = e.target.dataset.action;
      if (action === 'refresh') {
        loadQueue();
        return;
      }
      if (action === 'pause_all') {
        const newAction = e.target.textContent === 'Pause All' ? 'pause_all' : 'resume_all';
        await fetch('/api/downloads/action', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ action: newAction }),
        });
        e.target.textContent = newAction === 'pause_all' ? 'Resume All' : 'Pause All';
        return;
      }
      if (action === 'empty_queue') {
        if (
          !confirm(
            'Are you sure you want to remove all queued and failed items? This cannot be undone.'
          )
        ) {
          return;
        }
      }
      if (action === 'reload_plugins') {
        const reloadBtn = e.target;
        if (
          !confirm(
            'Are you sure you want to reload all plugins? This may interrupt ongoing downloads.'
          )
        ) {
          return;
        }
        reloadBtn.disabled = true;
        reloadBtn.textContent = 'Reloading...';
        try {
          const response = await fetch('/api/admin/plugins/reload', {
            method: 'POST',
          });
          if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to reload plugins');
          }
          if (window.toast) {
            toast.success('All plugins reloaded successfully');
          }
        } catch (error) {
          console.error('Failed to reload plugins:', error);
          if (window.toast) {
            toast.error(error.message || 'Failed to reload plugins');
          }
        } finally {
          reloadBtn.disabled = false;
          reloadBtn.textContent = 'Reload All Plugins';
        }
        return;
      }

      await fetch('/api/downloads/action', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action }),
      });

      // For retry_failed, ensure websocket is connected and reload queue immediately
      if (action === 'retry_failed') {
        ensureWebSocketConnected();
        // Reload queue immediately to ensure all items are in DOM for websocket updates
        setTimeout(() => {
          loadQueue();
        }, 300);
      } else {
        setTimeout(loadQueue, 500); // Give backend a moment before refreshing
      }
    });
  });

  // Add event delegation for individual item action buttons
  queueTableBody.addEventListener('click', async e => {
    const button = e.target.closest('.action-btn');
    if (!button) return;

    const action = button.dataset.action;
    const itemId = button.dataset.id;

    if (!itemId) return;

    // Disable button to prevent double-clicks
    button.disabled = true;

    try {
      if (action === 'delete') {
        if (!confirm('Are you sure you want to delete this item from the queue?')) {
          button.disabled = false;
          return;
        }
      }

      await fetch(`/api/downloads/queue/${itemId}/action`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action }),
      });

      // Refresh the queue to show updated status
      setTimeout(loadQueue, 100);
    } catch (error) {
      console.error(`Failed to ${action} item ${itemId}:`, error);
      toast.error(`Failed to ${action} item. Please try again.`);
      button.disabled = false;
    }
  });

  loadQueue();
  handleWebSocket();
});
