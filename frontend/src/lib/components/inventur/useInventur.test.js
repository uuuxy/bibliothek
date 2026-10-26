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
import {
	schliesseAb,
	ladeOffeneSessions,
	ladeAbgeschlosseneInventuren,
	scanne,
	deuteScanErgebnis
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
describe('Scan während der Anfrage', () => {
	const scan = vi.mocked(scanne);
	/** @param {string} barcode */
	const treffer = (barcode) =>
		/** @type {any} */ ({ ok: true, status: 200, data: { barcode_id: barcode, titel: barcode } });

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
		scan.mockImplementation(async (_sitzung, barcode) => treffer(barcode));
		const inventur = useInventur();

		const erster = inventur.verarbeiteScan('B-1');
		const zweiter = inventur.verarbeiteScan('B-2');
		const dritter = inventur.verarbeiteScan('B-3');
		expect(scan).toHaveBeenCalledTimes(1);

		ersterFertig(treffer('B-1'));
		await Promise.all([erster, zweiter, dritter]);

		expect(scan.mock.calls.map((aufruf) => aufruf[1])).toEqual(['B-1', 'B-2', 'B-3']);
		expect(inventur.stand.erfasst).toBe(3);
		expect(inventur.letzterScan.barcode).toBe('B-3');
		expect(inventur.scanLaeuft).toBe(false);
	});

	it('bucht die folgenden Scans auch, wenn einer scheitert', async () => {
		scan.mockRejectedValueOnce(new Error('Netz weg'));
		scan.mockImplementation(async (_sitzung, barcode) => treffer(barcode));
		const inventur = useInventur();

		await Promise.all([inventur.verarbeiteScan('B-1'), inventur.verarbeiteScan('B-2')]);

		expect(scan).toHaveBeenCalledTimes(2);
		expect(inventur.stand.erfasst).toBe(1);
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

		ersterFertig(treffer('B-1'));
		await Promise.all([laufend, abschlussFrage]);
		expect(frage).toHaveBeenCalledTimes(1);
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
		await ablauf.ladeFruehereInventuren();
		expect(ablauf.nichtGeladen).toEqual({
			laeufe: 'Die Datenbank antwortet nicht',
			frueher: 'Zeitüberschreitung'
		});

		vi.mocked(ladeOffeneSessions).mockResolvedValue([{ session_id: 's1' }]);
		vi.mocked(ladeAbgeschlosseneInventuren).mockResolvedValue([]);
		await ablauf.ladeOffeneLaeufe();
		await ablauf.ladeFruehereInventuren();
		expect(ablauf.nichtGeladen).toEqual({ laeufe: '', frueher: '' });
		expect(ablauf.offeneLaeufe).toEqual([{ session_id: 's1' }]);
	});
});
