import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { flushSync } from 'svelte';

vi.mock('../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));
vi.mock('./offlineSync.svelte.js', () => ({
	offlineSync: { isOffline: false, pendingCount: 0, startSync: async () => {} }
}));
vi.mock('./netzLage.svelte.js', () => ({ netzLage: { beiRueckkehr: () => () => {} } }));
vi.mock('../liveEvents.js', () => ({
	abonniere: vi.fn(() => vi.fn()),
	verbinde: vi.fn(),
	trenne: vi.fn()
}));

import { apiFetch } from '../apiFetch.js';
import { IdleLock } from './idleLock.svelte.js';
import { authStore } from './authStore.svelte.js';

// App.svelte stellt den Wächter aus einem $effect scharf. Hinge der Effekt am Zustand, den
// start() liest und schreibt, löste jeder Lauf den nächsten aus; Tests, die start() direkt
// rufen, sehen das nicht.
describe('idleLock.start in einem $effect', () => {
	/** @type {IdleLock} */
	let lock;

	beforeEach(() => {
		vi.useFakeTimers();
		localStorage.clear();
		vi.mocked(apiFetch).mockReset();
		vi.mocked(apiFetch).mockImplementation(
			async () => /** @type {any} */ ({ ok: false, status: 423 })
		);
		lock = new IdleLock();
		authStore.currentUser = { email: 'theke@schule.example', permissions: [] };
		authStore.isLoggedIn = true;
	});

	afterEach(() => {
		lock.stop();
		authStore.isLoggedIn = false;
		authStore.currentUser = null;
		vi.useRealTimers();
	});

	/** Der Effekt aus App.svelte, mit Zähler. */
	function appEffekt() {
		const stand = { laeufe: 0 };
		const stopp = $effect.root(() => {
			$effect(() => {
				stand.laeufe++;
				if (!authStore.isLoggedIn) {
					lock.stop();
					return;
				}
				lock.start();
				lock.ladeFristen();
				return () => lock.stop();
			});
		});
		return { stand, stopp };
	}

	it('Start in eine gesperrte Anmeldung: ein Lauf, verdeckt, keine Schleife', () => {
		lock.verdeckeVorDemStart();
		const { stand, stopp } = appEffekt();
		try {
			expect(
				() => flushSync(),
				'Svelte brach den Effekt ab (effect_update_depth_exceeded)'
			).not.toThrow();
			expect(stand.laeufe, 'jeder Lauf löste den nächsten aus').toBe(1);
			expect(lock.gesperrt).toBe(true);
			expect(vi.mocked(apiFetch).mock.calls.length, 'je Lauf eine Anfrage nach den Fristen').toBe(
				1
			);
		} finally {
			stopp();
		}
	});

	it('eine Sperre zur Laufzeit und geänderte Fristen stoßen den Effekt nicht neu an', () => {
		const { stand, stopp } = appEffekt();
		try {
			flushSync();
			expect(stand.laeufe).toBe(1);

			lock.sperren();
			flushSync();
			expect(lock.gesperrt).toBe(true);
			expect(stand.laeufe, 'der Neulauf riefe stop() und höbe die Sperre im Fenster auf').toBe(1);

			// Die Bibliothek ändert die Fristen, während dieses Fenster gesperrt ist.
			lock.sperreMinuten = 30;
			lock.thekeLeerenMinuten = 10;
			flushSync();
			expect(stand.laeufe).toBe(1);
			expect(lock.gesperrt, 'die Sperre darf nicht mit den Fristen fallen').toBe(true);
		} finally {
			stopp();
		}
	});
});
