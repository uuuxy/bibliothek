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

/** Einträge in Format 2, wie schnappschuss() sie im Omnibox-Store baut. */
let zaehler = 0;
/** @param {string} barcode @param {string} schuelerId */
const ausleihe = (barcode, schuelerId) => ({
	id: crypto.randomUUID(),
	art: /** @type {const} */ ('ausleihe'),
	barcode,
	schueler_id: schuelerId,
	lehrer_id: null,
	gescannt_am: ++zaehler
});
/** @param {string} barcode */
const rueckgabe = (barcode) => ({
	id: crypto.randomUUID(),
	art: /** @type {const} */ ('rueckgabe'),
	barcode,
	schueler_id: null,
	lehrer_id: null,
	gescannt_am: ++zaehler
});

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
		await enqueueOfflineAction(rueckgabe('B-1'));
		await enqueueOfflineAction(ausleihe('B-2', 'schueler-1'));

		const q = await loadQueue();
		expect(q).toHaveLength(2);
		expect(new Set(q.map((i) => i.id)).size).toBe(2);

		const checkout = q.find((i) => i.art === 'ausleihe');
		expect(checkout?.schueler_id).toBe('schueler-1');

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
		await enqueueOfflineAction(ausleihe('B-100', 'schueler-42'));

		vi.mocked(apiClient.post).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({
					results: [{ index: 0, success: true, data: { type: 'ausleihe' } }]
				})
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
		await enqueueOfflineAction(rueckgabe('B-KAPUTT'));

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
		await enqueueOfflineAction(rueckgabe('B-200'));

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
		await enqueueOfflineAction(rueckgabe('B-10234'));
		await enqueueOfflineAction(rueckgabe('B-10243')); // Etikett abgerieben, gibt es nicht
		// Die Antwort wird aus dem Payload gebaut: Die Reihenfolge in der Warteschlange
		// hängt am Zeitstempel, und zwei Einträge derselben Millisekunde sind nicht
		// geordnet. Ein Test, der die Reihenfolge rät, misst die falsche Zeile.
		vi.mocked(apiClient.post).mockImplementation(async (_pfad, payload) => {
			const results = /** @type {any[]} */ (payload).map((p, i) =>
				p.query === 'B-10243'
					? { index: i, success: false, status: 404, error: 'Barcode nicht gefunden' }
					: { index: i, success: true, status: 200, data: { type: 'rueckgabe' } }
			);
			return /** @type {any} */ ({ ok: true, json: async () => ({ results }) });
		});

		await offlineSync.startSync();

		expect(await loadQueue(), 'die Warteschlange ist geleert').toHaveLength(0);
		const meldungen = vi.mocked(showToast).mock.calls.map((c) => String(c[0]));
		expect(meldungen.join(' | ')).toContain('B-10243');
		expect(meldungen.join(' | '), 'der angenommene Scan gehört nicht in die Warnung').not.toContain(
			'B-10234'
		);
	});

	it('schweigt, wenn alles angenommen wurde (Gegenprobe)', async () => {
		await enqueueOfflineAction(rueckgabe('B-10234'));
		vi.mocked(apiClient.post).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({
					results: [{ index: 0, success: true, status: 200, data: { type: 'rueckgabe' } }]
				})
			})
		);
		await offlineSync.startSync();
		expect(vi.mocked(showToast)).not.toHaveBeenCalled();
	});
});

// Eine Ausleihe an eine Lehrkraft schickt active_teacher_id (OFFEN.md 2.2, Commit 4). Der
// Stapel-Endpunkt kennt das Feld seit jeher; der Payload-Bauer schickte es nie.
describe('offlineSync: Handapparat trägt die Lehrkraft mit', () => {
	beforeEach(async () => {
		await clearQueue();
		vi.clearAllMocks();
	});

	it('schickt active_teacher_id für eine offline gespeicherte Handapparat-Ausleihe', async () => {
		await enqueueOfflineAction({
			id: crypto.randomUUID(),
			art: 'ausleihe',
			barcode: 'B-10236',
			schueler_id: null,
			lehrer_id: 'lehrkraft-3',
			gescannt_am: ++zaehler
		});
		vi.mocked(apiClient.post).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({
					results: [{ index: 0, success: true, status: 200, data: { type: 'ausleihe' } }]
				})
			})
		);

		await offlineSync.startSync();

		const payload = vi.mocked(apiClient.post).mock.calls[0][1];
		expect(payload[0].active_teacher_id, 'ohne Lehrkraft bucht der Server eine Rückgabe').toBe(
			'lehrkraft-3'
		);
		expect(payload[0].active_student_id).toBeUndefined();
	});
});

