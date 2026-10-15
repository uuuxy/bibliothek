import { snackbars } from '../../lib/stores/snackbars.svelte.js';

/** @type {{ searchQuery: string, bestandsAnsicht: 'mit' | 'ohne', selectedBook: any, activeBookId: string | null, adminAuthenticated: boolean, guestAuthenticated: boolean, triggerStudentScan: string, bookToEdit: any, requestAdminView: boolean }} */
export const appState = $state({
	searchQuery: '',
	// Sicht der Titelverwaltung: der Katalog ('mit') oder die Aufräumsicht ('ohne'
	// Exemplare). Das Portal kennt nur den Katalog.
	bestandsAnsicht: 'mit',
	selectedBook: null,
	activeBookId: null,
	adminAuthenticated: false,
	guestAuthenticated: false,
	triggerStudentScan: '',
	bookToEdit: null,
	requestAdminView: false
});

/**
 * Delegiert an das globale Toast-System der Haupt-App (SnackbarAnzeige in
 * App.svelte). Das frühere Single-Slot-toastState hatte keinen gemounteten
 * Renderer und konnte nur eine Meldung gleichzeitig halten.
 * @param {string} message
 * @param {import('../../lib/stores/snackbars.svelte.js').ToastTyp} [type='success']
 */
export function showToast(message, type = 'success') {
	snackbars.addToast(message, type);
}
