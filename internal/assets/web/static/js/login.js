import { checkAuth } from './auth.js';

document.addEventListener('DOMContentLoaded', async () => {
  checkAuth().then(user => {
    if (user) {
      window.location.href = '/';
    }
  });

  const loginForm = document.getElementById('login-form');
  const errorMessage = document.getElementById('error-message');

  loginForm.addEventListener('submit', async e => {
    e.preventDefault();
    errorMessage.style.display = 'none';

    const username = document.getElementById('username').value;
    const password = document.getElementById('password').value;

    try {
      const response = await fetch('/api/users/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      });

      if (response.ok) {
        window.location.href = '/';
      } else {
        const errorData = await response.json();
        errorMessage.textContent =
          errorData.error || 'Login failed. Please check your credentials.';
        errorMessage.style.display = 'block';
      }
    } catch (err) {
      console.error('Login request failed:', err);
      errorMessage.textContent = 'An error occurred. Please try again later.';
      errorMessage.style.display = 'block';
    }
  });
});
