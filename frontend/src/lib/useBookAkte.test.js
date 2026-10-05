import { describe, it, expect, vi, beforeEach } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { srcRoot, ohneKommentare } from './hygiene-quellen.js';
import { useBookAkte } from './useBookAkte.svelte.js';
import { apiFetch } from './apiFetch.js';
import { appState } from '../inventur/lib/store.svelte.js';

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
vi.mock('./apiFetch.js', () => ({
	apiFetch: vi.fn(),
	extractApiError: vi.fn(async (/** @type {any} */ res) => {
		const text = await res.text();
		return text ? JSON.parse(text).error : `Fehler ${res.status}`;
	})
}));
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

	// „(0)" ist eine Aussage über den Bestand. Scheitert der Abruf einer Liste, hat
	// niemand diese Aussage geprüft — bis zum Sweep am 06.09.2026 machte die Akte sie
	// trotzdem („Ausleiher (0)" für einen Titel, der Ausleiher hat).
	it('nennt die Listen, die nicht geladen werden konnten', async () => {
		vi.mocked(apiFetch).mockImplementation(async (/** @type {any} */ url) => {
			const u = String(url);
			if (u.startsWith('/api/books/'))
				return /** @type {any} */ (ok({ id: 'A', titel: 'Titel A' }));
			if (u.includes('/ausleiher')) return /** @type {any} */ ({ ok: false, status: 500 });
			if (u.includes('/historie')) return /** @type {any} */ ({ ok: false, status: 500 });
			return /** @type {any} */ (ok([]));
		});
		const akte = useBookAkte();
		await akte.loadAll('A');

		expect(akte.fehlendeListen).toEqual(['Ausleiher', 'Historie']);
		expect(akte.borrowers, 'die Anzeige braucht trotzdem ein Array').toEqual([]);
	});

	// „Buch nicht gefunden." ist ebenfalls eine Aussage über den Bestand, und die Akte traf
	// sie bis zum 12.09.2026 auch dann, wenn der KOPF nicht geladen werden konnte
	// (`kopf = res.ok ? await res.json() : null`, Register 10.09.2026). Der Unterschied
	// zählt: Bei „nicht gefunden" legt die Bibliothek den Titel neu an.
	it('unterscheidet einen nicht geladenen Kopf von einem Titel, den es nicht gibt', async () => {
		vi.mocked(apiFetch).mockImplementation(async (/** @type {any} */ url) => {
			const u = String(url);
			if (u.startsWith('/api/books/'))
				return /** @type {any} */ ({
					ok: false,
					status: 500,
					text: async () => JSON.stringify({ error: 'Datenbank nicht erreichbar' })
				});
			return /** @type {any} */ (ok([]));
		});
		const akte = useBookAkte();
		await akte.loadAll('A');
		expect(akte.book).toBeNull();
		expect(akte.kopfFehler).toBe('Datenbank nicht erreichbar');

		// Der 404 dagegen IST die Auskunft „gibt es nicht" — dann bleibt es beim alten Bild.
		vi.mocked(apiFetch).mockImplementation(async (/** @type {any} */ url) =>
			String(url).startsWith('/api/books/')
				? /** @type {any} */ ({ ok: false, status: 404, text: async () => '' })
				: /** @type {any} */ (ok([]))
		);
		await akte.loadAll('B');
		expect(akte.book).toBeNull();
		expect(akte.kopfFehler).toBe('');
	});

	it('vergisst die Fehlliste beim nächsten Titel', async () => {
		vi.mocked(apiFetch).mockImplementation(async (/** @type {any} */ url) => {
			const u = String(url);
			if (u.includes('/ausleiher')) return /** @type {any} */ ({ ok: false, status: 500 });
			if (u.startsWith('/api/books/')) return /** @type {any} */ (ok({ id: 'A' }));
			return /** @type {any} */ (ok([]));
		});
		const akte = useBookAkte();
		await akte.loadAll('A');
		expect(akte.fehlendeListen).toEqual(['Ausleiher']);

		vi.mocked(apiFetch).mockImplementation(antworten('B'));
		await akte.loadAll('B');
		expect(akte.fehlendeListen).toEqual([]);
	});

	// Die zweite Hälfte der Kette: Die Liste nützt nur, solange der Reiter sie fragt.
	// Am Quelltext OHNE Kommentare — sonst genügte der erklärende Satz daneben.
	it('der Reiter zeigt ein Fragezeichen statt einer ungeprüften Null', () => {
		const quelle = ohneKommentare(readFileSync(join(srcRoot, 'lib', 'BookAkte.svelte'), 'utf8'));
		expect(quelle).toMatch(/fehlendeListen\.includes\(name\)\s*\?\s*'\?'/);
	});
});

