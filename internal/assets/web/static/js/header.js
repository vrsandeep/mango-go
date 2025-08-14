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
        themeToggleBtn.innerHTML = theme === 'light' ? '<i class="ph-bold ph-moon"></i>' : '<i class="ph-bold ph-sun"></i>';
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
        btn.addEventListener('click', (e) => {
            e.preventDefault();
            setOpen(!open);
        });

    });

    // Init
    applyTheme(localStorage.getItem('theme'));
    loadVersion();
});