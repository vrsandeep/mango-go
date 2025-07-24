import { checkAuth } from './auth.js';

document.addEventListener('DOMContentLoaded', async () => {
  const currentUser = await checkAuth();
  if (!currentUser) return;

  const queueTableBody = document.getElementById('queue-table-body');
  const headerActionButtons = document.querySelectorAll('.header-actions button');
  const pauseResumeBtn = document.getElementById('pause-resume-btn');
  let ws;

  const renderRow = (item) => {
    let row = document.getElementById(`item-${item.id}`);
    if (!row) {
      row = document.createElement('tr');
      row.id = `item-${item.id}`;
    }
    const statusClass = `status-${item.status.replace(' ', '_').toLowerCase()}`;
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
                    <td><!-- Per-item action icons can be added here --></td>
                `;
    return row;
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
      console.error("Failed to load queue:", error);
      queueTableBody.innerHTML = '<tr><td colspan="7">Error loading queue.</td></tr>';
    }
  };

  const handleWebSocket = () => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${protocol}//${window.location.host}/ws/admin/progress`);

    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.job_name !== 'downloader' || !data.item_id) return;

      let row = document.getElementById(`item-${data.item_id}`);
      if (!row) {
        // The item is new to us, create a new row
        row = renderRow({
          id: data.item_id,
          progress: data.progress,
          status: data.status
          // We don't have all the info, a full refresh is better
        });
        loadQueue(); // Refresh the whole list to get all data
        return;
      }

      const progressBar = row.querySelector('.progress-bar');
      const statusEl = row.querySelector('[class^="status-"]');

      progressBar.style.width = `${data.progress}%`;
      progressBar.textContent = `${Math.round(data.progress)}%`;
      statusEl.textContent = data.status;
      statusEl.className = `status-${data.status.replace(' ', '_').toLowerCase()}`;
    };

    ws.onclose = () => {
      console.log('WebSocket for downloads disconnected. Reconnecting in 5 seconds...');
      setTimeout(handleWebSocket, 5000);
    };
    ws.onerror = (err) => {
      console.error('WebSocket error:', err);
    };
  };

  headerActionButtons.forEach(button => {
    button.addEventListener('click', async (e) => {
      const action = e.target.dataset.action;
      if (action === 'refresh') {
        loadQueue();
        return;
      }
      if (action === "pause_all") {
        const newAction = e.target.textContent === 'Pause All' ? 'pause_all' : 'resume_all';
        await fetch('/api/downloads/action', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ action: newAction })
        });
        e.target.textContent = newAction === 'pause_all' ? 'Resume All' : 'Pause All';
        return;
      }
      if (action === 'empty_queue') {
        if (!confirm('Are you sure you want to remove all queued and failed items? This cannot be undone.')) {
          return;
        }
      }

      await fetch('/api/downloads/action', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action })
      });
      setTimeout(loadQueue, 500); // Give backend a moment before refreshing
    });
  });

  loadQueue();
  handleWebSocket();
});