// Woher der Kopf kommt, und was ein Lauf noch schreiben darf, den ein jüngerer überholt hat.
describe('useBookAkte.loadAll: Kopf und überholte Läufe', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		appState.selectedBook = null;
	});

	/** Die Adressen, die die Akte abgerufen hat. */
	const abrufe = () => vi.mocked(apiFetch).mock.calls.map((c) => String(c[0]));

	// Die Titel-Verwaltung reicht den Titel mit, den sie gerade geöffnet hat. Für diesen Titel
	// spart die Akte den Abruf; für jeden anderen wäre der mitgereichte Kopf der falsche.
	it('nimmt den mitgereichten Titel nur als Kopf, wenn es derselbe ist', async () => {
		vi.mocked(apiFetch).mockImplementation(antworten('A'));
		const akte = useBookAkte();

		appState.selectedBook = { id: 'X', titel: 'ein anderer Titel' };
		await akte.loadAll('A');
		expect(akte.book).toEqual({ id: 'A', titel: 'Titel A' });
		expect(abrufe()).toContain('/api/books/A');

		vi.mocked(apiFetch).mockClear();
		appState.selectedBook = { id: 'A', titel: 'mitgereicht' };
		await akte.loadAll('A');
		expect(akte.book).toEqual({ id: 'A', titel: 'mitgereicht' });
		expect(abrufe()).not.toContain('/api/books/A');
		expect(abrufe(), 'die Listen lädt die Akte trotzdem').toHaveLength(4);
	});

	it('nennt den Netzwerkfehler, wenn der Kopf gar nicht ankommt', async () => {
		const konsole = vi.spyOn(console, 'error').mockImplementation(() => {});
		vi.mocked(apiFetch).mockImplementation(async (/** @type {any} */ url) => {
			if (String(url).startsWith('/api/books/')) throw new Error('Failed to fetch');
			return /** @type {any} */ (ok([]));
		});
		const akte = useBookAkte();

		await akte.loadAll('A');

		expect(akte.book).toBeNull();
		expect(akte.kopfFehler).toBe('Der Titel konnte nicht geladen werden (Netzwerkfehler).');
		expect(akte.isLoading).toBe(false);
		konsole.mockRestore();
	});

	it('ein überholter Lauf meldet seinen Netzwerkfehler nicht', async () => {
		/** @type {(grund: Error) => void} */
		let scheitereA = () => {};
		const kopfVonA = new Promise((_, ablehnen) => (scheitereA = ablehnen));
		vi.mocked(apiFetch).mockImplementation(async (/** @type {any} */ url) => {
			const u = String(url);
			if (u === '/api/books/A') return /** @type {any} */ (kopfVonA);
			if (u.startsWith('/api/books/')) return /** @type {any} */ (ok({ id: 'B' }));
			return /** @type {any} */ (ok([]));
		});
		const akte = useBookAkte();

		const langsam = akte.loadAll('A');
		await akte.loadAll('B');
		scheitereA(new Error('Failed to fetch'));
		await langsam;

		expect(akte.book, 'der gescheiterte Lauf A hat den Kopf von B geleert').toEqual({ id: 'B' });
		expect(akte.kopfFehler, 'der Fehler von A steht über dem Titel B').toBe('');
	});

	// Zwischen der Antwort und ihrem Körper kann die Akte schon beim nächsten Titel stehen. Der
	// Kopf von A läge sonst über den Listen von B, und „Gesamten Titel löschen" träfe A.
	it.each(['mitgereicht', 'vom Server'])(
		'der Körper des Kopfs von A kommt, als B schon steht (B %s): der Kopf von B bleibt',
		async (weg) => {
			/** @type {((kopf: any) => void) | null} */
			let koerperVonA = null;
			vi.mocked(apiFetch).mockImplementation(async (/** @type {any} */ url) => {
				const u = String(url);
				if (u === '/api/books/A')
					return /** @type {any} */ ({
						ok: true,
						json: () => new Promise((kommt) => (koerperVonA = kommt))
					});
				if (u.startsWith('/api/books/'))
					return /** @type {any} */ (ok({ id: 'B', titel: 'Titel B' }));
				if (u.includes('/exemplare'))
					return /** @type {any} */ (ok([{ id: u.includes('/A/') ? 'e-A' : 'e-B' }]));
				return /** @type {any} */ (ok([]));
			});
			const akte = useBookAkte();

			const alt = akte.loadAll('A');
			await vi.waitFor(() => expect(koerperVonA).not.toBeNull());
			if (weg === 'mitgereicht') appState.selectedBook = { id: 'B', titel: 'Titel B' };
			await akte.loadAll('B');
			/** @type {any} */ (koerperVonA)({ id: 'A', titel: 'Titel A' });
			await alt;

			expect(akte.book, 'der Kopf von A steht über den Listen von B').toEqual({
				id: 'B',
				titel: 'Titel B'
			});
			expect(akte.exemplare).toEqual([{ id: 'e-B' }]);
			expect(abrufe(), 'der überholte Lauf lädt keine Listen mehr').not.toContain(
				'/api/buecher/titel/A/exemplare'
			);
		}
	);

	it('ein überholter Lauf beendet die Ladeanzeige des jüngeren nicht', async () => {
		/** @type {Record<string, (antwort: any) => void>} */
		const kommt = {};
		vi.mocked(apiFetch).mockImplementation(async (/** @type {any} */ url) => {
			const u = String(url);
			if (!u.startsWith('/api/books/')) return /** @type {any} */ (ok([]));
			return /** @type {any} */ (new Promise((antworte) => (kommt[u] = antworte)));
		});
		const akte = useBookAkte();

		const alt = akte.loadAll('A');
		const jung = akte.loadAll('B');
		kommt['/api/books/A'](ok({ id: 'A' }));
		await alt;

		expect(akte.isLoading, 'B lädt noch').toBe(true);
		expect(akte.book).toBeNull();

		kommt['/api/books/B'](ok({ id: 'B' }));
		await jung;
		expect(akte.isLoading).toBe(false);
		expect(akte.book).toEqual({ id: 'B' });
	});

	// Der Kopf von A ist schon da, seine Listen noch nicht: Wechselt die Akte jetzt zu B,
	// dürfen die Listen von A nicht mehr unter dem Kopf von B landen.
	it('ein beim Laden der Listen überholter Lauf schreibt sie nicht mehr', async () => {
		/** @type {() => void} */
		let listenVonAKommen = () => {};
		const warteA = new Promise((r) => (listenVonAKommen = /** @type {any} */ (r)));
		vi.mocked(apiFetch).mockImplementation(async (/** @type {any} */ url) => {
			const u = String(url);
			const istA = u.includes('/A') || u.endsWith('=A');
			if (u.startsWith('/api/books/')) return /** @type {any} */ (ok({ id: istA ? 'A' : 'B' }));
			if (istA) await warteA;
			if (u.includes('/exemplare')) return /** @type {any} */ (ok([{ id: istA ? 'e-A' : 'e-B' }]));
			// Die übrigen Listen von A scheitern, die von B kommen an.
			return /** @type {any} */ (istA ? { ok: false, status: 500 } : ok([]));
		});
		const akte = useBookAkte();

		const langsam = akte.loadAll('A');
		await vi.waitFor(() => expect(abrufe()).toContain('/api/buecher/titel/A/exemplare'));
		await akte.loadAll('B');
		listenVonAKommen();
		await langsam;

		expect(akte.book).toEqual({ id: 'B' });
		expect(akte.exemplare).toEqual([{ id: 'e-B' }]);
		expect(akte.fehlendeListen, 'die Fehlliste von A steht unter dem Titel B').toEqual([]);
	});
});
