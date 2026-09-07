import { describe, it, expect, vi, beforeEach } from 'vitest';
import { flushSync } from 'svelte';
import { useBookAkte } from './useBookAkte.svelte.js';
import { apiFetch } from './apiFetch.js';
import { appState } from '../inventur/lib/store.svelte.js';

// Die Buch-Akte ruft loadAll aus einem $effect (BookAkte.svelte). Vom 06.09.2026, 21:18,
// bis zum 07.09. lief dieser Effekt in einer Endlosschleife: loadAll schrieb `book = null`,
// las `book` im selben synchronen Zug wieder (Cover-Kandidaten) und abonnierte es damit.
// Beim nächsten Anlass — ein frisches Objekt in appState.selectedBook, wie es die
// Titel-Verwaltung beim Öffnen und nach dem Speichern setzt — löste jeder Lauf mit seinem
// eigenen `book = null` den nächsten aus, bis Svelte nach 1.000 Umläufen abbrach
// (effect_update_depth_exceeded). Der Effekt starb, isLoading blieb auf true, die Akte
// zeigte für immer den Ladekringel; der Trace zählte 1.001 Anfragen an einen Endpunkt.
//
// Warum das kein anderer Test sah: useBookAkte.test.js ruft loadAll DIREKT — ohne Effekt
// gibt es kein Abonnement und keine Schleife. vitest 455/455 grün, svelte-check 0/0, und
// nur der e2e-Lauf in CI wurde rot ([[svelte-effekt-liest-eigenen-state]]).
//
// Dieser Test stellt den Effekt nach, mit dem ECHTEN reaktiven Store — die Bedingung, die
// den Mock der Schwester-Datei zum blinden Fleck machte.
vi.mock('./apiFetch.js', () => ({ apiFetch: vi.fn() }));

/** @param {any} body */
const ok = (body) => /** @type {any} */ ({ ok: true, json: async () => body });

describe('useBookAkte.loadAll in einem $effect', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(apiFetch).mockImplementation(async () => ok([]));
		appState.selectedBook = null;
	});

	it('löst sich nicht selbst aus: ein neues selectedBook-Objekt heißt EIN weiterer Lauf, keine Schleife', async () => {
		appState.selectedBook = { id: 'A', title: 'Titel A' };
		const akte = useBookAkte();
		let laeufe = 0;
		const stopp = $effect.root(() => {
			$effect(() => {
				laeufe++;
				akte.loadAll('A');
			});
		});
		try {
			flushSync();
			expect(laeufe).toBe(1);
			// Der Anlass: dasselbe Buch als frisches Objekt (Titel-Verwaltung nach dem
			// Speichern, Öffnen über den Stift). Ein Neulauf ist erlaubt — genau einer.
			appState.selectedBook = { id: 'A', title: 'Titel A (neu geladen)' };
			expect(
				() => flushSync(),
				'Svelte brach den Effekt ab (effect_update_depth_exceeded)'
			).not.toThrow();
			expect(laeufe, 'jeder Lauf löste den nächsten aus').toBe(2);
		} finally {
			stopp();
		}
	});
});
