import { describe, it, expect, vi, beforeEach } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/svelte';
import Katalogseite from './KatalogSuche.svelte';
import { buecherLaden } from '$lib/startseiten_api.js';

// Die Frage nach einem Exemplar geht an den Server; ohne eigene Antwort trägt keine Nummer eines.
vi.mock('../../lib/apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn(async () => ({ ok: true, json: async () => ({ treffer: null }) }))
}));
vi.mock('$lib/startseiten_api.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	buecherLaden: vi.fn()
}));

const BUCH = { id: 'b1', title: 'Green Line 3', isbn: '978-3-06-013076-4', author: 'Cornelsen' };

// Scheitert das Laden des Katalogs, sagt die Seite es mit dem Grund des Servers. Ohne Liste
// antwortete sonst jede Suche „Keine Bücher gefunden", als hätte die Schule das Buch nicht.
describe('Medienkatalog: der Katalog lädt nicht', () => {
	beforeEach(() => {
		vi.mocked(buecherLaden).mockReset();
	});

	it('nennt den Grund statt „Keine Bücher gefunden"', async () => {
		vi.mocked(buecherLaden).mockRejectedValueOnce(
			new Error('Ein interner Serverfehler ist aufgetreten')
		);
		render(Katalogseite);
		const feld = await screen.findByLabelText(/Suchen nach Titel/);

		await fireEvent.input(feld, { target: { value: 'Green Line' } });

		expect(await screen.findByText('Katalog nicht geladen')).toBeTruthy();
		expect(screen.getByText('Ein interner Serverfehler ist aufgetreten')).toBeTruthy();
		expect(screen.queryByText('Keine Bücher gefunden')).toBeNull();
	});

	it('lädt auf „Erneut versuchen" und zeigt dann den Katalog', async () => {
		vi.mocked(buecherLaden)
			.mockRejectedValueOnce(new Error('Ein interner Serverfehler ist aufgetreten'))
			.mockResolvedValueOnce([BUCH]);
		render(Katalogseite);
		await screen.findByText('Katalog nicht geladen');

		await fireEvent.click(screen.getByRole('button', { name: 'Erneut versuchen' }));

		expect(await screen.findByText('Green Line 3')).toBeTruthy();
		expect(screen.queryByText('Katalog nicht geladen')).toBeNull();
	});
});
