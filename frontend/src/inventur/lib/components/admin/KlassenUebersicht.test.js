import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';
import KlassenUebersicht from './KlassenUebersicht.svelte';
import { apiFetch } from '../../../../lib/apiFetch.js';
import { authStore } from '../../../../lib/stores/authStore.svelte.js';

vi.mock('../../../../lib/apiFetch.js', () => ({
	apiFetch: vi.fn(),
	registriereSitzungAbgelaufenHandler: vi.fn()
}));

// Diese Seite hat am 08.08.2026 den Klassen-Reiter aus dem Medienkatalog abgelöst.
// Der Reiter stand jedem mit view_books offen, diese Seite verlangte manage_users —
// faktisch Administrator. Wäre die Zusammenlegung ohne diese beiden Anpassungen
// gelaufen, hätten die Bibliotheks-Helfer die Klassensätze verloren:
//
//   1. Der Menüpunkt hängt jetzt an view_books  (menu.js)
//   2. Gelesen wird über /api/class-books       (view_books, nicht edit_books)
//
// Und weil damit Leute auf die Seite kommen, die NICHT pflegen dürfen, dürfen die
// Verwaltungsknöpfe nicht mehr bedingungslos dastehen — sie liefen ins 403.
//
// Kein E2E-Test: Im lokalen Stack trägt jede Rolle edit_books, ein Benutzer ohne das
// Recht müsste erst role_permissions umschreiben — also genau die Konfiguration, die
// ein E2E-Teardown schon einmal mitgenommen hat.

const GRUPPEN = [
	{ className: '09z1', books: [{ id: 'b1', title: 'Mathe 9', subject: 'Mathe' }] },
	{ className: '09z2', books: [{ id: 'b2', title: 'Deutsch 9', subject: 'Deutsch' }] }
];

/** @param {string[]} permissions */
function alsBenutzerMit(permissions) {
	authStore.currentUser = { id: 1, rolle: 'helfer', permissions };
}

describe('KlassenUebersicht', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(apiFetch).mockImplementation(
			/** @type {any} */ (async () => ({ ok: true, json: async () => ({ data: GRUPPEN }) }))
		);
	});

	it('liest über die view_books-Route, nicht über die admin-Route', async () => {
		alsBenutzerMit(['view_books']);
		const screen = render(KlassenUebersicht);

		await screen.findByText('09z1');

		const angefragt = vi.mocked(apiFetch).mock.calls.map((c) => String(c[0]));
		expect(angefragt.some((u) => u.startsWith('/api/class-books?'))).toBe(true);
		expect(angefragt.some((u) => u.includes('/api/admin/class-books'))).toBe(false);
	});

	it('zeigt ohne edit_books die Klassensätze, aber keine Verwaltungsknöpfe', async () => {
		alsBenutzerMit(['view_books']);
		const screen = render(KlassenUebersicht);

		// Die Liste selbst MUSS da sein — sonst prüft der Test nur eine kaputte Seite.
		await screen.findByText('09z1');
		await screen.findByText('09z2');

		expect(screen.queryByRole('button', { name: /Klasse hinzufügen/ })).toBeNull();
		expect(screen.queryByRole('button', { name: 'Klasse bearbeiten' })).toBeNull();
		expect(screen.queryByRole('button', { name: 'Klasse löschen' })).toBeNull();
	});

	it('zeigt die Verwaltungsknöpfe mit edit_books', async () => {
		alsBenutzerMit(['view_books', 'edit_books']);
		const screen = render(KlassenUebersicht);

		await screen.findByText('09z1');

		expect(screen.getByRole('button', { name: /Klasse hinzufügen/ })).toBeTruthy();
		expect(screen.getAllByRole('button', { name: 'Klasse bearbeiten' })).toHaveLength(2);
		expect(screen.getAllByRole('button', { name: 'Klasse löschen' })).toHaveLength(2);
	});
});
