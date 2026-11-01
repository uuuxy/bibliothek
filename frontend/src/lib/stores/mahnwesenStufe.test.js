import { describe, it, expect, vi } from 'vitest';

vi.mock('../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

import { apiFetch } from '../apiFetch.js';
import { mahnwesenStore } from './mahnwesen.svelte.js';

// Die Stufe entscheidet, in welchem Reiter ein Kind steht und wie die Zeile aussieht: bis 14 Tage
// über der Frist die Erinnerung, danach die Mahnung.
describe('Mahnwesen: die Stufe eines Kindes', () => {
	it('hat die Grenze bei 14 Tagen', async () => {
		const kind = (/** @type {string} */ name, /** @type {number} */ tage) => ({
			id: name,
			name,
			medien: [{ tage_ueberfaellig: tage }]
		});
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({
					klassen: [
						{
							klasse: '5a',
							lehrer_email: '',
							schueler: [kind('frisch', 0), kind('vierzehn', 14), kind('fuenfzehn', 15)]
						}
					]
				})
			})
		);

		await mahnwesenStore.lade();

		const stufe = Object.fromEntries(
			mahnwesenStore.gefilterteSchueler.map((/** @type {any} */ s) => [s.name, s.mahnstufe])
		);
		expect(stufe).toEqual({
			frisch: '1. Erinnerung',
			vierzehn: '1. Erinnerung',
			fuenfzehn: 'Mahnung'
		});
	});
});
