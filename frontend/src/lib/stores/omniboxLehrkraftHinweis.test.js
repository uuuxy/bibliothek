import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import 'fake-indexeddb/auto';

// Der Hinweis nach dem Scan eines Lehrerausweises beschreibt, was an der Theke passiert: Die
// Lehrkraft ist geladen, die folgenden Bücher gehen auf sie. Bis zum 16.09.2026 hieß er
// „Handapparat-Sitzung gestartet für Lehrer/in …" — ein Wort aus dem Code, nicht von der Theke.
//
// Seit Migration 125 liefert der Server auch für einen Kollegen `type: "student"` — ein
// gescannter Ausweis ist ein LESER, und seine Art steht an ihm. Der Hinweis hängt an der Art,
// nicht mehr an einem eigenen Antworttyp.
//
// Der Store meldet über seine eigene showToast → toastStore.addToast (omnibox.svelte.js), nicht
// über den Inventur-Store. Beobachtet wird deshalb addToast am echten toastStore.
vi.mock('../apiFetch.js', () => ({
	apiFetch: vi.fn(),
	apiClient: { post: vi.fn() }
}));
vi.mock('../audio.js', () => ({
	playSoundSuccess: vi.fn(),
	playSoundError: vi.fn(),
	playSuccessBeep: vi.fn(),
	playErrorBeep: vi.fn()
}));

import { apiClient } from '../apiFetch.js';
import { toastStore } from './toastStore.svelte.js';
import { omniboxStore } from './omnibox.svelte.js';

// Nach jedem Fall die Zeitgeber der Theke stoppen. `scanfeldWiederScharfstellen` plant
// einen Fokussprung über 50 ms, der `document` anfasst — endet die Datei vorher, baut
// Vitest jsdom ab, und der Rückruf reisst den GANZEN Lauf rot („Unhandled Errors:
// document is not defined"), obwohl jeder Test grün ist. Genau so stand die CI am
// 16.09.2026. Belegt in stores/omniboxZeitgeber.test.js.
afterEach(() => omniboxStore.stoppeZeitgeber());

describe('Omnibox: Lehrerausweis gescannt', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
		vi.clearAllMocks();
		omniboxStore.activeStudent = null;
		omniboxStore.queryVal = '';
	});

	it('meldet den geladenen Kollegen ohne Wörter aus dem Code', async () => {
		const addToast = vi.spyOn(toastStore, 'addToast');
		vi.mocked(apiClient.post).mockResolvedValueOnce(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({
					type: 'student',
					student: { id: 'l1', vorname: 'Karl', nachname: 'Lehmann', art: 'lehrkraft' }
				})
			})
		);
		omniboxStore.queryVal = 'L-4711';

		await omniboxStore.submitAction(new Event('submit'));

		expect(omniboxStore.activeStudent?.id).toBe('l1');
		const texte = addToast.mock.calls.map((aufruf) => String(aufruf[0]));
		const hinweis = texte.find((t) => t.includes('Karl Lehmann'));
		expect(hinweis, `kein Hinweis mit dem Namen, gemeldet: ${JSON.stringify(texte)}`).toBeTruthy();
		expect(hinweis).not.toContain('Handapparat');
	});

	// Ein Schüler bekommt KEINEN solchen Hinweis: Sein Profil steht groß daneben, und ein
	// zusätzlicher Zuruf bei jedem Scan wäre Lärm.
	it('meldet einen geladenen Schüler nicht eigens', async () => {
		const addToast = vi.spyOn(toastStore, 'addToast');
		vi.mocked(apiClient.post).mockResolvedValueOnce(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({
					type: 'student',
					student: { id: 's1', vorname: 'Mia', nachname: 'Muster', art: 'schueler' }
				})
			})
		);
		omniboxStore.queryVal = 'S-1';

		await omniboxStore.submitAction(new Event('submit'));

		expect(omniboxStore.activeStudent?.id).toBe('s1');
		const texte = addToast.mock.calls.map((aufruf) => String(aufruf[0]));
		expect(texte.find((t) => t.includes('Mia Muster'))).toBeFalsy();
	});
});
