import { checkAuth } from './auth.js';

document.addEventListener('DOMContentLoaded', async () => {
  const currentUser = await checkAuth('admin');
  if (!currentUser) return;

  // Load and restore running job states on page load
  const loadJobStatuses = async () => {
    try {
      const response = await fetch('/api/admin/jobs/status');
      if (!response.ok) return;
      const statuses = await response.json();

      statuses.forEach(status => {
        if (status.status === 'running') {
          const jobEl = document.getElementById(status.id);
          if (!jobEl) return;

          const progressContainer = jobEl.querySelector('.job-progress-container');
          const progressBar = jobEl.querySelector('.job-progress-bar');
          const description = jobEl.querySelector('.job-description');
          const button = jobEl.querySelector('.start-job-btn');

          // Show progress container for running jobs
          if (progressContainer) {
            progressContainer.style.display = 'block';
          }

          // Update description with current message
          if (description && status.message) {
            description.textContent = status.message;
          }

          // Disable button while job is running
          if (button) {
            button.disabled = true;
          }

          // Set initial progress (will be updated by websocket)
          if (progressBar) {
            progressBar.style.width = '0%';
          }
        }
      });
    } catch (error) {
      console.error('Failed to load job statuses:', error);
    }
  };

  const startJob = async button => {
    button.disabled = true;
    const jobId = button.dataset.jobId;
    const jobEl = document.getElementById(jobId);
    const progressContainer = jobEl.querySelector('.job-progress-container');
    const progressBar = jobEl.querySelector('.job-progress-bar');
    const description = jobEl.querySelector('.job-description');

    progressContainer.style.display = 'block';
    progressBar.style.width = '0%';
    description.textContent = 'Starting job...';

    await fetch('/api/admin/jobs/run', {
      method: 'POST',
      body: JSON.stringify({ job_id: jobId }),
    });
  };

  const initWebSocket = () => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(`${protocol}//${window.location.host}/ws/admin/progress`);

    ws.onopen = () => {
      console.log('WebSocket connected for admin page');
    };

    ws.onmessage = event => {
      const data = JSON.parse(event.data);
      const jobEl = document.getElementById(data.jobId);
      if (!jobEl) return;

      const progressContainer = jobEl.querySelector('.job-progress-container');
      const progressBar = jobEl.querySelector('.job-progress-bar');
      const description = jobEl.querySelector('.job-description');
      const button = jobEl.querySelector('.start-job-btn');

      // Ensure progress container is visible when receiving updates
      // Check both inline style and computed style to catch hidden containers
      if (progressContainer) {
        const isHidden =
          progressContainer.style.display === 'none' ||
          window.getComputedStyle(progressContainer).display === 'none';
        if (isHidden) {
          progressContainer.style.display = 'block';
        }
      }

      // Update progress bar if it exists
      if (progressBar) {
        progressBar.style.width = `${data.progress}%`;
      }

      // Update description if it exists
      if (description && data.message) {
        description.textContent = data.message;
      }

      if (data.done) {
        if (button) {
          button.disabled = false;
        }
        setTimeout(() => {
          if (progressContainer) {
            progressContainer.style.display = 'none';
          }
        }, 5000);
      } else {
        // Job is still running, ensure button is disabled
        if (button) {
          button.disabled = true;
        }
      }
    };

    ws.onclose = () => {
      console.log('WebSocket disconnected. Reconnecting in 5 seconds...');
      setTimeout(initWebSocket, 5000);
    };

    ws.onerror = err => {
      console.error('WebSocket error:', err);
    };
  };

  // Load job statuses first to restore running jobs
  await loadJobStatuses();

  initWebSocket();

  document.querySelectorAll('.start-job-btn').forEach(button => {
    button.addEventListener('click', e => {
      startJob(e.target);
    });
  });

  document.querySelectorAll('.href').forEach(button => {
    button.addEventListener('click', e => {
      window.location.href = e.target.dataset.endpoint;
    });
  });
});
