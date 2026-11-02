import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import Katalogseite from './KatalogSuche.svelte';
import { buecherLaden } from '$lib/startseiten_api.js';
import { appState } from '$lib/store.svelte.js';

vi.mock('../../lib/apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn(async () => ({ ok: true, json: async () => ({ treffer: null }) }))
}));
vi.mock('$lib/startseiten_api.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	buecherLaden: vi.fn()
}));

const BUCH = { id: 'b1', title: 'Green Line 3', isbn: '978-3-06-013076-4', author: 'Cornelsen' };

// Solange ein Titel offen ist, steht er in appState.selectedBook. Darauf baut das Speichern in der
// Maske, das die offene Ansicht mit dem gespeicherten Stand nachzieht (AdminBuchAktionen).
describe('Medienkatalog: der offene Titel', () => {
	beforeEach(() => {
		vi.mocked(buecherLaden).mockReset();
		vi.mocked(buecherLaden).mockResolvedValue([BUCH]);
		appState.selectedBook = null;
	});

	it('bleibt im gemeinsamen Zustand, solange er offen ist', async () => {
		render(Katalogseite);
		await screen.findByLabelText(/Suchen nach Titel/);

		appState.selectedBook = BUCH;
		await tick();
		await tick();

		expect(appState.selectedBook).toEqual(BUCH);
	});
});
