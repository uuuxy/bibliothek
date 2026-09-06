import 'fake-indexeddb/auto';
import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../apiFetch.js', () => ({
	apiClient: { post: vi.fn() }
}));
vi.mock('../../inventur/lib/store.svelte.js', () => ({ showToast: vi.fn() }));
vi.mock('../audio.js', () => ({
	playSoundSuccess: vi.fn(),
	playSoundError: vi.fn(),
	playSuccessBeep: vi.fn(),
	playErrorBeep: vi.fn()
}));

import { apiClient } from '../apiFetch.js';
import { enqueueOfflineAction, loadQueue, dequeueOfflineAction } from '../offlineQueue.js';
import { offlineSync } from './offlineSync.svelte.js';
import { showToast } from '../../inventur/lib/store.svelte.js';

async function clearQueue() {
	for (const item of await loadQueue()) {
		await dequeueOfflineAction(item.id);
	}
}

// Die Offline-Queue ist der riskante Teil des PWA-Verhaltens — bewusst als
// Unit-Test statt E2E (Service-Worker-Offline-Simulation ist CI-flaky).
describe('offlineQueue', () => {
	beforeEach(async () => {
		await clearQueue();
		vi.clearAllMocks();
	});

	it('reiht Aktionen mit eindeutigen Idempotenz-Keys ein und entfernt gezielt', async () => {
		await enqueueOfflineAction('checkin', 'B-1');
		await enqueueOfflineAction('checkout', 'B-2', 'schueler-1');

		const q = await loadQueue();
		expect(q).toHaveLength(2);
		expect(new Set(q.map((i) => i.id)).size).toBe(2);

		const checkout = q.find((i) => i.action_type === 'checkout');
		expect(checkout.schueler_id).toBe('schueler-1');

		await dequeueOfflineAction(q[0].id);
		expect(await loadQueue()).toHaveLength(1);
	});
});

describe('offlineSync.startSync', () => {
	beforeEach(async () => {
		await clearQueue();
		vi.clearAllMocks();
	});

	it('synct die Queue als Batch mit Idempotenz-Keys und leert sie bei Erfolg', async () => {
		await enqueueOfflineAction('checkout', 'B-100', 'schueler-42');

		vi.mocked(apiClient.post).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({ results: [{ index: 0, success: true }] })
			})
		);

		await offlineSync.startSync();

		expect(apiClient.post).toHaveBeenCalledWith('/api/action/batch', [
			expect.objectContaining({
				query: 'B-100',
				active_student_id: 'schueler-42',
				idempotency_key: expect.any(String)
			})
		]);
		expect(await loadQueue()).toHaveLength(0);
		expect(offlineSync.pendingCount).toBe(0);
	});

	it('wirft dauerhaft abgelehnte Aktionen (4xx) aus der Queue statt endlos zu hängen', async () => {
		await enqueueOfflineAction('checkin', 'B-KAPUTT');

		vi.mocked(apiClient.post).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({ results: [{ index: 0, success: false, status: 404 }] })
			})
		);

		await offlineSync.startSync();
		expect(await loadQueue()).toHaveLength(0);
	});

	it('behält die Queue, wenn der Batch-Request selbst scheitert (z. B. 502)', async () => {
		await enqueueOfflineAction('checkin', 'B-200');

		vi.mocked(apiClient.post).mockResolvedValue(/** @type {any} */ ({ ok: false, status: 502 }));

		await offlineSync.startSync();
		expect(await loadQueue()).toHaveLength(1);
	});
});

