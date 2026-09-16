import { describe, it, expect, vi, beforeEach } from 'vitest';
import 'fake-indexeddb/auto';

// Ein Offline-Scan bei geladenem Schüler ist eine AUSLEIHE (Rasterdurchgang 06.09.2026).
//
// Bis dahin legte die Omnibox JEDEN Offline-Scan als „checkin" ab. Der Payload-Bauer des
// Syncs schickt `active_leser_id` nur bei „checkout" — einem Typ, den niemand je
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
		// Ein ECHTER Versandfehler, wie er im WLAN-Loch entsteht. Bis zum 15.09.2026 lieferte
		// der Mock undefined, und der TypeError kam aus `res.ok` — der Test maß den Fehler aus
		// Commit 2 (Auswertung im selben catch), nicht den Netzausfall.
		vi.mocked(apiClient.post).mockRejectedValue(new TypeError('Failed to fetch'));
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
		expect(q[0].leser_id, 'die Person vom Scan').toBe('schueler-7');
		expect(q[0].art, 'die Absicht vom Scan').toBe('ausleihe');
		expect(q[0].gescannt_am).toBeGreaterThan(0);
	});

	// Nur ein gescheiterter VERSAND gehört in die Warteschlange (OFFEN.md 2.2, Commit 2). Bis
	// dahin lag verarbeiteAktionsErgebnis im selben catch: Ein TypeError aus der Auswertung
	// einer gelungenen 200-Antwort wurde eingereiht — mit demselben Idempotenz-Schlüssel, den
	// der Server schon kannte; nach Ablauf des Caches (24 h) wäre neu gebucht worden.
	it('reiht eine gelungene, aber unauswertbare Antwort NICHT ein', async () => {
		vi.mocked(apiClient.post).mockResolvedValueOnce(
			// Eine Fremdrückgabe ohne Buch: Die Auswertung greift auf data.book.titel zu.
			/** @type {any} */ ({
				ok: true,
				json: async () => ({ type: 'rueckgabe', fremdrueckgabe: true })
			})
		);
		omniboxStore.queryVal = 'B-10234';
		await omniboxStore.submitAction(new Event('submit'));
		expect(await loadQueue(), 'eine Antwort kam an — das ist kein Netzausfall').toHaveLength(0);
		expect(omniboxStore.errorMessage, 'der Fehler wird gezeigt, nicht versteckt').toMatch(/Fehler/);
	});

	// „Buch zurückgeben" in der Akte ist eine RÜCKGABE — auch offline (OFFEN.md 2.2, Commit 3).
	// Online entscheidet der Server (das Buch liegt beim geladenen Schüler → Rückgabe); offline
	// entschied bis zum 15.09.2026 nur „Schüler geladen?", und der Knopf reihte eine Ausleihe
	// ein. Beim Nachbuchen wäre daraus eine Rückgabe geworden (Buch schon bei ihm) — beim
	// Doppelklick aber die zweite Ausleihe eine echte, und das Kind hätte das Buch wieder.
	it('„Buch zurückgeben" im Profil ist offline eine Rückgabe, auch beim Doppelklick', async () => {
		omniboxStore.activeStudent = { id: 'schueler-7', vorname: 'Anna', nachname: 'Müller' };
		await omniboxStore.gibZurueck('B-10234');
		await omniboxStore.gibZurueck('B-10234');
		const q = await loadQueue();
		expect(q.map((e) => e.art)).toEqual(['rueckgabe', 'rueckgabe']);
		expect(q.map((e) => e.leser_id)).toEqual(['schueler-7', 'schueler-7']);
	});

	// Mit geladenem Kollegen ist ein Offline-Buch eine Ausleihe an ihn (OFFEN.md 2.2,
	// Commit 4; Entscheidung (b) vom 13.09.2026). Bis dahin entschied nur activeStudent, und
	// das Buch ging mit geladener Lehrkraft als Rückgabe in die Warteschlange.
	//
	// Seit Migration 125 steht ein Kollege in demselben Platz wie ein Schüler — der Eintrag
	// trägt seine Leser-Kennung im selben Feld.
	it('reiht mit geladenem Kollegen eine Ausleihe an ihn ein', async () => {
		omniboxStore.activeStudent = {
			id: 'leser-3',
			vorname: 'Karl',
			nachname: 'Kraft',
			art: 'lehrkraft'
		};
		omniboxStore.queryVal = 'B-10234';
		await omniboxStore.submitAction(new Event('submit'));
		const q = await loadQueue();
		expect(q).toHaveLength(1);
		expect(q[0].art, 'eine Ausleihe, keine Rückgabe').toBe('ausleihe');
		expect(q[0].leser_id).toBe('leser-3');
		omniboxStore.activeStudent = null;
	});

	it('reiht mit geladenem Schüler eine Ausleihe ein, ohne ihn eine Rückgabe', async () => {
		omniboxStore.activeStudent = { id: 'schueler-7', vorname: 'Anna', nachname: 'Müller' };
		omniboxStore.queryVal = 'B-10234';
		await omniboxStore.submitAction(new Event('submit'));

		let q = await loadQueue();
		expect(q, 'der Scan wurde offline gespeichert').toHaveLength(1);
		expect(q[0].art, 'mit Schüler an der Theke ist der Scan eine Ausleihe').toBe('ausleihe');
		expect(q[0].leser_id).toBe('schueler-7');

		// Gegenprobe: ohne Schüler bleibt es eine Rückgabe.
		await leere();
		omniboxStore.activeStudent = null;
		omniboxStore.queryVal = 'B-10235';
		await omniboxStore.submitAction(new Event('submit'));
		q = await loadQueue();
		expect(q).toHaveLength(1);
		expect(q[0].art).toBe('rueckgabe');
		expect(q[0].leser_id).toBeNull();
	});
});
