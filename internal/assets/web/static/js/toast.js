// Toast notification system
class Toast {
    constructor() {
        this.createToastContainer();
    }

    createToastContainer() {
        // Create toast container if it doesn't exist
        if (!document.getElementById('toast-container')) {
            const container = document.createElement('div');
            container.id = 'toast-container';
            container.style.cssText = `
                position: fixed;
                top: 20px;
                right: 20px;
                z-index: 10000;
                display: flex;
                flex-direction: column;
                gap: 10px;
            `;
            document.body.appendChild(container);
        }
    }

    show(message, type = 'info') {
        const container = document.getElementById('toast-container');
        const toast = document.createElement('div');

        // Set toast styles
        toast.style.cssText = `
            background: var(--toast-bg, #333);
            color: var(--toast-color, white);
            padding: 12px 16px;
            border-radius: 6px;
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
            font-size: 14px;
            max-width: 300px;
            word-wrap: break-word;
            opacity: 0;
            transform: translateX(100%);
            transition: all 0.3s ease;
            border-left: 4px solid var(--toast-border, #666);
        `;

        // Set type-specific styles
        switch (type) {
            case 'success':
                toast.style.setProperty('--toast-bg', '#4caf50');
                toast.style.setProperty('--toast-color', 'white');
                toast.style.setProperty('--toast-border', '#45a049');
                break;
            case 'error':
                toast.style.setProperty('--toast-bg', '#f44336');
                toast.style.setProperty('--toast-color', 'white');
                toast.style.setProperty('--toast-border', '#d32f2f');
                break;
            case 'warning':
                toast.style.setProperty('--toast-bg', '#ff9800');
                toast.style.setProperty('--toast-color', 'white');
                toast.style.setProperty('--toast-border', '#f57c00');
                break;
            default:
                toast.style.setProperty('--toast-bg', '#2196f3');
                toast.style.setProperty('--toast-color', 'white');
                toast.style.setProperty('--toast-border', '#1976d2');
        }

        toast.textContent = message;
        container.appendChild(toast);

        // Animate in
        setTimeout(() => {
            toast.style.opacity = '1';
            toast.style.transform = 'translateX(0)';
        }, 10);

        // Auto remove after 3 seconds
        setTimeout(() => {
            toast.style.opacity = '0';
            toast.style.transform = 'translateX(100%)';
            setTimeout(() => {
                if (toast.parentNode) {
                    toast.parentNode.removeChild(toast);
                }
            }, 300);
        }, 3000);
    }

    success(message) {
        this.show(message, 'success');
    }

    error(message) {
        this.show(message, 'error');
    }

    warning(message) {
        this.show(message, 'warning');
    }

    info(message) {
        this.show(message, 'info');
    }
}

// Create global toast instance
window.toast = new Toast();