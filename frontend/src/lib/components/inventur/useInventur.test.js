import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../../stores/snackbars.svelte.js', () => ({
	snackbars: { addToast: vi.fn() }
}));
vi.mock('../../stores/bestaetigung.svelte.js', () => ({ bestaetigen: vi.fn() }));
vi.mock('./inventurApi.js');
vi.mock('../../apiFetch.js', () => ({
	apiFetch: vi.fn(async () => ({ ok: false, json: async () => ({}) }))
}));
vi.mock('../../utils/audio.js', () => ({ spieleErfolgston: vi.fn(), spieleFehlerton: vi.fn() }));

import { bestaetigen } from '../../stores/bestaetigung.svelte.js';
import { snackbars } from '../../stores/snackbars.svelte.js';
import {
	schliesseAb,
	ladeOffeneSessions,
	ladeAbgeschlosseneInventuren,
	scanne,
	deuteScanErgebnis,
	brichAb
} from './inventurApi.js';
import { useInventur, abschlussText } from './useInventur.svelte.js';
import { spieleErfolgston, spieleFehlerton } from '../../utils/audio.js';

const frage = vi.mocked(bestaetigen);
const abschluss = vi.mocked(schliesseAb);

// Der Abschluss bucht jedes nicht gescannte Exemplar als Verlust und sondert es aus. Davor
// steht die Rückfrage des Hauses; „nein" (auch Escape) lässt die Inventur weiterlaufen.
describe('Inventur abschließen', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		vi.mocked(ladeOffeneSessions).mockResolvedValue([]);
	});

	it('nennt in der Rückfrage, wie viele Bücher als verloren gebucht werden', () => {
		expect(abschlussText(0)).toContain('keines als verloren');
		expect(abschlussText(1)).toMatch(/^1 Buch .* ist nicht gescannt\. Es wird unwiderruflich/);
		expect(abschlussText(47)).toMatch(
			/^47 Bücher .* sind nicht gescannt\. Sie werden unwiderruflich/
		);
	});

	it('schließt bei „nein" nicht ab und gibt den Fokus zurück ins Scanfeld', async () => {
		frage.mockResolvedValue(false);
		const fokus = vi.fn();
		await useInventur().abschliessenNachRueckfrage(fokus);

		expect(frage).toHaveBeenCalledWith(
			expect.objectContaining({ titel: 'Inventur abschließen?', gefaehrlich: true })
		);
		expect(abschluss).not.toHaveBeenCalled();
		expect(fokus).toHaveBeenCalledTimes(1);
	});

	it('schließt bei „ja" ab', async () => {
		frage.mockResolvedValue(true);
		abschluss.mockResolvedValue(
			/** @type {any} */ ({ ok: true, data: { verloren_gemeldet: 0, fehlbestand: [] } })
		);
		await useInventur().abschliessenNachRueckfrage();

		expect(abschluss).toHaveBeenCalledTimes(1);
	});
});

// Ein Handscanner wartet nicht auf die Antwort. Ein Scan, der eintrifft, solange der vorige am
// Server ist, wird danach gebucht: Verworfen fiele das Buch beim Abschluss als Verlust auf,
// obwohl es im Regal steht.
/** Die Antwort des Servers auf einen gezählten Scan, mit dem Stand des Laufs: neun erwartete
 * Bücher, davon `fehlend` noch nicht gescannt.
 * @param {string} barcode @param {number} fehlend */
const treffer = (barcode, fehlend) =>
	/** @type {any} */ ({
		ok: true,
		status: 200,
		data: { barcode_id: barcode, titel: barcode, erwartet: 9, fehlend }
	});

