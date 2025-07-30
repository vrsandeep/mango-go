/**
 * Checks the user's authentication status by calling the /api/users/me endpoint.
 * This function should be called on every page that requires authentication.
 * * @param {string | null} requiredRole - Optional. If provided (e.g., 'admin'), the function will
 * check if the user has the specified role and redirect if they do not.
 * @returns {Promise<object | null>} A promise that resolves with the user object if authenticated
 * and authorized, or null otherwise.
 */
async function checkAuth(requiredRole = null) {
    // Define public pages that don't require an auth check.
    const publicPages = ['/login'];
    const currentPath = window.location.pathname;

    // If we are on a public page, don't perform the check.
    if (publicPages.some(page => currentPath.endsWith(page))) {
        return null;
    }

    try {
        const response = await fetch('/api/users/me', {
            headers: { 'Accept': 'application/json' }
        });

        // If the response is not OK (e.g., 401 Unauthorized), redirect to login.
        if (!response.ok) {
            window.location.href = '/login';
            return null;
        }

        const user = await response.json();

        // If a specific role is required, check it.
        if (requiredRole && user.role !== requiredRole) {
            if (window.toast) {
                toast.error('Access Denied: You do not have permission to view this page.');
            } else {
                alert('Access Denied: You do not have permission to view this page.');
            }
            window.location.href = '/'; // Redirect to a safe default page
            return null;
        }

        // --- Update UI based on authenticated state ---

        // Display the username
        const usernameDisplay = document.getElementById('username-display');
        if (usernameDisplay) {
            usernameDisplay.textContent = user.username;
        }

        // Show elements that are only for authenticated users
        document.querySelectorAll('.auth-only').forEach(el => el.style.display = 'block');

        // Show elements that are only for admin users
        if (user.role === 'admin') {
            document.querySelectorAll('.admin-only').forEach(el => el.style.display = 'block');
        }

        // Make the logout button functional
        const logoutBtn = document.getElementById('logout-btn');
        if (logoutBtn) {
            logoutBtn.addEventListener('click', async () => {
                await fetch('/api/users/logout', { method: 'POST' });
                window.location.href = '/login';
            });
        }

        return user;

    } catch (e) {
        console.error('Authentication check failed:', e);
        // In case of a network error, redirect to login as a failsafe.
        window.location.href = '/login';
        return null;
    }
}

export { checkAuth };
