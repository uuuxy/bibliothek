import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

vi.mock('./apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('./stores/bestaetigung.svelte.js', () => ({ bestaetigen: vi.fn() }));
vi.mock('./stores/snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));

import { apiFetch } from './apiFetch.js';
import { bestaetigen } from './stores/bestaetigung.svelte.js';
import { snackbars } from './stores/snackbars.svelte.js';
import { druckeBogen } from './etikettenDruck.js';

const LMF = 'LMF-9783060158423-001';
const BOGEN = { formatId: 'zweckform_l4760', startFeld: 1, alsQR: false };
const ETIKETTEN = [
	{ barcode_id: 'B-10001', titel: 'Natur und Technik', autor: '' },
	{ barcode_id: LMF, titel: 'Natur und Technik', autor: '' }
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

		await druckeBogen(BOGEN, [ETIKETTEN[0]], vi.fn());

		expect(nummernIm(rumpfe('/api/print/labels')[0])).toEqual(['B-10001']);
		expect(rumpfe('/api/exemplare/etiketten-gedruckt')[0].barcode_ids).toEqual(['B-10001']);
		expect(window.open).toHaveBeenCalledOnce();
		expect(bestaetigen).not.toHaveBeenCalled();
	});

	// Der Server nennt die LMF-Nummer; nach dem Ja trägt das Exemplar eine kurze Nummer, und mit
	// ihr gehen Druck und Vermerk.
	it('fragt bei einer LMF-Nummer und druckt mit der neuen Nummer', async () => {
		const ersetzt = vi.fn();
		vi.mocked(bestaetigen).mockResolvedValueOnce(true);
		vi.mocked(apiFetch)
			.mockResolvedValueOnce(antwort(409, { error: 'passt nicht', lange_nummern: [LMF] }))
			.mockResolvedValueOnce(antwort(200, { nummern: { [LMF]: 'B-10023' } }))
			.mockResolvedValueOnce(antwort(200))
			.mockResolvedValueOnce(antwort(200));

		await druckeBogen(BOGEN, ETIKETTEN, ersetzt);

		const frage = vi.mocked(bestaetigen).mock.calls[0][0];
		expect(frage.titel).toBe('Neue Nummer vergeben?');
		expect(frage.text).toContain(LMF);
		expect(frage.text).toContain('nicht mehr scannen');
		expect(frage.aktion).toBe('Neue Nummer vergeben');
		expect(rumpfe('/api/exemplare/neue-nummern')).toEqual([{ barcode_ids: [LMF] }]);
		expect(ersetzt).toHaveBeenCalledWith({ [LMF]: 'B-10023' });
		const drucke = rumpfe('/api/print/labels');
		expect(drucke.map(nummernIm)).toEqual([
			['B-10001', LMF],
			['B-10001', 'B-10023']
		]);
		expect(rumpfe('/api/exemplare/etiketten-gedruckt')[0].barcode_ids).toEqual([
			'B-10001',
			'B-10023'
		]);
		expect(window.open).toHaveBeenCalledOnce();
	});

	it('nennt bei mehreren Nummern ihre Zahl', async () => {
		vi.mocked(bestaetigen).mockResolvedValueOnce(false);
		vi.mocked(apiFetch).mockResolvedValueOnce(
			antwort(409, { lange_nummern: [LMF, 'LMF-9783060158423-002'] })
		);

		await druckeBogen(BOGEN, ETIKETTEN, vi.fn());

		const frage = vi.mocked(bestaetigen).mock.calls[0][0];
		expect(frage.titel).toBe('2 neue Nummern vergeben?');
		expect(frage.aktion).toBe('Neue Nummern vergeben');
	});

	it('vergibt und druckt nichts, wenn die Rückfrage abgebrochen wird', async () => {
		const ersetzt = vi.fn();
		vi.mocked(bestaetigen).mockResolvedValueOnce(false);
		vi.mocked(apiFetch).mockResolvedValueOnce(antwort(409, { lange_nummern: [LMF] }));

		await druckeBogen(BOGEN, ETIKETTEN, ersetzt);

		expect(apiFetch).toHaveBeenCalledOnce();
		expect(ersetzt).not.toHaveBeenCalled();
		expect(window.open).not.toHaveBeenCalled();
	});

	it('meldet eine gescheiterte Vergabe und druckt nicht', async () => {
		const ersetzt = vi.fn();
		vi.mocked(bestaetigen).mockResolvedValueOnce(true);
		vi.mocked(apiFetch)
			.mockResolvedValueOnce(antwort(409, { lange_nummern: [LMF] }))
			.mockResolvedValueOnce(antwort(404, { error: 'Eine der Nummern trägt kein Exemplar mehr' }));

		await druckeBogen(BOGEN, ETIKETTEN, ersetzt);

		expect(snackbars.addToast).toHaveBeenCalledWith(
			'Eine der Nummern trägt kein Exemplar mehr',
			'error'
		);
		expect(rumpfe('/api/print/labels')).toHaveLength(1);
		expect(ersetzt).not.toHaveBeenCalled();
		expect(window.open).not.toHaveBeenCalled();
	});

	// Eine Ablehnung, die keine Nummer nennt, lässt sich nicht beantworten: Sie ist ein Fehler.
	it('meldet eine Ablehnung ohne Nummern als Fehler', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(antwort(409, {}));

		await druckeBogen(BOGEN, ETIKETTEN, vi.fn());

		expect(bestaetigen).not.toHaveBeenCalled();
		expect(snackbars.addToast).toHaveBeenCalledWith('Fehler beim Erstellen des PDFs', 'error');
	});
});
