import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));
vi.mock('../../stores/snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));

import { apiFetch } from '../../apiFetch.js';
import { snackbars } from '../../stores/snackbars.svelte.js';
import { druckeBogen } from './etikettenDruck.js';
import { PDF_BLOCKIERT } from '../../utils/pdfImFenster.js';
import { serverAntwort } from '../../serverAntwort.js';

const BOGEN = { formatId: 'zweckform_l4760', startFeld: 1, alsQR: false };
const ETIKETTEN = [
	{ barcode_id: 'B-10001', titel: 'Natur und Technik', autor: '' },
	{ barcode_id: 'B-10002', titel: 'Natur und Technik', autor: '' }
];

/** @param {number} status @param {any} [koerper] @returns {any} */
const antwort = (status, koerper = {}) => ({
	ok: status < 400,
	status,
	json: async () => koerper,
	blob: async () => new Blob(['%PDF'])
});
/** Die Aufrufe einer Adresse mit ihrem Rumpf. @param {string} adresse */
const rumpfe = (adresse) =>
	vi
		.mocked(apiFetch)
		.mock.calls.filter(([url]) => url === adresse)
		.map(([, optionen]) => JSON.parse(String(optionen?.body)));
/** @param {any} rumpf */
const nummernIm = (rumpf) => rumpf.items.map((/** @type {any} */ i) => i.BarcodeID);

/** @type {{ closed: boolean, close: any, location: { replace: any } } | null} */
let fenster;
beforeEach(() => {
	vi.clearAllMocks();
	fenster = { closed: false, close: vi.fn(), location: { replace: vi.fn() } };
	vi.stubGlobal(
		'open',
		vi.fn(() => fenster)
	);
	URL.createObjectURL = vi.fn(() => 'blob:probe');
});
afterEach(() => {
	vi.unstubAllGlobals();
});

describe('druckeBogen', () => {
	it('druckt den Bogen und vermerkt die Etiketten als gedruckt', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(antwort(200)).mockResolvedValueOnce(antwort(200));

		await druckeBogen(BOGEN, ETIKETTEN);

		expect(rumpfe('/api/print/labels').map(nummernIm)).toEqual([['B-10001', 'B-10002']]);
		expect(rumpfe('/api/exemplare/etiketten-gedruckt')[0].barcode_ids).toEqual([
			'B-10001',
			'B-10002'
		]);
		expect(fenster?.location.replace).toHaveBeenCalledWith('blob:probe');
	});

	it('meldet einen gescheiterten Druck, schließt den Tab und vermerkt nichts', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(serverAntwort(502));

		await druckeBogen(BOGEN, ETIKETTEN);

		expect(snackbars.addToast).toHaveBeenCalledWith('Fehler beim Erstellen des PDFs', 'error');
		expect(rumpfe('/api/exemplare/etiketten-gedruckt')).toHaveLength(0);
		expect(fenster?.close).toHaveBeenCalledOnce();
	});

	it('nennt beim gescheiterten Druck den Grund, den der Server angibt', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(
			serverAntwort(403, { error: 'Keine Berechtigung für diese Aktion' })
		);

		await druckeBogen(BOGEN, ETIKETTEN);

		expect(snackbars.addToast).toHaveBeenCalledWith('Keine Berechtigung für diese Aktion', 'error');
		expect(rumpfe('/api/exemplare/etiketten-gedruckt')).toHaveLength(0);
	});

	// Scheiterte der Vermerk nach dem Druck, stand das nur in der Konsole: Die Etiketten blieben
	// unter „Fehlende Etiketten" und wurden womöglich ein zweites Mal gedruckt (OFFEN.md 5.106).
	it.each([
		[
			'bei einer Ablehnung',
			async () => serverAntwort(503, { error: 'Die Datenbank antwortet nicht' })
		],
		['ohne Antwort', () => Promise.reject(new Error('Keine Verbindung zum Server.'))]
	])('warnt %s des Vermerks, dass die Etiketten als fehlend gelten', async (_, vermerk) => {
		vi.mocked(apiFetch)
			.mockResolvedValueOnce(antwort(200))
			.mockImplementationOnce(/** @type {any} */ (vermerk));

		await druckeBogen(BOGEN, ETIKETTEN);

		expect(fenster?.location.replace).toHaveBeenCalledWith('blob:probe');
		expect(snackbars.addToast).toHaveBeenCalledWith(
			expect.stringMatching(
				/^Nicht als gedruckt vermerkt: (Die Datenbank antwortet nicht|Keine Verbindung zum Server)\. Die Etiketten stehen weiter unter „Fehlende Etiketten"\.$/
			),
			'warning'
		);
	});

	// Ohne offenes PDF ist nichts gedruckt. Ein Vermerk nähme die Etiketten von der Liste, und
	// niemand druckte sie nach.
	it('vermerkt nichts, wenn der Browser den Tab blockiert', async () => {
		fenster = null;

		await druckeBogen(BOGEN, ETIKETTEN);

		expect(apiFetch).not.toHaveBeenCalled();
		expect(snackbars.addToast).toHaveBeenCalledWith(PDF_BLOCKIERT, 'warning');
	});
});
