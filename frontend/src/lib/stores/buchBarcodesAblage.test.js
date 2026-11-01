import 'fake-indexeddb/auto';
import { describe, it, expect, vi } from 'vitest';

vi.mock('../apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('./netzLage.svelte.js', () => ({ netzLage: { offline: false } }));

// Ohne Netz ist die Ablage die einzige Quelle: Nach dem Neuladen der Seite muss die Liste von dort
// zurückkommen, sonst gilt jede nackte Nummer an der Theke als unklar.
describe('Barcode-Liste: die Ablage', () => {
	it('überlebt das Neuladen der Seite', async () => {
		const { apiFetch } = await import('../apiFetch.js');
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				status: 200,
				ok: true,
				json: async () => ({ barcodes: ['B-7'], stand: 'abc' })
			})
		);
		const { buchBarcodes } = await import('./buchBarcodes.svelte.js');
		await buchBarcodes.auffrischen();

		vi.resetModules();
		const { buchBarcodes: nachDemNeuladen } = await import('./buchBarcodes.svelte.js');
		await nachDemNeuladen.laden();

		expect(nachDemNeuladen.istBuch('B-7')).toBe(true);
		expect(nachDemNeuladen.anzahl).toBe(1);
	});
});
