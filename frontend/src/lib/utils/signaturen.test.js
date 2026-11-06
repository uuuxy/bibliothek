import { describe, it, expect, vi, beforeEach } from 'vitest';
import { ladeCoverZurSignatur } from './signaturen.js';
import { apiFetch } from '../apiFetch.js';

vi.mock('../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

beforeEach(() => {
	vi.mocked(apiFetch).mockReset();
});

const titel = { titel_id: 't1', titel: 'Emil', isbn: '978', cover_url: '/uploads/emil.webp' };

// Die Vorschau am Vorschlagsmenü der Signatur zeigt zwei Cover. Sie fragt die Regalansicht mit
// kleiner Grenze und nur nach Titeln mit Cover; ohne beides holte sie bis zu 500 Zeilen und
// zeigte leere Plätze.
describe('ladeCoverZurSignatur', () => {
	it('fragt höchstens zwei Titel mit Cover ab und liefert sie', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ({ buecher: [titel] }) })
		);

		await expect(ladeCoverZurSignatur('LMF Deu 7')).resolves.toEqual([titel]);
		expect(apiFetch).toHaveBeenCalledWith(
			'/api/signaturen/buecher?signatur=LMF+Deu+7&limit=2&nur_mit_cover=true'
		);
	});

	it('fragt je Signatur nur einmal, auch wenn der Zeiger zurückkehrt', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ({ buecher: [titel] }) })
		);

		await ladeCoverZurSignatur('Sd');
		await ladeCoverZurSignatur('Sd');
		expect(apiFetch).toHaveBeenCalledTimes(1);
	});

	it('liefert ohne Antwort eine leere Liste und fragt beim nächsten Mal neu', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(/** @type {any} */ ({ ok: false, status: 500 }));
		await expect(ladeCoverZurSignatur('Pg')).resolves.toEqual([]);

		vi.mocked(apiFetch).mockRejectedValueOnce(new Error('offline'));
		await expect(ladeCoverZurSignatur('Pg')).resolves.toEqual([]);

		expect(apiFetch).toHaveBeenCalledTimes(2);
	});
});
