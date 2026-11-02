import { describe, it, expect, vi } from 'vitest';

// Die Leitung meldet sich mit „connected" und „ping"; daran hängt der Herzschlag. Ohne ihn hielte
// App.svelte die Verbindung nach 25 Sekunden für verloren, obwohl sie steht.
const leitung = vi.hoisted(() => ({
	/** @type {Map<string, () => void>} */
	hoerer: new Map()
}));
vi.mock('../liveEvents.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	verbinde: vi.fn(),
	trenne: vi.fn(),
	abonniere: vi.fn((/** @type {string} */ name, /** @type {() => void} */ hoere) => {
		leitung.hoerer.set(name, hoere);
		return () => leitung.hoerer.delete(name);
	})
}));

import { abonniere } from '../liveEvents.js';
import { authStore } from './authStore.svelte.js';

describe('authStore: Herzschlag der Leitung', () => {
	it('hängt sich einmal an „connected" und „ping", und ein Lebenszeichen frischt ihn auf', () => {
		authStore.connectSSE();
		authStore.connectSSE();
		expect(vi.mocked(abonniere).mock.calls.map(([name]) => name)).toEqual(['connected', 'ping']);

		authStore.heartbeatOk = false;
		authStore.lastHeartbeatTime = 0;
		leitung.hoerer.get('ping')?.();

		expect(authStore.heartbeatOk).toBe(true);
		expect(authStore.lastHeartbeatTime).toBeGreaterThan(0);
	});
});
