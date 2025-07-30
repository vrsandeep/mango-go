import { checkAuth } from './auth.js';

document.addEventListener('DOMContentLoaded', async () => {
    const currentUser = await checkAuth('admin');
    if (!currentUser) return;

    const tableBody = document.getElementById('users-table-body');
    const modal = document.getElementById('user-modal');
    const modalTitle = document.getElementById('modal-title');
    const userForm = document.getElementById('user-form');
    const userIdInput = document.getElementById('user-id');
    const usernameInput = document.getElementById('username-input');
    const passwordInput = document.getElementById('password-input');
    const roleSelect = document.getElementById('role-select');

    let allUsers = [];

    const renderUsers = () => {
        tableBody.innerHTML = '';
        allUsers.forEach(user => {
            const row = document.createElement('tr');
            const createdAt = new Date(user.created_at).toLocaleDateString();
            row.innerHTML = `
    <td>${user.username}</td>
    <td>${user.role}</td>
    <td>${createdAt}</td>
    <td class="actions-cell">
        <button class="edit-btn" data-id="${user.id}" title="Edit User"><i class="ph-bold ph-pencil-simple"></i></button>
        <button class="delete-btn" data-id="${user.id}" title="Delete User" ${currentUser.id === user.id ? 'disabled' : ''}><i class="ph-bold ph-trash"></i></button>
    </td>
    `;
            tableBody.appendChild(row);
        });
    };

    const loadUsers = async () => {
        try {
            const response = await fetch('/api/admin/users');
            allUsers = await response.json();
            renderUsers();
        } catch (e) {
            console.error("Failed to load users:", e);
            toast.error("Could not load users.");
        }
    };

    const openModal = (user = null) => {
        userForm.reset();
        if (user) {
            modalTitle.textContent = 'Edit User';
            userIdInput.value = user.id;
            usernameInput.value = user.username;
            roleSelect.value = user.role;
            passwordInput.placeholder = "Leave blank to keep unchanged";
            passwordInput.required = false;
        } else {
            modalTitle.textContent = 'Add New User';
            userIdInput.value = '';
            passwordInput.placeholder = "Password";
            passwordInput.required = true;
        }
        modal.style.display = 'flex';
    };

    const closeModal = () => {
        modal.style.display = 'none';
    };

    const handleFormSubmit = async (e) => {
        e.preventDefault();
        const id = userIdInput.value;
        const isEditing = id !== '';
        const url = isEditing ? `/api/admin/users/${id}` : '/api/admin/users';
        const method = isEditing ? 'PUT' : 'POST';

        const payload = {
            username: usernameInput.value,
            role: roleSelect.value,
        };
        if (passwordInput.value) {
            payload.password = passwordInput.value;
        }

        if (!isEditing && !payload.password) {
            toast.error("Password is required for new users.");
            return;
        }

        try {
            const response = await fetch(url, {
                method,
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
            if (response.ok) {
                closeModal();
                await loadUsers();
            } else {
                const error = await response.json();
                toast.error(error.error);
            }
        } catch (e) {
            toast.error("An unexpected error occurred.");
        }
    };

    const handleDelete = async (userId, username) => {
        if (!confirm(`Are you sure you want to delete user "${username}"? This cannot be undone.`)) {
            return;
        }
        try {
            const response = await fetch(`/api/admin/users/${userId}`, { method: 'DELETE' });
            if (response.ok) {
                await loadUsers();
            } else {
                const error = await response.json();
                toast.error(error.error);
            }
        } catch (e) {
            toast.error("An unexpected error occurred.");
        }
    };

    document.getElementById('add-user-btn').addEventListener('click', () => openModal());
    document.getElementById('modal-cancel-btn').addEventListener('click', closeModal);
    modal.addEventListener('click', (e) => {
        if (e.target === modal) closeModal();
    });
    userForm.addEventListener('submit', handleFormSubmit);

    tableBody.addEventListener('click', (e) => {
        const editBtn = e.target.closest('.edit-btn');
        if (editBtn) {
            const user = allUsers.find(u => u.id == editBtn.dataset.id);
            if (user) openModal(user);
        }

        const deleteBtn = e.target.closest('.delete-btn');
        if (deleteBtn) {
            const user = allUsers.find(u => u.id == deleteBtn.dataset.id);
            if (user) handleDelete(user.id, user.username);
        }
    });

    loadUsers();
});