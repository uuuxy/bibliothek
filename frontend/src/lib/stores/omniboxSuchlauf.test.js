import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

vi.mock('../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

import { apiFetch } from '../apiFetch.js';
import { erzeugeOmniboxStore } from './omnibox.svelte.js';
import { snackbars } from './snackbars.svelte.js';
import { serverAntwort } from '../serverAntwort.js';

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
		const store = erzeugeOmniboxStore();
		/** @type {(r: Response) => void} */
		let alteAntwort = () => {};
		vi.mocked(apiFetch)
			.mockImplementationOnce(() => new Promise((aufloesen) => (alteAntwort = aufloesen)))
			.mockImplementation(async () => treffer('Schmidt'));

		// Erste Eingabe: Abruf startet und bleibt hängen.
		store.eingabe = 'Mül';
		store.suche.verarbeiteEingabe();
		await vi.advanceTimersByTimeAsync(300);

		// Zweite Eingabe: neuer Abruf, Antwort kommt sofort.
		store.eingabe = 'Schmidt';
		store.suche.verarbeiteEingabe();
		await vi.advanceTimersByTimeAsync(300);
		expect(/** @type {any} */ (store.suche.suchTreffer.students[0]).id).toBe('Schmidt');

		// Jetzt trifft die ALTE Antwort ein — sie darf die Liste nicht mehr anfassen.
		alteAntwort(treffer('Müller'));
		await vi.advanceTimersByTimeAsync(0);
		expect(
			/** @type {any} */ (store.suche.suchTreffer.students[0]).id,
			'überholte Antwort hat die Trefferliste überschrieben — der nächste Klick bucht den falschen Schüler'
		).toBe('Schmidt');
	});

	// Auch das Lesen des Rumpfs dauert: Die Antwort des alten Laufs kann schon da sein, während
	// ihr Rumpf erst nach dem neuen fertig ist.
	it('verwirft auch eine überholte Antwort, deren Rumpf zuletzt gelesen ist', async () => {
		const store = erzeugeOmniboxStore();
		/** @type {(daten: unknown) => void} */
		let alterRumpf = () => {};
		vi.mocked(apiFetch)
			.mockImplementationOnce(
				async () =>
					/** @type {any} */ ({ ok: true, json: () => new Promise((ja) => (alterRumpf = ja)) })
			)
			.mockImplementation(async () => treffer('Schmidt'));

		store.eingabe = 'Mül';
		store.suche.verarbeiteEingabe();
		await vi.advanceTimersByTimeAsync(300);

		store.eingabe = 'Schmidt';
		store.suche.verarbeiteEingabe();
		await vi.advanceTimersByTimeAsync(300);
		expect(/** @type {any} */ (store.suche.suchTreffer.students[0]).id).toBe('Schmidt');

		alterRumpf({ students: [{ id: 'Müller', vorname: 'Müller', nachname: 'Test' }], books: [] });
		await vi.advanceTimersByTimeAsync(0);
		expect(
			/** @type {any} */ (store.suche.suchTreffer.students[0]).id,
			'der Rumpf des überholten Laufs hat die Trefferliste überschrieben'
		).toBe('Schmidt');
	});

	it('füllt eine geleerte Liste nicht nachträglich wieder', async () => {
		const store = erzeugeOmniboxStore();
		/** @type {(r: Response) => void} */
		let antwort = () => {};
		vi.mocked(apiFetch).mockImplementationOnce(
			() => new Promise((aufloesen) => (antwort = aufloesen))
		);

		store.eingabe = 'Mül';
		store.suche.verarbeiteEingabe();
		await vi.advanceTimersByTimeAsync(300);

		store.eingabe = '';
		store.suche.verarbeiteEingabe();
		antwort(treffer('Müller'));
		await vi.advanceTimersByTimeAsync(0);

		expect(store.suche.suchTreffer.students).toHaveLength(0);
		expect(store.suche.trefferlisteOffen).toBe(false);
	});

	// Die zweite Tür zu derselben Gefahr (Sweep „verschluckte Fehlantwort", 06.09.2026):
	// Nicht die verspätete Antwort, sondern die AUSBLEIBENDE. Bis dahin stand bei einem
	// Fehlschlag die Trefferliste des vorigen Suchtextes weiter im offenen Dropdown —
	// und waehleTreffer bucht ohne Rückfrage auf den ersten Treffer.
	it('verwirft die Treffer, wenn die Suche fehlschlägt', async () => {
		const store = erzeugeOmniboxStore();
		vi.mocked(apiFetch).mockImplementation(async () => treffer('Müller'));
		store.eingabe = 'Mül';
		store.suche.verarbeiteEingabe();
		await vi.advanceTimersByTimeAsync(300);
		expect(store.suche.trefferlisteOffen).toBe(true);

		vi.mocked(apiFetch).mockImplementation(
			async () => /** @type {any} */ ({ ok: false, status: 500, json: async () => ({}) })
		);
		store.eingabe = 'Schmidt';
		store.suche.verarbeiteEingabe();
		await vi.advanceTimersByTimeAsync(300);

		expect(
			store.suche.suchTreffer.students,
			'Müller steht noch unter dem Suchtext Schmidt'
		).toEqual([]);
		expect(store.suche.trefferlisteOffen, 'das Dropdown blieb mit fremden Treffern offen').toBe(
			false
		);
	});

	it('verwirft die Treffer auch bei einem Netzfehler', async () => {
		const store = erzeugeOmniboxStore();
		vi.mocked(apiFetch).mockImplementation(async () => treffer('Müller'));
		store.eingabe = 'Mül';
		store.suche.verarbeiteEingabe();
		await vi.advanceTimersByTimeAsync(300);

		vi.mocked(apiFetch).mockImplementation(async () => {
			throw new Error('offline');
		});
		store.eingabe = 'Schmidt';
		store.suche.verarbeiteEingabe();
		await vi.advanceTimersByTimeAsync(300);

		expect(store.suche.suchTreffer.students).toEqual([]);
		expect(store.suche.trefferlisteOffen).toBe(false);
	});
});

