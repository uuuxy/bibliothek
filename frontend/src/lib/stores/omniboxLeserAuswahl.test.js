import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

vi.mock('../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn(),
	apiClient: { post: vi.fn() }
}));

import { apiFetch, apiClient } from '../apiFetch.js';
import { createOmniboxStore } from './omnibox.svelte.js';

/**
 * Eine Trefferliste mit genau einem Leser.
 * @param {any} leser
 * @returns {Response}
 */
function trefferliste(leser) {
	return /** @type {any} */ ({
		ok: true,
		json: async () => ({ students: [leser], books: [] })
	});
}

/** Antwort von POST /api/action auf einen geladenen Leser. @param {any} leser */
function geladen(leser) {
	return /** @type {any} */ ({
		ok: true,
		json: async () => ({ type: 'student', student: leser })
	});
}

// Die Auswahl eines Treffers an der Theke muss den LESER laden — auch einen Kollegen
// ohne Ausweisnummer.
//
// Bis zum 16.09.2026 schrieb selectDropdownItem die Ausweisnummer in die Scanleiste und
// schickte sie los. Bei einem Kollegen aus der Selbstanmeldung ist die leer: submitAction
// bricht bei leerer Eingabe ab, und der Klick tat sichtbar gar nichts — kein Profil,
// keine Meldung. Seitdem geht die ID hinaus, die jeder Treffer hat.
describe('Omnibox: Auswahl aus der Trefferliste', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		vi.mocked(apiClient.post).mockReset();
	});
	afterEach(() => vi.useRealTimers());

	/** @param {any} leser */
	async function waehleErstenTreffer(leser) {
		const store = createOmniboxStore();
		vi.mocked(apiFetch).mockImplementation(async () => trefferliste(leser));
		vi.mocked(apiClient.post).mockImplementation(async () => geladen(leser));

		store.queryVal = 'Wendlandt';
		store.handleInput();
		await vi.advanceTimersByTimeAsync(300);
		expect(store.isDropdownOpen, 'die Trefferliste ist gar nicht aufgegangen').toBe(true);

		store.selectDropdownItem(0, null);
		await vi.advanceTimersByTimeAsync(0);
		return store;
	}

	it('lädt einen Kollegen ohne Ausweisnummer über seine ID', async () => {
		const store = await waehleErstenTreffer({
			id: 'l1',
			vorname: 'Hendrik',
			nachname: 'Wendlandt',
			art: 'liv',
			barcode_id: ''
		});

		expect(vi.mocked(apiClient.post)).toHaveBeenCalledTimes(1);
		const [pfad, koerper] = vi.mocked(apiClient.post).mock.calls[0];
		expect(pfad).toBe('/api/action');
		expect(
			/** @type {any} */ (koerper).query,
			'ohne Ausweisnummer ging eine leere Eingabe hinaus — der Klick tat nichts'
		).toBe('leser:l1');
		expect(/** @type {any} */ (store.activeStudent)?.id).toBe('l1');
	});

	it('lädt auch einen Schüler über seine ID, nicht über die Ausweisnummer', async () => {
		await waehleErstenTreffer({
			id: 's1',
			vorname: 'Lena',
			nachname: 'Hoffmann',
			art: 'schueler',
			barcode_id: 'S-00012'
		});

		const [, koerper] = vi.mocked(apiClient.post).mock.calls[0];
		expect(/** @type {any} */ (koerper).query).toBe('leser:s1');
	});
});
