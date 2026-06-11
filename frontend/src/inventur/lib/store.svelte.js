import { apiFetch } from '../../lib/apiFetch.js';
// src/lib/store.svelte.js

/** @type {{ searchQuery: string, selectedBook: any, activeBookId: string | null, isSidebarOpen: boolean, adminAuthenticated: boolean, guestAuthenticated: boolean, pendingPrintCopies: any[] | null, triggerStudentScan: string, bookToEdit: any, requestAdminView: boolean }} */
export const appState = $state({
    searchQuery: '',
    selectedBook: null,
    activeBookId: null,
    isSidebarOpen: true,
    adminAuthenticated: false,
    guestAuthenticated: false,
    pendingPrintCopies: null,
    triggerStudentScan: '',
    bookToEdit: null,
    requestAdminView: false
});

export const toastState = $state({
    visible: false,
    message: '',
    type: 'success' // 'success' oder 'error'
});

/** @type {ReturnType<typeof setTimeout> | null} */
let toastTimeout = null;
/**
 * @param {string} message
 * @param {string} type
 */
export function showToast(message, type = 'success') {
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
        await apiFetch('/api/auth/logout', {
            method: 'POST'
        });
    } catch {
        // UI-State wurde bereits zurückgesetzt
    }
}
