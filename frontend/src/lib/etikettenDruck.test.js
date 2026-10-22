import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

vi.mock('./apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('./stores/snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));

import { apiFetch } from './apiFetch.js';
import { snackbars } from './stores/snackbars.svelte.js';
import { druckeBogen } from './etikettenDruck.js';

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

beforeEach(() => {
	vi.clearAllMocks();
	vi.stubGlobal('open', vi.fn());
	URL.createObjectURL = vi.fn(() => 'blob:probe');
});
afterEach(() => vi.unstubAllGlobals());

describe('druckeBogen', () => {
	it('druckt den Bogen und vermerkt die Etiketten als gedruckt', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(antwort(200)).mockResolvedValueOnce(antwort(200));

		await druckeBogen(BOGEN, ETIKETTEN);

		expect(rumpfe('/api/print/labels').map(nummernIm)).toEqual([['B-10001', 'B-10002']]);
		expect(rumpfe('/api/exemplare/etiketten-gedruckt')[0].barcode_ids).toEqual([
			'B-10001',
			'B-10002'
		]);
		expect(window.open).toHaveBeenCalledOnce();
	});

	it('meldet einen gescheiterten Druck und vermerkt nichts', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(antwort(500));

		await druckeBogen(BOGEN, ETIKETTEN);

		expect(snackbars.addToast).toHaveBeenCalledWith('Fehler beim Erstellen des PDFs', 'error');
		expect(rumpfe('/api/exemplare/etiketten-gedruckt')).toHaveLength(0);
		expect(window.open).not.toHaveBeenCalled();
	});
});
