import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';
import TitelVerwaltung from './TitelVerwaltung.svelte';
import { holeBuecherListe } from '$lib/admin_api.js';
import { appState } from '$lib/store.svelte.js';

vi.mock('$lib/admin_api.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	holeBuecherListe: vi.fn()
}));

const TITEL = { id: 't1', title: 'Mathe 9', author: 'Autor', isbn: '', verfuegbar: 1, gesamt: 1 };

// Scheitert das Laden der Liste, nennt die Seite den Grund. Eine leere Tabelle sagte sonst
// „Keine Bücher gefunden", und weil der Fehler die Anmeldung zurücknahm, lud danach auch die
// Suche nicht mehr.
describe('TitelVerwaltung: die Liste lädt nicht', () => {
	beforeEach(() => {
		vi.mocked(holeBuecherListe).mockReset();
		appState.adminAuthenticated = true;
		appState.searchQuery = '';
	});

	it('nennt den Grund und lädt auf „Erneut versuchen"', async () => {
		vi.mocked(holeBuecherListe)
			.mockRejectedValueOnce(new Error('Ein interner Serverfehler ist aufgetreten'))
			.mockResolvedValueOnce([TITEL]);
		const screen = render(TitelVerwaltung);

		expect(await screen.findByText('Titel nicht geladen')).toBeTruthy();
		expect(screen.getByText('Ein interner Serverfehler ist aufgetreten')).toBeTruthy();
		expect(screen.queryByText('Keine Bücher gefunden.')).toBeNull();

		await fireEvent.click(screen.getByRole('button', { name: 'Erneut versuchen' }));

		expect(await screen.findByText('Mathe 9')).toBeTruthy();
		expect(screen.queryByText('Titel nicht geladen')).toBeNull();
	});

	it('lädt nach dem Fehler auf eine neue Suche hin', async () => {
		vi.mocked(holeBuecherListe)
			.mockRejectedValueOnce(new Error('Ein interner Serverfehler ist aufgetreten'))
			.mockResolvedValueOnce([TITEL]);
		const screen = render(TitelVerwaltung);
		await screen.findByText('Titel nicht geladen');

		appState.searchQuery = 'Mathe';

		expect(await screen.findByText('Mathe 9')).toBeTruthy();
		await waitFor(() => expect(holeBuecherListe).toHaveBeenCalledTimes(2));
	});
});
