import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import KlassenUebersicht from './KlassenUebersicht.svelte';
import { apiFetch } from '../../../../lib/apiFetch.js';
import { authStore } from '../../../../lib/stores/authStore.svelte.js';

vi.mock('../../../../lib/apiFetch.js', () => ({
	apiFetch: vi.fn(),
	registriereSitzungAbgelaufenHandler: vi.fn()
}));

// Die Seite steht jedem mit view_books offen, auch den Bibliotheks-Helfern:
//
//   1. Der Menüpunkt hängt an view_books        (menu.js)
//   2. Gelesen wird über /api/class-books       (view_books, nicht edit_books)
//
// Wer nicht pflegen darf, sieht deshalb keine Verwaltungsknöpfe — sie liefen ins 403.
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

		// Die Liste selbst muss da sein — sonst prüft der Test nur eine kaputte Seite.
		await screen.findByText('09z1');
		await screen.findByText('09z2');

		expect(screen.queryByRole('button', { name: /Klasse hinzufügen/ })).toBeNull();
		expect(screen.queryByRole('button', { name: 'Klasse bearbeiten' })).toBeNull();
		expect(screen.queryByRole('button', { name: 'Buchliste löschen' })).toBeNull();
	});

	it('zeigt die Verwaltungsknöpfe mit edit_books', async () => {
		alsBenutzerMit(['view_books', 'edit_books']);
		const screen = render(KlassenUebersicht);

		await screen.findByText('09z1');

		expect(screen.getByRole('button', { name: /Klasse hinzufügen/ })).toBeTruthy();
		expect(screen.getAllByRole('button', { name: 'Klasse bearbeiten' })).toHaveLength(2);
		expect(screen.getAllByRole('button', { name: 'Buchliste löschen' })).toHaveLength(2);
	});

	// Der Ladefehler bietet einen zweiten Versuch an, und ein gelungener Versuch räumt ihn ab.
	it('zeigt nach „Erneut versuchen" die Liste statt der Fehlermeldung', async () => {
		alsBenutzerMit(['view_books']);
		vi.mocked(apiFetch).mockImplementationOnce(
			/** @type {any} */ (async () => ({ ok: false, json: async () => ({}) }))
		);
		const screen = render(KlassenUebersicht);

		await screen.findByText('Klassensätze nicht geladen');
		expect(screen.queryByText('09z1')).toBeNull();

		await fireEvent.click(screen.getByRole('button', { name: 'Erneut versuchen' }));

		await screen.findByText('09z1');
		expect(screen.queryByText('Klassensätze nicht geladen')).toBeNull();
	});
});
