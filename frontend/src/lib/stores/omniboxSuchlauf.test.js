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
// zu sehen glaubte, und waehleTreffer buchte ohne Rückfrage auf den falschen
// Schüler. Das Schwester-Muster im bestellStore hatte den Schutz längst.
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
		store.eingabe = 'Mül';
		store.verarbeiteEingabe();
		await vi.advanceTimersByTimeAsync(300);

		// Zweite Eingabe: neuer Abruf, Antwort kommt sofort.
		store.eingabe = 'Schmidt';
		store.verarbeiteEingabe();
		await vi.advanceTimersByTimeAsync(300);
		expect(/** @type {any} */ (store.suchTreffer.students[0]).id).toBe('Schmidt');

		// Jetzt trifft die ALTE Antwort ein — sie darf die Liste nicht mehr anfassen.
		alteAntwort(treffer('Müller'));
		await vi.advanceTimersByTimeAsync(0);
		expect(
			/** @type {any} */ (store.suchTreffer.students[0]).id,
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

		store.eingabe = 'Mül';
		store.verarbeiteEingabe();
		await vi.advanceTimersByTimeAsync(300);

		store.eingabe = '';
		store.verarbeiteEingabe();
		antwort(treffer('Müller'));
		await vi.advanceTimersByTimeAsync(0);

		expect(store.suchTreffer.students).toHaveLength(0);
		expect(store.trefferlisteOffen).toBe(false);
	});

	// Die zweite Tür zu derselben Gefahr (Sweep „verschluckte Fehlantwort", 06.09.2026):
	// Nicht die verspätete Antwort, sondern die AUSBLEIBENDE. Bis dahin stand bei einem
	// Fehlschlag die Trefferliste des vorigen Suchtextes weiter im offenen Dropdown —
	// und waehleTreffer bucht ohne Rückfrage auf den ersten Treffer.
	it('verwirft die Treffer, wenn die Suche fehlschlägt', async () => {
		const store = createOmniboxStore();
		vi.mocked(apiFetch).mockImplementation(async () => treffer('Müller'));
		store.eingabe = 'Mül';
		store.verarbeiteEingabe();
		await vi.advanceTimersByTimeAsync(300);
		expect(store.trefferlisteOffen).toBe(true);

		vi.mocked(apiFetch).mockImplementation(
			async () => /** @type {any} */ ({ ok: false, status: 500, json: async () => ({}) })
		);
		store.eingabe = 'Schmidt';
		store.verarbeiteEingabe();
		await vi.advanceTimersByTimeAsync(300);

		expect(store.suchTreffer.students, 'Müller steht noch unter dem Suchtext Schmidt').toEqual([]);
		expect(store.trefferlisteOffen, 'das Dropdown blieb mit fremden Treffern offen').toBe(false);
	});

	it('verwirft die Treffer auch bei einem Netzfehler', async () => {
		const store = createOmniboxStore();
		vi.mocked(apiFetch).mockImplementation(async () => treffer('Müller'));
		store.eingabe = 'Mül';
		store.verarbeiteEingabe();
		await vi.advanceTimersByTimeAsync(300);

		vi.mocked(apiFetch).mockImplementation(async () => {
			throw new Error('offline');
		});
		store.eingabe = 'Schmidt';
		store.verarbeiteEingabe();
		await vi.advanceTimersByTimeAsync(300);

		expect(store.suchTreffer.students).toEqual([]);
		expect(store.trefferlisteOffen).toBe(false);
	});
});