// Erledigt ist nur, was der Server WIE GESCANNT gebucht hat (OFFEN.md 2.2, Commit 6). Bis
// zum 15.09.2026 galt als erledigt: success, jeder 4xx außer 429 und „Server nannte den Index
// nicht"; der Antworttyp wurde nie mit der Absicht verglichen, und ein liegengebliebener
// Eintrag (5xx, 429) wurde ohne Pause sofort erneut gesendet.
describe('offlineSync: erledigt nur, was wie gescannt gebucht wurde', () => {
	beforeEach(async () => {
		await clearQueue();
		vi.clearAllMocks();
	});

	it('Schweigen des Servers (kein Ergebnis zum Index) ist kein Erfolg', async () => {
		await enqueueOfflineAction(rueckgabe('B-10234'));
		vi.mocked(apiClient.post).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ({ results: [] }) })
		);
		await offlineSync.startSync();
		expect(await loadQueue(), 'der Eintrag bleibt, bis der Server ihn beantwortet').toHaveLength(1);
		expect(apiClient.post, 'keine Endlosschleife: die Runde endet').toHaveBeenCalledTimes(1);
	});

	it('Ausleihe gescannt, Rückgabe gebucht: ausgebucht UND gemeldet, mit Barcode und beiden Typen', async () => {
		await enqueueOfflineAction(ausleihe('B-10234', 'schueler-7'));
		vi.mocked(apiClient.post).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({
					results: [{ index: 0, success: true, status: 200, data: { type: 'rueckgabe' } }]
				})
			})
		);
		await offlineSync.startSync();
		expect(await loadQueue(), 'blockiert nicht (erst Stufe 3 mit der neuen Tür)').toHaveLength(0);
		const meldungen = vi
			.mocked(showToast)
			.mock.calls.map((c) => String(c[0]))
			.join(' | ');
		expect(meldungen).toContain('B-10234');
		expect(meldungen).toMatch(/Ausleihe/);
		expect(meldungen).toMatch(/Rückgabe/);
	});

	it('wie gescannt gebucht: still ausgebucht', async () => {
		await enqueueOfflineAction(ausleihe('B-10234', 'schueler-7'));
		vi.mocked(apiClient.post).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({
					results: [{ index: 0, success: true, status: 200, data: { type: 'ausleihe' } }]
				})
			})
		);
		await offlineSync.startSync();
		expect(await loadQueue()).toHaveLength(0);
		expect(showToast).not.toHaveBeenCalled();
	});

	it('5xx bleibt liegen, und die Runde endet, statt sofort erneut zu senden', async () => {
		await enqueueOfflineAction(rueckgabe('B-10234'));
		vi.mocked(apiClient.post)
			.mockResolvedValueOnce(
				/** @type {any} */ ({
					ok: true,
					json: async () => ({ results: [{ index: 0, success: false, status: 503 }] })
				})
			)
			.mockResolvedValue(
				/** @type {any} */ ({
					ok: true,
					json: async () => ({
						results: [{ index: 0, success: true, status: 200, data: { type: 'rueckgabe' } }]
					})
				})
			);
		await offlineSync.startSync();
		expect(apiClient.post, 'ein Versuch je Runde').toHaveBeenCalledTimes(1);
		expect(await loadQueue(), 'der Eintrag wartet auf die nächste Runde').toHaveLength(1);
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
		await enqueueOfflineAction(ausleihe('B-10234', 'schueler-7'));
		await enqueueOfflineAction(rueckgabe('B-10235'));
		vi.mocked(apiClient.post).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({
					results: [
						{ index: 0, success: true, status: 200, data: { type: 'ausleihe' } },
						{ index: 1, success: true, status: 200, data: { type: 'rueckgabe' } }
					]
				})
			})
		);

		await offlineSync.startSync();

		const payload = vi.mocked(apiClient.post).mock.calls[0][1];
		const mitSchueler = payload.find((/** @type {any} */ p) => p.query === 'B-10234');
		const ohneSchueler = payload.find((/** @type {any} */ p) => p.query === 'B-10235');
		expect(mitSchueler.active_student_id, 'ohne Schüler bucht der Server eine Rückgabe').toBe(
			'schueler-7'
		);
		expect(ohneSchueler.active_student_id).toBeUndefined();
	});
});
