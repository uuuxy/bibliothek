import { describe, it, expect, vi, beforeEach } from 'vitest';
import { apiFetch } from '../../apiFetch.js';
import { statistikAbruf } from './statistikAbruf.svelte.js';

vi.mock('../../apiFetch.js', () => ({
	apiFetch: vi.fn(),
	registriereSitzungAbgelaufenHandler: vi.fn()
}));

/** Eine Antwort, deren Ankunft der Test selbst auslöst. @param {any} json */
function verzoegert(json) {
	/** @type {() => void} */
	let freigeben = () => {};
	const antwort = new Promise((resolve) => {
		freigeben = () => resolve({ ok: true, json: async () => json });
	});
	return { antwort, freigeben };
}

describe('statistikAbruf', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
	});

	it('eine langsame Antwort überholt keine schnellere', async () => {
		const alt = verzoegert({ zeitraum: 'all' });
		const neu = verzoegert({ zeitraum: 'monat' });
		vi.mocked(apiFetch)
			.mockReturnValueOnce(/** @type {any} */ (alt.antwort))
			.mockReturnValueOnce(/** @type {any} */ (neu.antwort));

		const abruf = statistikAbruf();
		const erster = abruf.laden('zeitraum=all');
		const zweiter = abruf.laden('zeitraum=monat');

		neu.freigeben();
		await zweiter;
		expect(abruf.daten).toEqual({ zeitraum: 'monat' });
		expect(abruf.loading).toBe(false);

		alt.freigeben();
		await erster;
		expect(abruf.daten, 'die alte Antwort darf die neue nicht überschreiben').toEqual({
			zeitraum: 'monat'
		});
	});

	it('ein Fehler ist ein Zustand, keine leere Seite', async () => {
		vi.spyOn(console, 'error').mockImplementation(() => {});
		vi.mocked(apiFetch).mockResolvedValueOnce(/** @type {any} */ ({ ok: false, status: 503 }));

		const abruf = statistikAbruf();
		await abruf.laden('limit=100');

		expect(abruf.fehler).toBe(true);
		expect(abruf.daten).toBeNull();
		expect(abruf.loading).toBe(false);
	});

	it('ein erneuter Abruf räumt den Fehler', async () => {
		vi.spyOn(console, 'error').mockImplementation(() => {});
		vi.mocked(apiFetch)
			.mockRejectedValueOnce(new Error('Netz weg'))
			.mockResolvedValueOnce(/** @type {any} */ ({ ok: true, json: async () => ({ ok: 1 }) }));

		const abruf = statistikAbruf();
		await abruf.laden('limit=100');
		expect(abruf.fehler).toBe(true);

		await abruf.laden('limit=100');
		expect(abruf.fehler).toBe(false);
		expect(abruf.daten).toEqual({ ok: 1 });
	});
});