// Lehnt der Server die Suche ab, nennt die Meldung seinen Grund, etwa ein fehlendes Recht. Auch
// das Lesen des Grundes dauert: Hat inzwischen ein neuer Suchlauf begonnen, meldet der alte nichts.
describe('Omnibox-Suche: Meldung bei Ablehnung', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		vi.restoreAllMocks();
	});
	afterEach(() => vi.useRealTimers());

	it('nennt den Grund, den der Server angibt', async () => {
		const store = erzeugeOmniboxStore();
		const meldung = vi.spyOn(snackbars, 'addToast');
		vi.mocked(apiFetch).mockImplementation(async () =>
			serverAntwort(403, { error: 'Keine Berechtigung für diese Aktion' })
		);

		store.eingabe = 'Mül';
		store.suche.verarbeiteEingabe();
		await vi.advanceTimersByTimeAsync(300);

		expect(meldung).toHaveBeenCalledWith('Keine Berechtigung für diese Aktion', 'error');
	});

	it('meldet die Ablehnung eines überholten Suchlaufs nicht', async () => {
		const store = erzeugeOmniboxStore();
		const meldung = vi.spyOn(snackbars, 'addToast');
		/** @type {(rumpf: string) => void} */
		let alterGrund = () => {};
		vi.mocked(apiFetch)
			.mockImplementationOnce(async () => ({
				...serverAntwort(500),
				text: () => new Promise((ja) => (alterGrund = ja))
			}))
			.mockImplementation(async () => treffer('Schmidt'));

		store.eingabe = 'Mül';
		store.suche.verarbeiteEingabe();
		await vi.advanceTimersByTimeAsync(300);

		store.eingabe = 'Schmidt';
		store.suche.verarbeiteEingabe();
		await vi.advanceTimersByTimeAsync(300);

		alterGrund(JSON.stringify({ error: 'Ein interner Serverfehler ist aufgetreten' }));
		await vi.advanceTimersByTimeAsync(0);
		expect(
			meldung,
			'die Ablehnung des Suchtexts „Mül" steht unter „Schmidt"'
		).not.toHaveBeenCalled();
		expect(/** @type {any} */ (store.suche.suchTreffer.students[0]).id).toBe('Schmidt');
	});
});
