import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

vi.mock('../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

import { apiFetch } from '../apiFetch.js';
import { createOmniboxStore } from './omnibox.svelte.js';

/**
 * Antwort mit genau einem Schüler. Als Response getypt: apiFetch verspricht eine
 * echte Response, und svelte-check prüft die Testdateien mit (checkJs).
 * @param {string} name
 * @returns {Response}
 */
function treffer(name) {
	return /** @type {any} */ ({
		ok: true,
		json: async () => ({ students: [{ id: name, vorname: name, nachname: 'Test' }], books: [] })
	});
}

// Eine verspätete Antwort darf die Trefferliste nicht übernehmen.
//
// Der Entprell-Timer verwirft nur NOCH NICHT gestartete Abrufe; zwei laufende
// beantwortet der Server in beliebiger Reihenfolge. Bis zum 31.08.2026 schrieb die
// zuletzt EINTREFFENDE Antwort die Liste — der Bediener klickte auf die Zeile, die er
// zu sehen glaubte, und selectDropdownItem buchte ohne Rückfrage auf den falschen
// Schüler. Das Schwester-Muster im orderStore hatte den Schutz längst.
describe('Omnibox-Suchlauf', () => {
	beforeEach(() => vi.useFakeTimers());
	afterEach(() => vi.useRealTimers());

	it('verwirft die Antwort eines überholten Suchlaufs', async () => {
		const store = createOmniboxStore();
		/** @type {(r: Response) => void} */
		let alteAntwort = () => {};
		vi.mocked(apiFetch)
			.mockImplementationOnce(() => new Promise((aufloesen) => (alteAntwort = aufloesen)))
			.mockImplementation(async () => treffer('Schmidt'));

		// Erste Eingabe: Abruf startet und bleibt hängen.
		store.queryVal = 'Mül';
		store.handleInput();
		await vi.advanceTimersByTimeAsync(300);

		// Zweite Eingabe: neuer Abruf, Antwort kommt sofort.
		store.queryVal = 'Schmidt';
		store.handleInput();
		await vi.advanceTimersByTimeAsync(300);
		expect(/** @type {any} */ (store.unifiedSearchResults.students[0]).id).toBe('Schmidt');

		// Jetzt trifft die ALTE Antwort ein — sie darf die Liste nicht mehr anfassen.
		alteAntwort(treffer('Müller'));
		await vi.advanceTimersByTimeAsync(0);
		expect(
			/** @type {any} */ (store.unifiedSearchResults.students[0]).id,
			'überholte Antwort hat die Trefferliste überschrieben — der nächste Klick bucht den falschen Schüler'
		).toBe('Schmidt');
	});

	it('füllt eine geleerte Liste nicht nachträglich wieder', async () => {
		const store = createOmniboxStore();
		/** @type {(r: Response) => void} */
		let antwort = () => {};
		vi.mocked(apiFetch).mockImplementationOnce(
			() => new Promise((aufloesen) => (antwort = aufloesen))
		);

		store.queryVal = 'Mül';
		store.handleInput();
		await vi.advanceTimersByTimeAsync(300);

		store.queryVal = '';
		store.handleInput();
		antwort(treffer('Müller'));
		await vi.advanceTimersByTimeAsync(0);

		expect(store.unifiedSearchResults.students).toHaveLength(0);
		expect(store.isDropdownOpen).toBe(false);
	});
});
