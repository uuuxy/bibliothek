import { describe, it, expect, vi, beforeEach } from 'vitest';
import 'fake-indexeddb/auto';

// Der Hinweis nach dem Scan eines Lehrerausweises beschreibt, was an der Theke passiert: Die
// Lehrkraft ist geladen, die folgenden Bücher gehen auf sie. Bis zum 16.09.2026 hieß er
// „Handapparat-Sitzung gestartet für Lehrer/in …" — ein Wort aus dem Code, nicht von der Theke.
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

describe('Omnibox: Lehrerausweis gescannt', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
		vi.clearAllMocks();
		omniboxStore.activeStudent = null;
		omniboxStore.activeTeacher = null;
		omniboxStore.queryVal = '';
	});

	it('meldet die geladene Lehrkraft ohne Wörter aus dem Code', async () => {
		const addToast = vi.spyOn(toastStore, 'addToast');
		vi.mocked(apiClient.post).mockResolvedValueOnce(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({
					type: 'teacher',
					teacher: { id: 'l1', vorname: 'Karl', nachname: 'Lehmann' }
				})
			})
		);
		omniboxStore.queryVal = 'L-4711';

		await omniboxStore.submitAction(new Event('submit'));

		expect(omniboxStore.activeTeacher?.id).toBe('l1');
		const texte = addToast.mock.calls.map((aufruf) => String(aufruf[0]));
		const hinweis = texte.find((t) => t.includes('Karl Lehmann'));
		expect(hinweis, `kein Hinweis mit dem Namen, gemeldet: ${JSON.stringify(texte)}`).toBeTruthy();
		expect(hinweis).toContain('Lehrkraft');
		expect(hinweis).not.toContain('Handapparat');
	});
});
