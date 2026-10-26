import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { zeigePdf } from './pdfImFenster.js';

/** Ein Tab, wie window.open ihn liefert, soweit zeigePdf ihn anfasst. */
function neuesFenster() {
	return { closed: false, close: vi.fn(), location: { replace: vi.fn() } };
}

/**
 * @param {number} status
 * @returns {any}
 */
const antwort = (status) => ({
	ok: status < 400,
	status,
	blob: async () => new Blob(['%PDF'], { type: 'application/pdf' })
});

describe('zeigePdf', () => {
	/** @type {ReturnType<typeof neuesFenster>} */
	let fenster;
	const erzeuge = URL.createObjectURL;
	const gibFrei = URL.revokeObjectURL;

	beforeEach(() => {
		vi.useFakeTimers();
		fenster = neuesFenster();
		vi.stubGlobal(
			'open',
			vi.fn(() => fenster)
		);
		URL.createObjectURL = vi.fn(() => 'blob:probe');
		URL.revokeObjectURL = vi.fn();
	});
	afterEach(() => {
		vi.useRealTimers();
		vi.unstubAllGlobals();
		URL.createObjectURL = erzeuge;
		URL.revokeObjectURL = gibFrei;
	});

	it('öffnet den Tab vor der Anfrage und lädt das PDF hinein', async () => {
		/** @type {string[]} */
		const reihenfolge = [];
		vi.mocked(window.open).mockImplementation(() => {
			reihenfolge.push('Tab');
			return /** @type {any} */ (fenster);
		});

		const ergebnis = await zeigePdf(async () => {
			reihenfolge.push('Anfrage');
			return antwort(200);
		});

		expect(reihenfolge).toEqual(['Tab', 'Anfrage']);
		expect(window.open).toHaveBeenCalledWith('', '_blank');
		expect(fenster.location.replace).toHaveBeenCalledWith('blob:probe');
		expect(ergebnis?.ok).toBe(true);
	});

	it('gibt die Adresse erst frei, wenn der Tab zu ist', async () => {
		await zeigePdf(async () => antwort(200));

		vi.advanceTimersByTime(60_000);
		expect(
			URL.revokeObjectURL,
			'Speichern im offenen Tab braucht die Adresse'
		).not.toHaveBeenCalled();

		fenster.closed = true;
		vi.advanceTimersByTime(10_000);
		expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:probe');
	});

	it('schließt den Tab bei einer Fehlerantwort und gibt sie dem Aufrufer', async () => {
		const ergebnis = await zeigePdf(async () => antwort(500));

		expect(fenster.close).toHaveBeenCalledOnce();
		expect(ergebnis?.status).toBe(500);
		expect(URL.createObjectURL).not.toHaveBeenCalled();
	});

	it('schließt den Tab, wenn die Anfrage scheitert', async () => {
		await expect(
			zeigePdf(async () => {
				throw new Error('Netz weg');
			})
		).rejects.toThrow('Netz weg');
		expect(fenster.close).toHaveBeenCalledOnce();
	});

	it('fragt nicht beim Server an, wenn der Browser den Tab blockiert', async () => {
		vi.mocked(window.open).mockReturnValue(null);
		const hole = vi.fn();

		expect(await zeigePdf(hole)).toBeNull();
		expect(hole).not.toHaveBeenCalled();
	});
});
