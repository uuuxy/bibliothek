import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Ein Fehler der Warteschlange heißt nicht „gespeichert" (OFFEN.md 2.2, Commit 5). Bis zum
// 15.09.2026 schluckten enqueueOfflineAction, ladeWarteschlange und dequeueOfflineAction jeden
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
	spieleErfolgston: vi.fn(),
	spieleFehlerton: vi.fn(),
	spieleErfolgspiep: vi.fn(),
	spieleFehlerpiep: vi.fn()
}));
vi.mock('../../inventur/lib/store.svelte.js', () => ({ melde: vi.fn() }));

import { enqueueOfflineAction, ladeWarteschlange } from '../offlineWarteschlange.js';
import { omniboxStore } from './omnibox.svelte.js';

// Nach jedem Fall die Zeitgeber der Theke stoppen. `scanfeldWiederScharfstellen` plant
// einen Fokussprung über 50 ms, der `document` anfasst — endet die Datei vorher, baut
// Vitest jsdom ab, und der Rückruf reisst den GANZEN Lauf rot („Unhandled Errors:
// document is not defined"), obwohl jeder Test grün ist. Genau so stand die CI am
// 16.09.2026. Belegt in stores/omniboxZeitgeber.test.js.
afterEach(() => omniboxStore.stoppeZeitgeber());

import { offlineSync } from './offlineSync.svelte.js';
import { snackbars } from './snackbars.svelte.js';
import { spieleErfolgston, spieleFehlerton } from '../audio.js';

describe('Warteschlange nicht erreichbar', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		vi.spyOn(console, 'error').mockImplementation(() => {});
		vi.spyOn(console, 'warn').mockImplementation(() => {});
		omniboxStore.aktiverLeser = null;
		omniboxStore.activeTeacher = null;
	});

	it('enqueue und ladeWarteschlange werfen, statt still zu schweigen', async () => {
		await expect(
			enqueueOfflineAction({
				id: 'k1',
				art: 'rueckgabe',
				barcode: 'B-1',
				leser_id: null,
				gescannt_am: 1
			})
		).rejects.toThrow(/IndexedDB/);
		await expect(ladeWarteschlange()).rejects.toThrow(/IndexedDB/);
	});

	it('die Theke meldet „NICHT gespeichert" mit Fehlerton, kein Erfolg', async () => {
		omniboxStore.eingabe = 'B-10234';
		await omniboxStore.fuehreAktionAus(new Event('submit'));
		expect(omniboxStore.fehler).toMatch(/NICHT gespeichert/);
		expect(spieleFehlerton).toHaveBeenCalled();
		expect(spieleErfolgston).not.toHaveBeenCalled();
		const erfolgsToast = snackbars.toasts.find((t) => /gespeichert\.$/.test(t.message));
		expect(erfolgsToast, 'kein „Offline: gespeichert"').toBeUndefined();
	});

	it('das Band zeigt „nicht lesbar" statt 0', async () => {
		await offlineSync.zaehleNeu();
		expect(offlineSync.warteschlangeFehler).toBe(true);
	});
});
