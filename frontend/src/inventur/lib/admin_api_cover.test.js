import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../../lib/apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('./store.svelte.js', () => ({ appState: {} }));
import { apiFetch } from '../../lib/apiFetch.js';
import { coverNeuHolen } from './admin_api.js';

// „Cover neu holen" (OFFEN.md 4.16, 22.09.2026): Die Auskunft des Servers — kein Cover
// bei den Katalogdiensten (404), Dienste nicht erreichbar (502) — muss den Menschen
// erreichen, nicht eine Vorgabe-Meldung. Und der Erfolg liefert den neuen Pfad.
/** @param {boolean} ok @param {number} status @param {any} body */
const antwort = (ok, status, body) => ({ ok, status, json: async () => body });

describe('admin_api: coverNeuHolen', () => {
	beforeEach(() => vi.mocked(apiFetch).mockReset());

	it('ruft die Tür des einen Titels auf und liefert den neuen Cover-Pfad', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(
			/** @type {any} */ (antwort(true, 200, { data: { coverUrl: '/uploads/cover_auto_1.webp' } }))
		);
		await expect(coverNeuHolen('abc')).resolves.toBe('/uploads/cover_auto_1.webp');
		expect(vi.mocked(apiFetch).mock.calls[0][0]).toBe('/api/books/abc/refresh-cover');
		expect(vi.mocked(apiFetch).mock.calls[0][1]).toMatchObject({ method: 'POST' });
	});

	it('reicht die Auskunft des Servers weiter', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(
			/** @type {any} */ (
				antwort(false, 404, {
					error: 'Bei DNB, Google Books und OpenLibrary gibt es kein Cover zu dieser ISBN'
				})
			)
		);
		await expect(coverNeuHolen('abc')).rejects.toThrow('kein Cover zu dieser ISBN');
	});

	it('hat eine Vorgabe, wenn der Server keine Auskunft gibt', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(
			/** @type {any} */ ({
				ok: false,
				status: 502,
				json: async () => {
					throw new Error('kein JSON');
				}
			})
		);
		await expect(coverNeuHolen('abc')).rejects.toThrow('Cover konnte nicht neu geholt werden');
	});
});
