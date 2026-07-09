import { apiFetch, apiClient } from "../../lib/apiFetch.js";
import { toastStore } from "../../lib/stores/toastStore.svelte.js";
// src/lib/store.svelte.js

/** @type {{ searchQuery: string, selectedBook: any, activeBookId: string | null, isSidebarOpen: boolean, adminAuthenticated: boolean, guestAuthenticated: boolean, triggerStudentScan: string, bookToEdit: any, requestAdminView: boolean }} */
export const appState = $state({
    searchQuery: '',
    selectedBook: null,
    activeBookId: null,
    isSidebarOpen: true,
    adminAuthenticated: false,
    guestAuthenticated: false,
    triggerStudentScan: '',
    bookToEdit: null,
    requestAdminView: false
});

/**
 * Delegiert an das globale Toast-System der Haupt-App (ToastContainer in
 * App.svelte). Das frühere Single-Slot-toastState hatte keinen gemounteten
 * Renderer und konnte nur eine Meldung gleichzeitig halten.
 * @param {string} message
 * @param {'success' | 'error' | 'info'} [type='success']
 */
export function showToast(message, type = 'success') {
    toastStore.addToast(message, type);
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
