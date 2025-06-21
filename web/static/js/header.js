document.addEventListener('DOMContentLoaded', () => {

    const menuToggleBtn = document.getElementById('menu-toggle-btn');
    const navLinks = document.getElementById('nav-links');
    const themeToggleBtn = document.getElementById('theme-toggle-btn');
    const logoutBtn = document.getElementById('logout-btn');

    // --- Mobile Menu Logic ---
    menuToggleBtn.addEventListener('click', () => navLinks.classList.toggle('active'));

    // --- Theme Logic ---
    const applyTheme = (theme) => {
        document.body.classList.toggle('light-theme', theme === 'light');
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

    // Init
    applyTheme(localStorage.getItem('theme'));
    loadVersion();
});