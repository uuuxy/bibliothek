import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useBookAkte } from './useBookAkte.svelte.js';
import { apiFetch } from './apiFetch.js';

// Die Buch-Akte lädt fünf Dinge je Titel: Kopf, Ausleiher, Exemplare, Historie,
// Vormerkungen. Sie bleibt beim Wechsel MONTIERT — die Omnibox setzt nur
// appState.activeBookId, der Router hält `book_detail` (Router.svelte:117-124, 217-226).
//
// Bis zum Rasterdurchgang am 06.09.2026 stand hier `if (res.ok) book = ...` ohne `else`
// und ohne Sequenznummer, und beim Wechsel wurde nichts zurückgesetzt. Das ist der
// Zwilling des Schülerakten-Fundes desselben Tages (529def4d) — nur hängt hier ein
// unumkehrbarer Massenlöschbefehl daran:
//
//   „Gesamten Titel löschen" schickt `book.id`. Stand dort noch der VORIGE Titel, fällt
//   dieser — mit allen Exemplaren, Ausleihen und offenen Forderungen —, während die
//   Rückfrage die Exemplarzahl des neuen nannte.
vi.mock('./apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('../inventur/lib/store.svelte.js', () => ({
	appState: { selectedBook: null, bookToEdit: null, requestAdminView: false, activeBookId: null },
	showToast: vi.fn()
}));
vi.mock('./stores/uiStore.svelte.js', () => ({ uiStore: { activeTab: 'book_detail' } }));

/** @param {any} body */
const ok = (body) => ({ ok: true, json: async () => body });
const fehler = { ok: false, status: 429, json: async () => ({}) };

/** Antwortet je Titel-ID; `kopfFehler` lässt NUR /api/books/{id} scheitern. */
function antworten(id, kopfFehler = false) {
	return async (/** @type {any} */ url) => {
		const u = String(url);
		if (u.startsWith('/api/books/')) {
			return /** @type {any} */ (kopfFehler ? fehler : ok({ id, titel: 'Titel ' + id }));
		}
		if (u.includes('/exemplare')) return /** @type {any} */ (ok([{ id: 'e-' + id }]));
		return /** @type {any} */ (ok([]));
	};
}

describe('useBookAkte.loadAll', () => {
	beforeEach(() => vi.mocked(apiFetch).mockReset());

	it('lässt den Kopf des vorigen Titels nicht stehen — sonst löscht der Knopf den falschen', async () => {
		vi.mocked(apiFetch).mockImplementation(antworten('A'));
		const akte = useBookAkte();
		await akte.loadAll('A');
		expect(akte.book?.id).toBe('A');

		// Titel B: die Kopf-Anfrage scheitert (429 vom Rate-Limiter, fünf parallele Anfragen).
		vi.mocked(apiFetch).mockImplementation(antworten('B', true));
		await akte.loadAll('B');

		expect(akte.book, 'der Kopf von A steht über den Listen von B').toBeNull();
		expect(akte.exemplare).toEqual([{ id: 'e-B' }]);
	});

	it('lässt die überholte Antwort nicht gewinnen', async () => {
		/** @type {() => void} */
		let loesenA = () => {};
		const aKam = new Promise((r) => (loesenA = /** @type {any} */ (r)));
		vi.mocked(apiFetch).mockImplementation(async (/** @type {any} */ url) => {
			const u = String(url);
			const istA = u.includes('/A');
			if (istA) await aKam;
			if (u.startsWith('/api/books/')) {
				return /** @type {any} */ (
					ok({ id: istA ? 'A' : 'B', titel: istA ? 'Titel A' : 'Titel B' })
				);
			}
			if (u.includes('/exemplare')) return /** @type {any} */ (ok([{ id: istA ? 'e-A' : 'e-B' }]));
			return /** @type {any} */ (ok([]));
		});

		const akte = useBookAkte();
		const langsam = akte.loadAll('A');
		await akte.loadAll('B');
		loesenA();
		await langsam;

		expect(akte.book?.id, 'die überholte Antwort von A hat B überschrieben').toBe('B');
		expect(akte.exemplare).toEqual([{ id: 'e-B' }]);
	});
});
