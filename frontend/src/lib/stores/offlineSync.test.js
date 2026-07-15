import 'fake-indexeddb/auto';
import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../apiFetch.js', () => ({
	apiClient: { post: vi.fn() }
}));
vi.mock('../audio.js', () => ({
	playSoundSuccess: vi.fn(),
	playSoundError: vi.fn(),
	playSuccessBeep: vi.fn(),
	playErrorBeep: vi.fn()
}));

import { apiClient } from '../apiFetch.js';
import { enqueueOfflineAction, loadQueue, dequeueOfflineAction } from '../offlineQueue.js';
import { offlineSync } from './offlineSync.svelte.js';

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
