import { describe, it, expect, vi, beforeEach } from 'vitest';

// Ein Fehler der Warteschlange heißt nicht „gespeichert" (OFFEN.md 2.2, Commit 5). Bis zum
// 15.09.2026 schluckten enqueueOfflineAction, loadQueue und dequeueOfflineAction jeden
// IndexedDB-Fehler (privates Fenster, gesperrte Website-Daten, voller Speicher): Die Theke
// meldete „Offline: gespeichert" mit Erfolgston, das Band zeigte 0 — und der Scan war weg.
vi.mock('idb', () => ({
	openDB: vi.fn(async () => {
		throw new Error('IndexedDB blockiert');
	})
}));
vi.mock('../apiFetch.js', () => ({
	apiFetch: vi.fn(),
	apiClient: {
		post: vi.fn(async () => {
			throw new TypeError('Failed to fetch');
		})
	}
}));
vi.mock('../audio.js', () => ({
	playSoundSuccess: vi.fn(),
	playSoundError: vi.fn(),
	playSuccessBeep: vi.fn(),
	playErrorBeep: vi.fn()
}));
vi.mock('../../inventur/lib/store.svelte.js', () => ({ showToast: vi.fn() }));

import { enqueueOfflineAction, loadQueue } from '../offlineQueue.js';
import { omniboxStore } from './omnibox.svelte.js';
import { offlineSync } from './offlineSync.svelte.js';
import { toastStore } from './toastStore.svelte.js';
import { playSoundSuccess, playSoundError } from '../audio.js';

describe('Warteschlange nicht erreichbar', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		vi.spyOn(console, 'error').mockImplementation(() => {});
		vi.spyOn(console, 'warn').mockImplementation(() => {});
		omniboxStore.activeStudent = null;
		omniboxStore.activeTeacher = null;
	});

	it('enqueue und loadQueue werfen, statt still zu schweigen', async () => {
		await expect(
			enqueueOfflineAction({
				id: 'k1',
				art: 'rueckgabe',
				barcode: 'B-1',
				schueler_id: null,
				lehrer_id: null,
				gescannt_am: 1
			})
		).rejects.toThrow(/IndexedDB/);
		await expect(loadQueue()).rejects.toThrow(/IndexedDB/);
	});

	it('die Theke meldet „NICHT gespeichert" mit Fehlerton, kein Erfolg', async () => {
		omniboxStore.queryVal = 'B-10234';
		await omniboxStore.submitAction(new Event('submit'));
		expect(omniboxStore.errorMessage).toMatch(/NICHT gespeichert/);
		expect(playSoundError).toHaveBeenCalled();
		expect(playSoundSuccess).not.toHaveBeenCalled();
		const erfolgsToast = toastStore.toasts.find((t) => /gespeichert\.$/.test(t.message));
		expect(erfolgsToast, 'kein „Offline: gespeichert"').toBeUndefined();
	});

	it('das Band zeigt „nicht lesbar" statt 0', async () => {
		await offlineSync.updateCount();
		expect(offlineSync.warteschlangeFehler).toBe(true);
	});
});
