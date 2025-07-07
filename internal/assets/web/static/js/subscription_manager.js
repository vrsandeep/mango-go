document.addEventListener('DOMContentLoaded', async () => {
  const currentUser = await checkAuth();
  if (!currentUser) return;

  const providerSelect = document.getElementById('provider-select');
  const subTableBody = document.getElementById('sub-table-body');

  const timeAgo = (date) => {
    if (!date) return 'Never';
    const seconds = Math.floor((new Date() - date) / 1000);
    let interval = seconds / 31536000;
    if (interval > 1) return Math.floor(interval) + " years ago";
    interval = seconds / 2592000;
    if (interval > 1) return Math.floor(interval) + " months ago";
    interval = seconds / 86400;
    if (interval > 1) return Math.floor(interval) + " days ago";
    interval = seconds / 3600;
    if (interval > 1) return Math.floor(interval) + " hours ago";
    interval = seconds / 60;
    if (interval > 1) return Math.floor(interval) + " minutes ago";
    return Math.floor(seconds) + " seconds ago";
  };

  const renderTable = (subs) => {
    subTableBody.innerHTML = '';
    if (!subs || subs.length === 0) {
      subTableBody.innerHTML = '<tr><td colspan="5">No subscriptions found.</td></tr>';
      return;
    }
    subs.forEach(sub => {
      const row = document.createElement('tr');
      row.innerHTML = `
                        <td title="${sub.series_title}">${sub.series_title}</td>
                        <td>${sub.provider_id}</td>
                        <td>${timeAgo(new Date(sub.created_at))}</td>
                        <td>${timeAgo(sub.last_checked_at ? new Date(sub.last_checked_at) : null)}</td>
                        <td class="actions-cell">
                            <button data-action="recheck" data-id="${sub.id}" title="Re-check for new chapters"><i class="ph-bold ph-arrow-clockwise"></i></button>
                            <button data-action="delete" data-id="${sub.id}" title="Delete subscription"><i class="ph-bold ph-trash"></i></button>
                        </td>
                    `;
      subTableBody.appendChild(row);
    });
  };

  const loadSubscriptions = async () => {
    const providerID = providerSelect.value;
    const url = `/api/subscriptions${providerID ? '?provider_id=' + providerID : ''}`;
    try {
      const response = await fetch(url);
      const subs = await response.json();
      renderTable(subs);
    } catch (e) {
      console.error("Failed to load subscriptions", e);
    }
  };

  const loadProviders = async () => {
    try {
      const response = await fetch('/api/providers');
      const providers = await response.json();
      if (providers) {
        providers.forEach(p => {
          const option = document.createElement('option');
          option.value = p.id;
          option.textContent = p.name;
          providerSelect.appendChild(option);
        });
      }
    } catch (e) { console.error("Failed to load providers", e); }
  };

  subTableBody.addEventListener('click', async (e) => {
    const button = e.target.closest('button');
    if (!button) return;

    button.disabled = true; // Prevent double-clicks
    const action = button.dataset.action;
    const id = button.dataset.id;

    try {
      if (action === 'delete') {
        if (confirm('Are you sure you want to delete this subscription?')) {
          await fetch(`/api/subscriptions/${id}`, { method: 'DELETE' });
          loadSubscriptions();
        }
      } else if (action === 'recheck') {
        await fetch(`/api/subscriptions/${id}/recheck`, { method: 'POST' });
        alert('Re-check initiated. New chapters will be added to the download queue if found.');
      }
    } catch (e) {
      console.error(`Action ${action} failed:`, e);
      alert(`Action ${action} failed.`);
    } finally {
      button.disabled = false;
    }
  });

  providerSelect.addEventListener('change', () => {
    localStorage.setItem('sub_provider_filter', providerSelect.value);
    loadSubscriptions();
  });

  const init = async () => {
    await loadProviders();
    const savedProvider = localStorage.getItem('sub_provider_filter');
    if (savedProvider) providerSelect.value = savedProvider;
    await loadSubscriptions();
  };
  init();
});