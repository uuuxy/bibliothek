import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useBuchakte } from './useBuchakte.svelte.js';
import { apiFetch } from '../../apiFetch.js';
import { loescheTitelMitRueckfrage } from './titelLoeschen.js';
import { authStore } from '../../stores/authStore.svelte.js';

// „Titel löschen" in der Buchakte: Nach dem Löschen meldet die Akte das Ergebnis und geht
// zurück; bleibt der Titel, weil ein verliehenes Exemplar nicht bestätigt war, lädt sie neu.
// Ein Abbruch im Dialog tut nichts. Die Browser-Suite löscht nur über die Buchmaske.
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));
vi.mock('../../../inventur/lib/store.svelte.js', () => ({
	appState: { selectedBook: null, bookToEdit: null, requestAdminView: false, activeBookId: null },
	melde: vi.fn()
}));
vi.mock('../../stores/uiStore.svelte.js', () => ({ uiStore: { activeTab: 'book_detail' } }));
vi.mock('./titelLoeschen.js', () => ({
	loescheTitelMitRueckfrage: vi.fn(),
	meldungZumErgebnis: () => 'Ergebnis der Löschung'
}));

/** @param {any} body */
const ok = (body) => ({ ok: true, json: async () => body });

/** Lädt die Akte des Titels A und liefert sie mit der Zahl der Anfragen danach. */
async function akteVonA() {
	vi.mocked(apiFetch).mockImplementation(
		async (/** @type {any} */ url) =>
			/** @type {any} */ (
				ok(String(url).startsWith('/api/books/') ? { id: 'A', title: 'Titel A' } : [])
			)
	);
	const akte = useBuchakte();
	await akte.ladeAlles('A');
	return { akte, anfragenVorher: vi.mocked(apiFetch).mock.calls.length };
}

describe('useBuchakte.loescheTitel', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(loescheTitelMitRueckfrage).mockReset();
		authStore.currentUser = /** @type {any} */ ({ rolle: 'admin', permissions: ['*'] });
	});

	it('tut nach einem Abbruch im Dialog nichts', async () => {
		const { akte } = await akteVonA();
		vi.mocked(loescheTitelMitRueckfrage).mockResolvedValue(null);
		const melde = vi.fn();
		const zurueck = vi.fn();

		await akte.loescheTitel(melde, zurueck);

		expect(melde).not.toHaveBeenCalled();
		expect(zurueck).not.toHaveBeenCalled();
	});

	it('meldet einen gelöschten Titel und geht zurück', async () => {
		const { akte } = await akteVonA();
		vi.mocked(loescheTitelMitRueckfrage).mockResolvedValue({ geloescht: ['A'], geblieben: [] });
		const melde = vi.fn();
		const zurueck = vi.fn();

		await akte.loescheTitel(melde, zurueck);

		expect(melde).toHaveBeenCalledWith('Ergebnis der Löschung', 'success');
		expect(zurueck).toHaveBeenCalledOnce();
	});

	// Die Rückfrage nennt den Titel („‚Titel A' löschen?“). GET /api/books/{id} trägt ihn als
	// title (repository/katalog_titel.go); unter titel stand nichts, und sie hieß „Titel löschen?“.
	it('gibt der Rückfrage den Namen des Titels', async () => {
		const { akte } = await akteVonA();
		vi.mocked(loescheTitelMitRueckfrage).mockResolvedValue(null);

		await akte.loescheTitel(vi.fn(), vi.fn());

		expect(loescheTitelMitRueckfrage).toHaveBeenCalledWith(['A'], 'Titel A');
	});

	it('bleibt beim Titel, der wegen eines verliehenen Exemplars bleibt, und lädt ihn neu', async () => {
		const { akte, anfragenVorher } = await akteVonA();
		vi.mocked(loescheTitelMitRueckfrage).mockResolvedValue({ geloescht: [], geblieben: ['A'] });
		const melde = vi.fn();
		const zurueck = vi.fn();

		await akte.loescheTitel(melde, zurueck);

		expect(melde).toHaveBeenCalledWith('Ergebnis der Löschung', 'success');
		expect(zurueck).not.toHaveBeenCalled();
		expect(vi.mocked(apiFetch).mock.calls.length).toBeGreaterThan(anfragenVorher);
	});
});