// Was der Server ABLEHNT, darf nicht spurlos verschwinden (Rasterdurchgang 06.09.2026).
//
// Dass ein 4xx nicht ewig wiederholt wird, ist richtig — der Server hat fachlich
// entschieden. Falsch war, dass es niemand erfuhr: Der Eintrag flog aus der
// Warteschlange, der Zähler ging auf 0, und der Erfolgston lief. Eine Klasse gibt 18
// Bücher offline zurück, vier werden abgelehnt (abgeriebenes Etikett, gesperrter Schüler,
// falscher Zustand) — die Ausleihen bleiben offen und laufen ins Mahnwesen.
describe('offlineSync: abgelehnte Vorgänge', () => {
	beforeEach(async () => {
		await clearQueue();
		vi.clearAllMocks();
	});

	it('meldet abgelehnte Scans mit Barcode, statt sie still zu löschen', async () => {
		await enqueueOfflineAction('checkin', 'B-10234');
		await enqueueOfflineAction('checkin', 'B-10243'); // Etikett abgerieben, gibt es nicht
		// Die Antwort wird aus dem Payload gebaut: Die Reihenfolge in der Warteschlange
		// hängt am Zeitstempel, und zwei Einträge derselben Millisekunde sind nicht
		// geordnet. Ein Test, der die Reihenfolge rät, misst die falsche Zeile.
		vi.mocked(apiClient.post).mockImplementation(
			async (/** @type {any} */ _pfad, /** @type {any} */ payload) => ({
				ok: true,
				json: async () => ({
					results: payload.map((/** @type {any} */ p, /** @type {number} */ i) =>
						p.query === 'B-10243'
							? { index: i, success: false, status: 404, error: 'Barcode nicht gefunden' }
							: { index: i, success: true, status: 200 }
					)
				})
			})
		);

		await offlineSync.startSync();

		expect(await loadQueue(), 'die Warteschlange ist geleert').toHaveLength(0);
		const meldungen = vi.mocked(showToast).mock.calls.map((c) => String(c[0]));
		expect(meldungen.join(' | ')).toContain('B-10243');
		expect(meldungen.join(' | '), 'der angenommene Scan gehört nicht in die Warnung').not.toContain(
			'B-10234'
		);
	});

	it('schweigt, wenn alles angenommen wurde (Gegenprobe)', async () => {
		await enqueueOfflineAction('checkin', 'B-10234');
		vi.mocked(apiClient.post).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({ results: [{ index: 0, success: true, status: 200 }] })
			})
		);
		await offlineSync.startSync();
		expect(vi.mocked(showToast)).not.toHaveBeenCalled();
	});
});

// Der Offline-Scan mit geladenem Schüler ist eine AUSLEIHE, keine Rückgabe
// (Rasterdurchgang 06.09.2026). Bis dahin legte die Omnibox jeden Offline-Scan als
// „checkin" ab, und der Payload-Bauer schickte `active_student_id` nur bei „checkout" —
// einem Typ, den niemand je einreihte. Der Server las das Schweigen als Rückgabe: Das
// Buch war schon draußen, die Rückgabe scheiterte, der Eintrag flog aus der Warteschlange.
describe('offlineSync: Ausleihe trägt den Schüler mit', () => {
	beforeEach(async () => {
		await clearQueue();
		vi.clearAllMocks();
	});

	it('schickt active_student_id für eine offline gespeicherte Ausleihe', async () => {
		await enqueueOfflineAction('checkout', 'B-10234', 'schueler-7');
		await enqueueOfflineAction('checkin', 'B-10235');
		vi.mocked(apiClient.post).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({
					results: [
						{ index: 0, success: true, status: 200 },
						{ index: 1, success: true, status: 200 }
					]
				})
			})
		);

		await offlineSync.startSync();

		const payload = vi.mocked(apiClient.post).mock.calls[0][1];
		const ausleihe = payload.find((/** @type {any} */ p) => p.query === 'B-10234');
		const rueckgabe = payload.find((/** @type {any} */ p) => p.query === 'B-10235');
		expect(ausleihe.active_student_id, 'ohne Schüler bucht der Server eine Rückgabe').toBe(
			'schueler-7'
		);
		expect(rueckgabe.active_student_id).toBeUndefined();
	});
});
