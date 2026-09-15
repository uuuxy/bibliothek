import { describe, it, expect, vi, beforeEach } from 'vitest';
import 'fake-indexeddb/auto';

// Ein Offline-Scan bei geladenem Schüler ist eine AUSLEIHE (Rasterdurchgang 06.09.2026).
//
// Bis dahin legte die Omnibox JEDEN Offline-Scan als „checkin" ab. Der Payload-Bauer des
// Syncs schickt `active_student_id` nur bei „checkout" — einem Typ, den niemand je
// einreihte; der Zweig war unerreichbar. Der Server las das Schweigen als Rückgabe: Das
// Buch war schon draußen, die Rückgabe scheiterte mit 400, und der Eintrag flog aus der
// Warteschlange. Das Kind hatte das Buch, das System sagte „verfügbar".
//
// Geprüft wird der ECHTE Weg: submitAction mit einem Netzwerkfehler, so wie er im
// WLAN-Loch entsteht.
vi.mock('../apiFetch.js', () => ({
	apiFetch: vi.fn(async () => {
		throw new TypeError('Failed to fetch');
	}),
	apiClient: { post: vi.fn() }
}));
vi.mock('../audio.js', () => ({
	playSoundSuccess: vi.fn(),
	playSoundError: vi.fn(),
	playSuccessBeep: vi.fn(),
	playErrorBeep: vi.fn()
}));
vi.mock('../../inventur/lib/store.svelte.js', () => ({ showToast: vi.fn() }));

import { apiClient } from '../apiFetch.js';
import { omniboxStore } from './omnibox.svelte.js';
import { loadQueue, dequeueOfflineAction } from '../offlineQueue.js';

async function leere() {
	for (const item of await loadQueue()) await dequeueOfflineAction(item.id);
}

describe('Omnibox offline', () => {
	beforeEach(async () => {
		await leere();
		vi.clearAllMocks();
		omniboxStore.activeStudent = null;
		omniboxStore.queryVal = '';
	});

	// Der Eintrag trägt die Person VOM SCAN, nicht die vom Zeitpunkt des Scheiterns
	// (OFFEN.md 2.2, Commit 1). Bis dahin las speichereOfflineAktion Person und Absicht erst
	// nach der hängenden Anfrage (Timeout 10 s): Escape oder „Theke leeren" in dieser Zeit,
	// und das Buch ging als Rückgabe ohne Person in die Warteschlange.
	it('hält Person und Absicht beim Scan fest, nicht erst beim Scheitern', async () => {
		/** @type {(e: Error) => void} */
		let scheitern = () => {};
		vi.mocked(apiClient.post).mockImplementationOnce(
			() => new Promise((_, rej) => (scheitern = /** @type {any} */ (rej)))
		);
		omniboxStore.activeStudent = { id: 'schueler-7', vorname: 'Anna', nachname: 'Müller' };
		omniboxStore.queryVal = 'B-10234';
		const laeuft = omniboxStore.submitAction(new Event('submit'));
		// Während die Anfrage hängt: Escape an der Theke.
		omniboxStore.activeStudent = null;
		scheitern(new Error('Netzwerk-Timeout: Die Anfrage hat zu lange gedauert.'));
		await laeuft;

		const q = await loadQueue();
		expect(q).toHaveLength(1);
		expect(q[0].schueler_id, 'die Person vom Scan').toBe('schueler-7');
		expect(q[0].art, 'die Absicht vom Scan').toBe('ausleihe');
		expect(q[0].gescannt_am).toBeGreaterThan(0);
	});

	it('reiht mit geladenem Schüler eine Ausleihe ein, ohne ihn eine Rückgabe', async () => {
		omniboxStore.activeStudent = { id: 'schueler-7', vorname: 'Anna', nachname: 'Müller' };
		omniboxStore.queryVal = 'B-10234';
		await omniboxStore.submitAction(new Event('submit'));

		let q = await loadQueue();
		expect(q, 'der Scan wurde offline gespeichert').toHaveLength(1);
		expect(q[0].art, 'mit Schüler an der Theke ist der Scan eine Ausleihe').toBe('ausleihe');
		expect(q[0].schueler_id).toBe('schueler-7');

		// Gegenprobe: ohne Schüler bleibt es eine Rückgabe.
		await leere();
		omniboxStore.activeStudent = null;
		omniboxStore.queryVal = 'B-10235';
		await omniboxStore.submitAction(new Event('submit'));
		q = await loadQueue();
		expect(q).toHaveLength(1);
		expect(q[0].art).toBe('rueckgabe');
		expect(q[0].schueler_id).toBeNull();
	});
});
