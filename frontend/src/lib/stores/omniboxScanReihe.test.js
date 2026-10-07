import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import 'fake-indexeddb/auto';

// Ein Handscanner wartet nicht auf die Antwort. Ein Scan, der eintrifft, solange die vorige
// Buchung läuft, wird eingereiht und danach gebucht; verworfen bliebe er unbemerkt, in der
// Schnellrückgabe bliebe das Buch verliehen (docs/OFFEN.md 5.56). Dieselbe Nummer noch
// einmal fällt weg wie bisher: Gebucht gäbe sie das eben geliehene Buch gleich zurück.
//
// Weiter geht die Reihe nur nach einer glatten Buchung. Sonst wird, was wartet, nicht
// gebucht und im Banner genannt: Nach einem gescheiterten Ausweis gingen die Bücher an den
// Leser davor.

vi.mock('./toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));
vi.mock('../apiFetch.js', () => ({ apiFetch: vi.fn(), apiClient: { post: vi.fn() } }));
vi.mock('../audio.js', () => ({
	playSoundSuccess: vi.fn(),
	playSoundError: vi.fn(),
	playSuccessBeep: vi.fn(),
	playErrorBeep: vi.fn()
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

import { apiClient } from '../apiFetch.js';
import { playSoundError } from '../audio.js';
import { createOmniboxStore } from './omnibox.svelte.js';
import { uiStore } from './uiStore.svelte.js';
import { loadQueue, dequeueOfflineAction } from '../offlineQueue.js';

const MIA = { id: 'leser-1', art: 'schueler', vorname: 'Mia', nachname: 'Muster', klasse: '07B' };
const TOM = { id: 'leser-2', art: 'schueler', vorname: 'Tom', nachname: 'Test', klasse: '07B' };
const BUCH = { titel: 'Natura 2' };

/** @param {Record<string, unknown>} data @returns {any} */
const antwort = (data) => ({ ok: true, status: 200, json: async () => data });
/** @param {string} text @param {number} [status] @param {Record<string, string>} [kopf] @returns {any} */
const ablehnung = (text, status = 409, kopf = {}) => ({
	ok: false,
	status,
	headers: new Headers(kopf),
	text: async () => JSON.stringify({ error: text })
});
const AUSLEIHE = antwort({ type: 'ausleihe', book: BUCH });
const RUECKGABE = antwort({ type: 'rueckgabe', book: BUCH, student: MIA });

/** @type {ReturnType<typeof createOmniboxStore>[]} */
const stores = [];
function neueTheke() {
	const store = createOmniboxStore();
	stores.push(store);
	return store;
}
// Ein Zeitgeber, der die Datei überlebt, reißt den ganzen Lauf (omniboxZeitgeber.test.js).
afterEach(() => {
	while (stores.length) stores.pop()?.stoppeZeitgeber();
	vi.useRealTimers();
});
beforeEach(async () => {
	vi.clearAllMocks();
	for (const eintrag of await loadQueue()) await dequeueOfflineAction(eintrag.id);
});

/** Scannt, ohne auf die Buchung zu warten. @param {ReturnType<typeof createOmniboxStore>} store @param {string} code */
function scanne(store, code) {
	store.queryVal = code;
	return store.submitAction(null, null);
}

/**
 * Die nächste Buchung bleibt offen, bis der Test sie beantwortet.
 * @returns {{ beantworte: (antwort: any) => void, scheitere: (grund: Error) => void }}
 */
function haltendeBuchung() {
	/** @type {(antwort: any) => void} */
	let beantworte = () => {};
	/** @type {(grund: Error) => void} */
	let scheitere = () => {};
	vi.mocked(apiClient.post).mockImplementationOnce(
		() =>
			new Promise((ja, nein) => {
				beantworte = ja;
				scheitere = nein;
			})
	);
	return {
		beantworte: (a) => beantworte(a),
		scheitere: (g) => scheitere(g)
	};
}

/** Die Nummern aller Anfragen an die Buchungstür, in ihrer Reihenfolge. */
const gebuchteNummern = () =>
	vi.mocked(apiClient.post).mock.calls.map((aufruf) => /** @type {any} */ (aufruf[1]).query);
/** Der Rumpf der Anfrage mit dieser Nummer. @param {string} nummer @returns {any} */
const anfrage = (nummer) =>
	vi.mocked(apiClient.post).mock.calls.find((a) => /** @type {any} */ (a[1]).query === nummer)?.[1];

describe('Scan während der laufenden Buchung', () => {
	it('bucht ihn danach, mit dem Leser, den der Scan davor geladen hat', async () => {
		const theke = neueTheke();
		const ausweis = haltendeBuchung();
		vi.mocked(apiClient.post).mockResolvedValue(AUSLEIHE);

		const erster = scanne(theke, 'A-1');
		const zweiter = scanne(theke, 'B-1');
		const dritter = scanne(theke, 'B-2');
		expect(gebuchteNummern(), 'vor der Antwort geht nur der erste Scan hinaus').toEqual(['A-1']);
		expect(theke.queryVal, 'das Feld ist frei für den nächsten Scan').toBe('');

		ausweis.beantworte(antwort({ type: 'student', student: MIA }));
		await Promise.all([erster, zweiter, dritter]);

		expect(gebuchteNummern()).toEqual(['A-1', 'B-1', 'B-2']);
		expect(anfrage('B-1').active_leser_id, 'das Buch geht an den Leser des Ausweises').toBe(
			'leser-1'
		);
		expect(anfrage('B-2').active_leser_id).toBe('leser-1');
		expect(anfrage('B-1').idempotency_key).not.toBe(anfrage('B-2').idempotency_key);
		expect(theke.errorMessage).toBe('');
	});

	it('nimmt in der Schnellrückgabe jedes Buch des Stapels zurück', async () => {
		const theke = neueTheke();
		theke.schalteSchnellrueckgabe(true);
		const erstes = haltendeBuchung();
		vi.mocked(apiClient.post).mockResolvedValue(RUECKGABE);

		const laeuft = scanne(theke, 'B-1');
		scanne(theke, 'B-2');
		scanne(theke, 'B-3');
		erstes.beantworte(RUECKGABE);
		await laeuft;

		expect(gebuchteNummern()).toEqual(['B-1', 'B-2', 'B-3']);
		expect(anfrage('B-3').active_leser_id, 'mit Leser liehe der Server aus').toBeUndefined();
		expect(theke.activeStudent).toBeNull();
	});

	it('verwirft dieselbe Nummer, solange sie gebucht wird', async () => {
		const theke = neueTheke();
		theke.activeStudent = MIA;
		const buch = haltendeBuchung();

		const laeuft = scanne(theke, 'B-1');
		scanne(theke, 'B-1');
		scanne(theke, 'b-1');
		buch.beantworte(AUSLEIHE);
		await laeuft;

		expect(gebuchteNummern(), 'der Doppelscan gäbe das Buch gleich wieder zurück').toEqual(['B-1']);
		expect(theke.errorMessage).toBe('');
		expect(playSoundError).not.toHaveBeenCalled();
	});

	it('verwirft dieselbe Nummer auch, solange sie noch wartet', async () => {
		const theke = neueTheke();
		theke.activeStudent = MIA;
		const erstes = haltendeBuchung();
		vi.mocked(apiClient.post).mockResolvedValue(AUSLEIHE);

		const laeuft = scanne(theke, 'B-1');
		scanne(theke, 'B-2');
		scanne(theke, 'B-2');
		erstes.beantworte(AUSLEIHE);
		await laeuft;

		expect(gebuchteNummern()).toEqual(['B-1', 'B-2']);
	});

	it('bucht dieselbe Nummer wieder, sobald sie gebucht ist', async () => {
		const theke = neueTheke();
		theke.activeStudent = MIA;
		vi.mocked(apiClient.post).mockResolvedValue(AUSLEIHE);

		await scanne(theke, 'B-1');
		await scanne(theke, 'B-1');

		expect(gebuchteNummern(), 'der zweite Scan von Hand ist gewollt').toEqual(['B-1', 'B-1']);
	});

	it('lässt das Scanfeld bereit, solange gebucht wird', async () => {
		const theke = neueTheke();
		const buch = haltendeBuchung();

		const laeuft = scanne(theke, 'B-1');
		expect(theke.scanfeldBereit(), 'gesperrt ginge der nächste Scan verloren').toBe(true);
		buch.beantworte(RUECKGABE);
		await laeuft;
	});

	it('bleibt eine Reihe, auch wenn die Theke währenddessen neu geöffnet wird', async () => {
		const theke = neueTheke();
		theke.activeStudent = MIA;
		const erstes = haltendeBuchung();
		vi.mocked(apiClient.post).mockResolvedValue(AUSLEIHE);

		const laeuft = scanne(theke, 'B-1');
		// Der Wechsel zur Ausleihe gibt dem Feld den Fokus; die Buchung läuft weiter.
		uiStore.beimWechselZurTheke?.();
		scanne(theke, 'B-2');
		expect(gebuchteNummern(), 'zwei Buchungen zugleich').toEqual(['B-1']);

		erstes.beantworte(AUSLEIHE);
		await laeuft;
		expect(gebuchteNummern()).toEqual(['B-1', 'B-2']);
	});
});

describe('Die Reihe hält an, und was wartet, wird genannt', () => {
	/** Was das Banner über nicht gebuchte Scans sagt. @param {string[]} nummern */
	const genannt = (nummern) =>
		`Gescannt und NICHT gebucht: ${nummern.map((n) => `„${n}“`).join(', ')} — bitte erneut scannen.`;

	it('nach einem gescheiterten Scan: Das Buch geht nicht an den Leser davor', async () => {
		const theke = neueTheke();
		theke.activeStudent = TOM;
		const ausweis = haltendeBuchung();

		const laeuft = scanne(theke, 'A-9');
		scanne(theke, 'B-1');
		scanne(theke, 'B-2');
		ausweis.beantworte(ablehnung('Ausweis A-9 ist nicht registriert', 404));
		await laeuft;

		expect(gebuchteNummern(), 'ein Buch ist an Tom gegangen').toEqual(['A-9']);
		expect(theke.errorMessage).toBe(
			`Fehler: Ausweis A-9 ist nicht registriert · ${genannt(['B-1', 'B-2'])}`
		);
		expect(playSoundError, 'wer nicht hinsieht, hört es').toHaveBeenCalledTimes(1);
		expect(theke.screenFlash).toBe('error');
		expect(theke.flashBorder).toBe('red');
	});

	it('nach einer Sperre: Die Rückfrage entscheidet ein Mensch', async () => {
		const theke = neueTheke();
		theke.activeStudent = MIA;
		const buch = haltendeBuchung();

		const laeuft = scanne(theke, 'B-1');
		scanne(theke, 'B-2');
		// Der nächste Scan ist halb getippt, als die Rückfrage aufgeht.
		theke.queryVal = 'B-3';
		buch.beantworte(ablehnung('Mia hat überfällige Medien.', 403, { 'X-Sperre': 'uebergehbar' }));
		await laeuft;

		expect(theke.blockAlert?.query).toBe('B-1');
		expect(gebuchteNummern()).toEqual(['B-1']);
		expect(theke.errorMessage).toBe(genannt(['B-2']));
		expect(theke.queryVal, 'der Rest stünde vor dem übernächsten Scan').toBe('');
		expect(theke.scanfeldBereit(), 'die Rückfrage behält die Tastatur').toBe(false);
	});

	it('nach einer Vormerkung, auch wenn die Rückgabe gebucht ist', async () => {
		const theke = neueTheke();
		theke.schalteSchnellrueckgabe(true);
		const buch = haltendeBuchung();

		const laeuft = scanne(theke, 'B-1');
		scanne(theke, 'B-2');
		buch.beantworte(antwort({ type: 'rueckgabe', book: BUCH, student: MIA, has_vormerkung: true }));
		await laeuft;

		expect(theke.vormerkungAlert).not.toBeNull();
		expect(gebuchteNummern()).toEqual(['B-1']);
		expect(theke.errorMessage).toBe(genannt(['B-2']));
	});

	it('nach einer Fremdrückgabe: Ihre Hinweiszeile bleibt stehen', async () => {
		const theke = neueTheke();
		theke.activeStudent = MIA;
		const buch = haltendeBuchung();

		const laeuft = scanne(theke, 'B-1');
		scanne(theke, 'B-2');
		buch.beantworte(
			antwort({ type: 'rueckgabe', book: BUCH, fremdrueckgabe: true, vorbesitzer: TOM })
		);
		await laeuft;

		expect(theke.lastFremdrueckgabe, 'der wartende Scan nähme die Zeile weg').toEqual({
			vorbesitzerName: 'Tom Test'
		});
		expect(gebuchteNummern()).toEqual(['B-1']);
		expect(theke.errorMessage).toBe(genannt(['B-2']));
	});

	it('nach einem Hinweis auf eine andere Auflage', async () => {
		const theke = neueTheke();
		theke.activeStudent = MIA;
		const buch = haltendeBuchung();
		const hinweis = {
			klasse: '07B',
			auflage: '3. Aufl.',
			andere: [{ auflage: '4. Aufl.', kinder: 12 }]
		};

		const laeuft = scanne(theke, 'B-1');
		scanne(theke, 'B-2');
		buch.beantworte(antwort({ type: 'ausleihe', book: BUCH, auflagen_hinweis: hinweis }));
		await laeuft;

		expect(theke.lastAuflagenHinweis).toEqual(hinweis);
		expect(gebuchteNummern()).toEqual(['B-1']);
		expect(theke.errorMessage).toBe(genannt(['B-2']));
	});

	it('wenn der Leser von außen weggenommen wird (Escape, Akte schließen, Theke leeren)', async () => {
		const theke = neueTheke();
		theke.activeStudent = MIA;
		const buch = haltendeBuchung();

		const laeuft = scanne(theke, 'B-1');
		scanne(theke, 'B-2');
		theke.activeStudent = null;
		expect(theke.errorMessage, 'sofort, nicht erst nach der Antwort').toBe(genannt(['B-2']));

		buch.beantworte(AUSLEIHE);
		await laeuft;
		expect(gebuchteNummern(), 'das Buch wäre ohne Leser eine Rückgabe').toEqual(['B-1']);
		expect(anfrage('B-1').active_leser_id, 'der laufende Scan behält seine Absicht').toBe(
			'leser-1'
		);
		expect(theke.errorMessage).toBe(genannt(['B-2']));
	});

	it('wenn die Schnellrückgabe umgeschaltet wird', async () => {
		const theke = neueTheke();
		theke.schalteSchnellrueckgabe(true);
		const buch = haltendeBuchung();

		const laeuft = scanne(theke, 'B-1');
		scanne(theke, 'B-2');
		theke.schalteSchnellrueckgabe(false);
		buch.beantworte(RUECKGABE);
		await laeuft;

		expect(gebuchteNummern()).toEqual(['B-1']);
		expect(theke.errorMessage).toBe(genannt(['B-2']));
	});

	it('nennt die Nummern, bis der nächste Scan kommt; der Grund verschwindet wie jede Meldung', async () => {
		vi.useFakeTimers();
		const theke = neueTheke();
		theke.activeStudent = TOM;
		const ausweis = haltendeBuchung();

		const laeuft = scanne(theke, 'A-9');
		scanne(theke, 'B-1');
		ausweis.beantworte(ablehnung('Ausweis A-9 ist nicht registriert', 404));
		await laeuft;
		expect(theke.errorMessage).toContain('Fehler: Ausweis A-9 ist nicht registriert');

		await vi.advanceTimersByTimeAsync(6000);
		expect(theke.errorMessage, 'die Rückfrage davor kann länger offen sein').toBe(genannt(['B-1']));
		await vi.advanceTimersByTimeAsync(60_000);
		expect(theke.errorMessage).toBe(genannt(['B-1']));

		vi.mocked(apiClient.post).mockResolvedValue(AUSLEIHE);
		await scanne(theke, 'B-1');
		expect(theke.errorMessage).toBe('');
	});

	it('nach einem unerwarteten Abbruch: Die Theke bucht danach wieder', async () => {
		const theke = neueTheke();
		theke.activeStudent = MIA;
		const fehler = vi.spyOn(console, 'error').mockImplementation(() => {});
		vi.spyOn(crypto, 'randomUUID').mockImplementationOnce(() => {
			throw new Error('kein Zufall');
		});

		await scanne(theke, 'B-1');
		expect(theke.errorMessage).toBe('Fehler: kein Zufall');
		expect(gebuchteNummern()).toEqual([]);
		fehler.mockRestore();

		vi.mocked(apiClient.post).mockResolvedValue(AUSLEIHE);
		await scanne(theke, 'B-2');
		expect(gebuchteNummern()).toEqual(['B-2']);
	});
});

describe('Ohne Netz', () => {
	it('legt jedes Buch der Reihe ab, mit dem Ausweis, der davor gemerkt wurde', async () => {
		const theke = neueTheke();
		const ausweis = haltendeBuchung();
		vi.mocked(apiClient.post).mockRejectedValue(new TypeError('Failed to fetch'));

		const laeuft = scanne(theke, 'A-10001');
		scanne(theke, 'B-1');
		scanne(theke, 'B-2');
		ausweis.scheitere(new TypeError('Failed to fetch'));
		await laeuft;

		expect(theke.offlineAusweis).toBe('A-10001');
		const abgelegt = (await loadQueue()).map((e) => `${e.barcode} ${e.art} ${e.ausweis_barcode}`);
		expect(abgelegt.toSorted()).toEqual(['B-1 ausleihe A-10001', 'B-2 ausleihe A-10001']);
		expect(theke.errorMessage).toBe('');
	});
});
