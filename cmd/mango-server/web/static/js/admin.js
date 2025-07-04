document.addEventListener('DOMContentLoaded', async () => {
  const currentUser = await checkAuth("admin");
  if (!currentUser) return;

  const startJob = async (endpoint, button) => {
    button.disabled = true;
    const jobEl = button.closest('.job-item');
    const progressContainer = jobEl.querySelector('.job-progress-container');
    const progressBar = jobEl.querySelector('.job-progress-bar');
    const description = jobEl.querySelector('.job-description');

    progressContainer.style.display = 'block';
    progressBar.style.width = '0%';
    description.textContent = 'Starting job...';

    await fetch(endpoint, { method: 'POST' });
  };

  const initWebSocket = () => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(`${protocol}//${window.location.host}/ws/admin/progress`);

    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      const jobElId = `job-${data.job_name.toLowerCase().replace(/\s+/g, '-')}`;
      const jobEl = document.getElementById(jobElId);
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
      startJob(e.target.dataset.endpoint, e.target);
    });
  });

  document.querySelectorAll('.href').forEach(button => {
    button.addEventListener('click', (e) => {
      window.location.href = e.target.dataset.endpoint;
    });
  });

});