describe('Scan während der Anfrage', () => {
	const scan = vi.mocked(scanne);

	beforeEach(async () => {
		vi.clearAllMocks();
		vi.mocked(ladeOffeneSessions).mockResolvedValue([]);
		const echt = /** @type {any} */ (await vi.importActual('./inventurApi.js'));
		vi.mocked(deuteScanErgebnis).mockImplementation(echt.deuteScanErgebnis);
	});

	it('bucht den zweiten Scan nach dem ersten, in der Reihenfolge der Scans', async () => {
		/** @type {(antwort: any) => void} */
		let ersterFertig = () => {};
		scan.mockImplementationOnce(() => new Promise((fertig) => (ersterFertig = fertig)));
		scan.mockImplementation(async (_sitzung, barcode) =>
			treffer(barcode, 9 - Number(barcode.slice(2)))
		);
		const inventur = useInventur();

		const erster = inventur.verarbeiteScan('B-1');
		const zweiter = inventur.verarbeiteScan('B-2');
		const dritter = inventur.verarbeiteScan('B-3');
		expect(scan).toHaveBeenCalledTimes(1);

		ersterFertig(treffer('B-1', 8));
		await Promise.all([erster, zweiter, dritter]);

		expect(scan.mock.calls.map((aufruf) => aufruf[1])).toEqual(['B-1', 'B-2', 'B-3']);
		expect(inventur.stand.fehlend).toBe(6);
		expect(inventur.letzterScan.barcode).toBe('B-3');
		expect(inventur.scanLaeuft).toBe(false);
	});

	it('bucht die folgenden Scans auch, wenn einer scheitert', async () => {
		scan.mockRejectedValueOnce(new Error('Netz weg'));
		scan.mockImplementation(async (_sitzung, barcode) => treffer(barcode, 8));
		const inventur = useInventur();

		await Promise.all([inventur.verarbeiteScan('B-1'), inventur.verarbeiteScan('B-2')]);

		expect(scan).toHaveBeenCalledTimes(2);
		expect(inventur.stand.fehlend).toBe(8);
		expect(inventur.letzterScan.barcode).toBe('B-2');
	});

	it('fragt vor dem Abschluss erst, wenn die eingereihten Scans gebucht sind', async () => {
		/** @type {(antwort: any) => void} */
		let ersterFertig = () => {};
		scan.mockImplementationOnce(() => new Promise((fertig) => (ersterFertig = fertig)));
		frage.mockResolvedValue(false);
		const inventur = useInventur();

		const laufend = inventur.verarbeiteScan('B-1');
		const abschlussFrage = inventur.abschliessenNachRueckfrage();
		await Promise.resolve();
		expect(frage, 'die Zahl der fehlenden Bücher stimmt erst nach dem Scan').not.toHaveBeenCalled();

		ersterFertig(treffer('B-1', 8));
		await Promise.all([laufend, abschlussFrage]);
		expect(frage).toHaveBeenCalledTimes(1);
	});
});

// Ein Buch, das zweimal gescannt wird (der Handscanner liest doppelt, oder jemand scannt zur
// Sicherheit noch einmal), zählt einmal: Der Server kennt jedes Exemplar je Lauf nur einmal.
// Mitgezählte Antworten zählten es doppelt, und die Rückfrage vor dem Abschluss nannte zu wenige
// Verluste.
describe('Zählung der Scans', () => {
	const scan = vi.mocked(scanne);

	beforeEach(async () => {
		vi.clearAllMocks();
		const echt = /** @type {any} */ (await vi.importActual('./inventurApi.js'));
		vi.mocked(deuteScanErgebnis).mockImplementation(echt.deuteScanErgebnis);
	});

	it('zeigt die Zahl des Servers: ein zweimal gescanntes Buch zählt einmal', async () => {
		scan.mockImplementation(async (_sitzung, barcode) => treffer(barcode, 8));
		const inventur = useInventur();
		await inventur.verarbeiteScan('B-1');
		await inventur.verarbeiteScan('B-1');

		expect(inventur.stand).toMatchObject({ erwartet: 9, fehlend: 8 });
	});

	it('übernimmt die Zahl des Servers auch, wenn ein anderer Scanner mitzählt', async () => {
		scan.mockResolvedValue(treffer('B-1', 2));
		const inventur = useInventur();
		await inventur.verarbeiteScan('B-1');

		expect(inventur.stand.fehlend).toBe(2);
	});

	// Ein verliehenes oder gesperrtes Buch im Regal wird erfasst, der Abschluss erwartet es aber
	// nicht. Zählte die Seite jede Erfassung, hieße es „2 / 2" und „keines als verloren", und der
	// Abschluss buchte trotzdem einen Verlust. Die Rückfrage nennt deshalb die Zahl des Servers.
	it('nennt in der Rückfrage die fehlenden Bücher, wie der Server sie zählt', async () => {
		scan.mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				status: 200,
				data: { barcode_id: 'B-V', titel: 'Verliehen', erwartet: 2, fehlend: 1 }
			})
		);
		frage.mockResolvedValue(false);
		const inventur = useInventur();
		inventur.setzeLaufFort({ session_id: 's1', erwartet: 2, fehlend: 2, label: 'Regal' });
		await inventur.verarbeiteScan('B-V');
		await inventur.abschliessenNachRueckfrage();

		expect(frage).toHaveBeenCalledWith(expect.objectContaining({ text: abschlussText(1) }));
	});
});

