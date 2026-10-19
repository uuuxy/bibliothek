import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import 'fake-indexeddb/auto';

// „Theke leeren" (Abmelden, Frist ohne Bedienung) bringt die Theke in einen Zustand ohne
// Personendaten, auch während ein Scan unterwegs ist. Seine Antwort darf danach keinen Leser
// mehr laden, keine Rückfrage öffnen und keinen Ausweis merken: Sonst sähe der nächste Bediener
// das Konto des vorigen Lesers, und sein nächster Scan buchte auf ihn (docs/OFFEN.md 5.74).
// Was der Server gebucht hat, bleibt gebucht; ein Buch, dessen Versand scheitert, geht mit
// seinem Schnappschuss in die Warteschlange.

vi.mock('./snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));
vi.mock('../apiFetch.js', () => ({ apiFetch: vi.fn(), apiClient: { post: vi.fn() } }));
vi.mock('../audio.js', () => ({
	spieleErfolgston: vi.fn(),
	spieleFehlerton: vi.fn(),
	spieleErfolgspiep: vi.fn(),
	spieleFehlerpiep: vi.fn()
}));
vi.mock('./buchBarcodes.svelte.js', () => ({
	buchBarcodes: {
		istBuch: () => false,
		bereitstellen: vi.fn(),
		laden: vi.fn(),
		auffrischen: vi.fn(),
		anzahl: 0,
		geholtAm: 0
	}
}));

import { apiClient, apiFetch } from '../apiFetch.js';
import { snackbars } from './snackbars.svelte.js';
import { omniboxStore } from './omnibox.svelte.js';
import { thekeLeeren } from './thekeLeeren.js';
import { ladeWarteschlange, dequeueOfflineAction } from '../offlineWarteschlange.js';

const MIA = { id: 'leser-1', art: 'schueler', vorname: 'Mia', nachname: 'Muster', klasse: '07B' };
const BUCH = { titel: 'Natura 2' };

/** @param {Record<string, unknown>} data @returns {any} */
const antwort = (data) => ({ ok: true, status: 200, json: async () => data });

/** Die nächste Anfrage an die Buchungstür bleibt offen, bis der Test sie beantwortet. */
function halteAnfrage() {
	/** @type {{ beantworte: (a: any) => void, scheitere: (grund: Error) => void }} */
	const griff = { beantworte: () => {}, scheitere: () => {} };
	vi.mocked(apiClient.post).mockImplementationOnce(
		() =>
			new Promise((ja, nein) => {
				griff.beantworte = ja;
				griff.scheitere = nein;
			})
	);
	return griff;
}

/** Scannt und leert die Theke, solange die Anfrage unterwegs ist. @param {string} code */
function scanneUndLeere(code) {
	omniboxStore.eingabe = code;
	const laeuft = omniboxStore.fuehreAktionAus(null, null);
	thekeLeeren();
	return laeuft;
}

/** Die Nummern aller Anfragen an die Buchungstür. */
const gesendet = () =>
	vi.mocked(apiClient.post).mock.calls.map((aufruf) => /** @type {any} */ (aufruf[1]).query);
/** Der Rumpf der Anfrage mit dieser Nummer. @param {string} nummer @returns {any} */
const anfrage = (nummer) =>
	vi.mocked(apiClient.post).mock.calls.find((a) => /** @type {any} */ (a[1]).query === nummer)?.[1];

beforeEach(async () => {
	thekeLeeren();
	vi.clearAllMocks();
	for (const eintrag of await ladeWarteschlange()) await dequeueOfflineAction(eintrag.id);
});
afterEach(() => omniboxStore.stoppeZeitgeber());

describe('Eine Antwort, die erst nach „Theke leeren" ankommt', () => {
	it('lädt keinen Leser mehr', async () => {
		const ausweis = halteAnfrage();
		const laeuft = scanneUndLeere('S-7');
		ausweis.beantworte(
			antwort({ type: 'student', student: MIA, abholbereit: [{ titel: 'Natura 2' }] })
		);
		await laeuft;

		expect(omniboxStore.aktiverLeser, 'der nächste Bediener sähe Mias Konto').toBeNull();
		expect(omniboxStore.abholbereit).toEqual([]);
	});

	it('lädt bei einer Rückgabe nicht den Ausleiher und nennt keine Vormerkung', async () => {
		const buch = halteAnfrage();
		const laeuft = scanneUndLeere('B-1');
		buch.beantworte(
			antwort({
				type: 'rueckgabe',
				book: BUCH,
				student: MIA,
				has_vormerkung: true,
				vormerkung_user: 'Tom Test (07B)'
			})
		);
		await laeuft;

		expect(omniboxStore.aktiverLeser).toBeNull();
		expect(omniboxStore.vormerkungHinweis, 'der Hinweis nennt einen Namen').toBeNull();
		expect(snackbars.addToast, 'die Theke meldet den Scan nicht mehr').not.toHaveBeenCalled();
	});

	it('öffnet keine Rückfrage zur Sperre', async () => {
		omniboxStore.aktiverLeser = MIA;
		const buch = halteAnfrage();
		const laeuft = scanneUndLeere('B-1');
		buch.beantworte({
			ok: false,
			status: 403,
			headers: new Headers({ 'X-Sperre': 'leser' }),
			text: async () => JSON.stringify({ error: 'Mia Muster ist gesperrt.' })
		});
		await laeuft;

		expect(omniboxStore.sperrHinweis, 'der Sperrgrund gehört zum vorigen Leser').toBeNull();
		expect(omniboxStore.rueckmeldung.fehler).toBe('');
	});

	it('bucht einen Scan, der nach dem Leeren kam, hinter ihr ohne den Leser von vorher', async () => {
		const ausweis = halteAnfrage();
		vi.mocked(apiClient.post).mockResolvedValue(antwort({ type: 'info', message: 'Im Bestand.' }));
		const laeuft = scanneUndLeere('S-7');
		omniboxStore.eingabe = 'B-9';
		omniboxStore.fuehreAktionAus(null, null);
		ausweis.beantworte(antwort({ type: 'student', student: MIA }));
		await laeuft;

		expect(gesendet(), 'der neue Scan wartet und geht danach hinaus').toEqual(['S-7', 'B-9']);
		expect(anfrage('B-9').active_leser_id, 'Mia stand nie an dieser Theke').toBeUndefined();
		expect(omniboxStore.rueckmeldung.fehler, 'kein Scan als nicht gebucht genannt').toBe('');
	});

	it('merkt keinen Ausweis, dessen Versand scheitert', async () => {
		const ausweis = halteAnfrage();
		const laeuft = scanneUndLeere('S-7');
		ausweis.scheitere(new TypeError('Failed to fetch'));
		await laeuft;

		expect(omniboxStore.offlineAusweis, 'die nächsten Bücher gingen an diesen Ausweis').toBe('');
	});

	it('legt ein Buch, dessen Versand scheitert, mit dem Leser vom Scan ab', async () => {
		omniboxStore.aktiverLeser = MIA;
		const buch = halteAnfrage();
		const laeuft = scanneUndLeere('B-1');
		buch.scheitere(new TypeError('Failed to fetch'));
		await laeuft;

		const abgelegt = (await ladeWarteschlange()).map((e) => `${e.barcode} ${e.art} ${e.leser_id}`);
		expect(abgelegt).toEqual(['B-1 ausleihe leser-1']);
	});

	it('legt das Buch nach einem gemerkten Ausweis mit dem Ausweis ab, statt es ohne Leser zu senden', async () => {
		omniboxStore.offlineAusweis = 'S-7';
		const aufloesung = halteAnfrage();
		const laeuft = scanneUndLeere('B-1');
		aufloesung.beantworte(antwort({ type: 'student', student: MIA }));
		await laeuft;

		expect(omniboxStore.aktiverLeser).toBeNull();
		expect(gesendet(), 'ohne Leser wäre das Buch am Server eine Rückgabe').toEqual(['S-7']);
		const abgelegt = (await ladeWarteschlange()).map(
			(e) => `${e.barcode} ${e.art} ${e.ausweis_barcode}`
		);
		expect(abgelegt).toEqual(['B-1 ausleihe S-7']);
	});
});

describe('Eine Suche, die erst nach „Theke leeren" antwortet', () => {
	it('füllt die Trefferliste nicht wieder mit Namen', async () => {
		/** @type {(a: any) => void} */
		let antworte = () => {};
		vi.mocked(apiFetch).mockImplementationOnce(() => new Promise((ja) => (antworte = ja)));
		omniboxStore.eingabe = 'Mus';
		omniboxStore.suche.verarbeiteEingabe();
		// Die Entprellung (300 ms) läuft ab, die Suche ist unterwegs.
		await vi.waitFor(() => expect(apiFetch).toHaveBeenCalledTimes(1), { timeout: 1000 });

		thekeLeeren();
		antworte(antwort({ students: [MIA], books: [], students_total: 1 }));
		await new Promise((fertig) => setTimeout(fertig, 20));

		expect(omniboxStore.suche.trefferlisteOffen, 'der nächste Bediener sähe Mia').toBe(false);
		expect(omniboxStore.suche.suchTreffer.students).toEqual([]);
	});
});
