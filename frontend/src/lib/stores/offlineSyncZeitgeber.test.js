import 'fake-indexeddb/auto';
import { describe, it, expect, vi } from 'vitest';

const netz = vi.hoisted(() => ({ offline: false, beiRueckkehr: vi.fn() }));
vi.mock('./netzLage.svelte.js', () => ({ netzLage: netz }));
vi.mock('../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiClient: { post: vi.fn() }
}));
vi.mock('../../inventur/lib/store.svelte.js', () => ({ melde: vi.fn() }));
vi.mock('../utils/audio.js', () => ({
	spieleErfolgston: vi.fn(),
	spieleFehlerton: vi.fn()
}));

import { apiClient } from '../apiFetch.js';
import { enqueueOfflineAction } from './offlineWarteschlange.js';
import { offlineSync } from './offlineSync.svelte.js';
import { serverAntwort } from '../serverAntwort.js';

// Jede Minute ein Anlauf: Er holt nach, was liegen blieb, wenn das Zurückkehren des Netzes kein
// Ereignis auslöste oder ein Eintrag nach 5xx wartet. Ohne Netz fragt er nicht an.
describe('offlineSync: der Anlauf jede Minute', () => {
	it('sendet die Warteschlange, sobald Netz da ist, und schweigt ohne Netz', async () => {
		vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] });
		try {
			vi.mocked(apiClient.post).mockResolvedValue(serverAntwort(200, { ergebnisse: [] }));
			await enqueueOfflineAction({
				id: crypto.randomUUID(),
				art: 'rueckgabe',
				barcode: 'B-1',
				leser_id: null,
				gescannt_am: 1
			});
			offlineSync.init();
			await vi.waitFor(() => expect(offlineSync.wartendeAnzahl).toBe(1));

			netz.offline = true;
			vi.advanceTimersByTime(60000);
			await new Promise((fertig) => setTimeout(fertig, 20));
			expect(apiClient.post).not.toHaveBeenCalled();

			netz.offline = false;
			vi.advanceTimersByTime(60000);
			await vi.waitFor(() => expect(apiClient.post).toHaveBeenCalled());
		} finally {
			vi.useRealTimers();
		}
	});
});
