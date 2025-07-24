import { checkAuth } from './auth.js';

document.addEventListener('DOMContentLoaded', async () => {
  const currentUser = await checkAuth("admin");
  if (!currentUser) return;

  const startJob = async (button) => {
    button.disabled = true;
    const jobId = button.dataset.jobId;
    const jobEl = document.getElementById(jobId);
    const progressContainer = jobEl.querySelector('.job-progress-container');
    const progressBar = jobEl.querySelector('.job-progress-bar');
    const description = jobEl.querySelector('.job-description');

    progressContainer.style.display = 'block';
    progressBar.style.width = '0%';
    description.textContent = 'Starting job...';

    await fetch('/api/admin/jobs/run', { method: 'POST', body: JSON.stringify({ job_id: jobId }) });
  };

  const initWebSocket = () => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(`${protocol}//${window.location.host}/ws/admin/progress`);

    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      const jobEl = document.getElementById(data.jobId);
      if (!jobEl) return;

      const progressBar = jobEl.querySelector('.job-progress-bar');
      const description = jobEl.querySelector('.job-description');
      const button = jobEl.querySelector('.start-job-btn');

      progressBar.style.width = `${data.progress}%`;
      description.textContent = data.message;

      if (data.done) {
        button.disabled = false;
        setTimeout(() => {
          jobEl.querySelector('.job-progress-container').style.display = 'none';
        }, 5000);
      }
    };

    ws.onclose = () => {
      console.log('WebSocket disconnected. Reconnecting in 5 seconds...');
      setTimeout(initWebSocket, 5000);
    };
  };

  initWebSocket();

  document.querySelectorAll('.start-job-btn').forEach(button => {
    button.addEventListener('click', (e) => {
      startJob(e.target);
    });
  });

  document.querySelectorAll('.href').forEach(button => {
    button.addEventListener('click', (e) => {
      window.location.href = e.target.dataset.endpoint;
    });
  });

});