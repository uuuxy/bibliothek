import { describe, it, expect, vi } from 'vitest';

// leseApiFehler bleibt echt (ARCHITEKTUR.md 8.14).
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));
vi.mock('../../../inventur/lib/store.svelte.js', () => ({ melde: vi.fn() }));
vi.mock('../../stores/bestaetigung.svelte.js', () => ({ bestaetigen: vi.fn(async () => true) }));

import { apiFetch } from '../../apiFetch.js';
import { melde } from '../../../inventur/lib/store.svelte.js';
import { serverAntwort } from '../../serverAntwort.js';
import { erzeugePlaner } from './lmfplanPlaner.svelte.js';

// Ein zweiter Auftrag zum Verwerfen vor der Antwort schickte ein zweites DELETE: Nach „Plan
// verworfen." stand „Kein Plan vorhanden", in Chromium, Firefox und WebKit (OFFEN.md 5.106).
describe('LMF-Planer: Verwerfen', () => {
	it('verwirft bei einem zweiten Auftrag vor der Antwort nicht noch einmal', async () => {
		/** @type {(antwort: any) => void} */
		let liefere = () => {};
		vi.mocked(apiFetch).mockImplementation(async (_url, o) => {
			if (o?.method === 'DELETE') return new Promise((fertig) => (liefere = fertig));
			return serverAntwort(503, { error: 'Datenbank nicht erreichbar' });
		});
		const deletes = () =>
			vi.mocked(apiFetch).mock.calls.filter(([, o]) => o?.method === 'DELETE').length;
		const planer = erzeugePlaner();
		planer.zustand.stand = /** @type {any} */ ({ plan: { erster_tag: '2027-08-09' } });

		const erster = planer.verwerfen();
		await vi.waitFor(() => expect(deletes()).toBe(1));
		const zweiter = planer.verwerfen();
		await new Promise((r) => setTimeout(r, 20));

		expect(deletes(), 'ein zweites DELETE vor der Antwort').toBe(1);
		expect(planer.zustand.speichert, 'Speichern und Veröffentlichen ruhen').toBe(true);

		liefere(serverAntwort(200, {}));
		await Promise.all([erster, zweiter]);
		expect(melde).toHaveBeenCalledWith('Plan verworfen.', 'success');
		expect(planer.zustand.speichert).toBe(false);
	});
});