// Die Inventur gibt wie die Theke einen Ton: Wer ins Regal sieht, hört, ob das Buch gezählt ist.
describe('Ton beim Scan', () => {
	const scan = vi.mocked(scanne);

	beforeEach(async () => {
		vi.clearAllMocks();
		vi.mocked(ladeOffeneSessions).mockResolvedValue([]);
		const echt = /** @type {any} */ (await vi.importActual('./inventurApi.js'));
		vi.mocked(deuteScanErgebnis).mockImplementation(echt.deuteScanErgebnis);
	});

	it('gibt den Erfolgston für ein gezähltes Buch, auch mit Hinweis', async () => {
		scan.mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				status: 200,
				data: { barcode_id: 'B-1', titel: 'Buch', warnungen: ['Ist ausgeliehen'] }
			})
		);
		await useInventur().verarbeiteScan('B-1');

		expect(spieleErfolgston).toHaveBeenCalledTimes(1);
		expect(spieleFehlerton).not.toHaveBeenCalled();
	});

	it('gibt den Fehlerton, wenn das Buch nicht zum Bereich gehört', async () => {
		scan.mockResolvedValue(
			/** @type {any} */ ({
				ok: false,
				status: 409,
				data: { status: 'ausser_scope', barcode_id: 'B-1', titel: 'Buch' }
			})
		);
		await useInventur().verarbeiteScan('B-1');

		expect(spieleFehlerton).toHaveBeenCalledTimes(1);
		expect(spieleErfolgston).not.toHaveBeenCalled();
	});

	it('gibt den Fehlerton für ein unbekanntes Buch und ohne Netz', async () => {
		scan.mockResolvedValueOnce(/** @type {any} */ ({ ok: false, status: 404, error: 'Unbekannt' }));
		scan.mockRejectedValueOnce(new Error('Netz weg'));
		const inventur = useInventur();
		await Promise.all([inventur.verarbeiteScan('B-1'), inventur.verarbeiteScan('B-2')]);

		expect(spieleFehlerton).toHaveBeenCalledTimes(2);
		expect(spieleErfolgston).not.toHaveBeenCalled();
	});
});

// Kamen die laufenden Inventuren nicht, fehlte ihre Liste still, und darüber stand „Keine
// Inventur aktiv" mit dem Knopf für einen neuen Lauf. Kamen die früheren nicht, fehlten sie
// ebenso still.
describe('Inventur: eine Liste kommt nicht', () => {
	beforeEach(() => vi.clearAllMocks());

	it('merkt sich je Liste den Grund und vergisst ihn beim nächsten Erfolg', async () => {
		vi.mocked(ladeOffeneSessions).mockRejectedValue(new Error('Die Datenbank antwortet nicht'));
		vi.mocked(ladeAbgeschlosseneInventuren).mockRejectedValue(new Error('Zeitüberschreitung'));
		const ablauf = useInventur();
		await ablauf.ladeOffeneLaeufe();
		await ablauf.frueher.lade();
		expect(ablauf.nichtGeladen).toEqual({
			laeufe: 'Die Datenbank antwortet nicht',
			frueher: 'Zeitüberschreitung'
		});

		vi.mocked(ladeOffeneSessions).mockResolvedValue([{ session_id: 's1' }]);
		vi.mocked(ladeAbgeschlosseneInventuren).mockResolvedValue({
			laeufe: [],
			weitere: 0,
			naechste_seite: ''
		});
		await ablauf.ladeOffeneLaeufe();
		await ablauf.frueher.lade();
		expect(ablauf.nichtGeladen).toEqual({ laeufe: '', frueher: '' });
		expect(ablauf.offeneLaeufe).toEqual([{ session_id: 's1' }]);
	});
});

