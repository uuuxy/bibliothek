// src/lib/store.svelte.js
import { csrfHeader } from './csrf.js';

export const appState = $state({
    searchQuery: '',
    selectedBook: null,
    isSidebarOpen: true,
    adminAuthenticated: false,
    guestAuthenticated: false
});

export const toastState = $state({
    visible: false,
    message: '',
    type: 'success' // 'success' oder 'error'
});

let toastTimeout;
export function showToast(message, type = 'success') {
    console.log(`[Toast] Showing ${type}: ${message}`);
    toastState.message = message;
    toastState.type = type;
    toastState.visible = true;

    if (toastTimeout) clearTimeout(toastTimeout);

    toastTimeout = setTimeout(() => {
        toastState.visible = false;
    }, 3000);
}

export async function logout() {
    appState.adminAuthenticated = false;
    appState.guestAuthenticated = false;

    try {
        await fetch('/api/logout', {
            method: 'POST',
            credentials: 'include',
            headers: {
                ...csrfHeader(),
            },
        });
    } catch {
        // UI-State wurde bereits zurückgesetzt
    }
}
