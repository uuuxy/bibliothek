import { toastStore } from '../../lib/stores/toastStore.svelte.js';
// src/lib/store.svelte.js

/** @type {{ searchQuery: string, selectedBook: any, activeBookId: string | null, adminAuthenticated: boolean, guestAuthenticated: boolean, triggerStudentScan: string, bookToEdit: any, requestAdminView: boolean }} */
export const appState = $state({
	searchQuery: '',
	selectedBook: null,
	activeBookId: null,
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
 * @param {import('../../lib/stores/toastStore.svelte.js').ToastTyp} [type='success']
 */
export function showToast(message, type = 'success') {
	toastStore.addToast(message, type);
}