// „Verwerfen" überging die Antwort des Servers: Lehnte er ab, blieb die Inventur in der Liste
// stehen, und nichts sagte, warum. Gleich in Chromium, Firefox und WebKit (OFFEN.md 5.106).
describe('Inventur verwerfen', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		vi.mocked(ladeOffeneSessions).mockResolvedValue([]);
		vi.mocked(ladeAbgeschlosseneInventuren).mockResolvedValue({
			laeufe: [],
			weitere: 0,
			naechste_seite: ''
		});
	});

	it('nennt den Grund, wenn der Server ablehnt', async () => {
		vi.mocked(brichAb).mockResolvedValue(
			/** @type {any} */ ({ ok: false, status: 503, error: 'Die Datenbank antwortet nicht' })
		);
		const ablauf = useInventur();
		await ablauf.verwerfeLauf({ session_id: 's1' });

		expect(ablauf.fehler).toBe('Die Datenbank antwortet nicht');
	});

	it('meldet nichts, wenn der Server verworfen hat', async () => {
		vi.mocked(brichAb).mockResolvedValue(/** @type {any} */ ({ ok: true, status: 200 }));
		const ablauf = useInventur();
		await ablauf.verwerfeLauf({ session_id: 's1' });

		expect(ablauf.fehler).toBe('');
		expect(ladeOffeneSessions).toHaveBeenCalledTimes(1);
	});
});

// Kam auf den Abschluss keine Antwort, hatte der Server oft schon abgeschlossen. Die Seite blieb
// beim Scanfeld, ohne Meldung und ohne Fehlbestand; ein zweiter Versuch meldete nur „Kein
// laufender Inventurlauf mit dieser Kennung". Gleich in Chromium, Firefox und WebKit (OFFEN.md
// 5.106).
describe('Abschluss ohne Antwort', () => {
	const lauf = { session_id: 's1', erwartet: 3, fehlend: 2, label: 'Gesamtbestand' };
	const frueher = [{ session_id: 's1', label: 'Gesamtbestand', erfasst: 1, verluste: 2 }];

	beforeEach(() => {
		vi.clearAllMocks();
		frage.mockResolvedValue(true);
		vi.mocked(ladeOffeneSessions).mockResolvedValue([]);
		vi.mocked(ladeAbgeschlosseneInventuren).mockResolvedValue({
			laeufe: frueher,
			weitere: 0,
			naechste_seite: ''
		});
	});

	it.each([
		['ohne Antwort', { ok: false, status: 0, error: 'Keine Verbindung zum Server.' }],
		[
			'auf „kein laufender Lauf"',
			{ ok: false, status: 404, error: 'Kein laufender Inventurlauf mit dieser Kennung' }
		]
	])('zeigt %s den Stand des Servers statt des Scanfelds', async (_name, antwort) => {
		abschluss.mockResolvedValue(/** @type {any} */ (antwort));
		const ablauf = useInventur();
		ablauf.setzeLaufFort(lauf);
		await ablauf.abschliessenNachRueckfrage();

		expect(snackbars.addToast).toHaveBeenCalledWith(antwort.error, 'error');
		expect(ablauf.status).toBe('idle');
		expect(ladeOffeneSessions).toHaveBeenCalledTimes(1);
		expect(ablauf.frueher.laeufe).toEqual(frueher);
	});

	it('bleibt beim Scanfeld, wenn der Server ablehnt', async () => {
		abschluss.mockResolvedValue(
			/** @type {any} */ ({ ok: false, status: 503, error: 'Die Datenbank antwortet nicht' })
		);
		const ablauf = useInventur();
		ablauf.setzeLaufFort(lauf);
		await ablauf.abschliessenNachRueckfrage();

		expect(ablauf.status).toBe('active');
		expect(ladeOffeneSessions).not.toHaveBeenCalled();
	});
});
