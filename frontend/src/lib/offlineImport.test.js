import { describe, it, expect, vi, beforeEach } from 'vitest';

// Die Warteschlangen-ID IST der Idempotenz-Schlüssel: baueBatchPayload schickt sie
// als idempotency_key, und der Server beantwortet einen bekannten Schlüssel aus dem
// Cache statt die Aktion erneut auszuführen (api/action.go).
//
// Beim Einspielen einer Sicherung muss diese ID deshalb mitwandern. Ohne sie vergibt
// enqueueOfflineAction eine frische UUID — und dieselbe Datei zweimal eingespielt
// führte jede Rückgabe ZWEIMAL aus. Bei zehn Kiosk-Rechnern mit gemeinsamem
// Sicherungsordner ist doppeltes Einspielen der Normalfall: zwei Admins, oder einer,
// der nicht mehr weiß, ob er es schon getan hat.

const enqueueSpy = vi.fn();

vi.mock('./offlineQueue.js', () => ({
	enqueueOfflineAction: (...args) => {
		enqueueSpy(...args);
		return Promise.resolve();
	},
	loadQueue: () => Promise.resolve([]),
	dequeueOfflineAction: () => Promise.resolve(),
	getQueueCount: () => Promise.resolve(0)
}));

vi.mock('./audio.js', () => ({ playSoundSuccess: () => {}, playSoundError: () => {} }));
vi.mock('./apiFetch.js', () => ({
	apiClient: { post: () => Promise.resolve({ ok: true, json: () => ({ results: [] }) }) },
	apiFetch: () => Promise.resolve({ ok: true })
}));

/** Sicherungsdatei, wie exportQueueAsJSON sie schreibt: die rohen Queue-Objekte. */
function sicherungsdatei(items) {
	// Absichtlich nur die zwei Felder, die der Import liest (name, text) — ein echtes
	// File nachzubauen braucht acht weitere, die hier nichts pruefen wuerden.
	return /** @type {File} */ (
		/** @type {unknown} */ ({
			name: 'offline_scans_backup_2026-07-29.json',
			text: () => Promise.resolve(JSON.stringify(items))
		})
	);
}

describe('Einspielen einer Offline-Sicherung', () => {
	beforeEach(() => enqueueSpy.mockClear());

	it('reicht die Idempotenz-ID aus der Datei durch', async () => {
		const { offlineSync } = await import('./stores/offlineSync.svelte.js');

		const anzahl = await offlineSync.importQueueFromJSON(
			sicherungsdatei([
				{
					id: '11111111-2222-3333-4444-555555555555',
					action_type: 'checkin',
					barcode_id: 'B-4711',
					schueler_id: null,
					timestamp: 1
				}
			])
		);

		expect(anzahl).toBe(1);
		expect(enqueueSpy).toHaveBeenCalledWith(
			'checkin',
			'B-4711',
			null,
			'11111111-2222-3333-4444-555555555555'
		);
	});

	it('vergibt beim zweiten Einspielen derselben Datei KEINE neue ID', async () => {
		const { offlineSync } = await import('./stores/offlineSync.svelte.js');
		const datei = () =>
			sicherungsdatei([
				{ id: 'stabile-id', action_type: 'checkin', barcode_id: 'B-4711', timestamp: 1 }
			]);

		await offlineSync.importQueueFromJSON(datei());
		await offlineSync.importQueueFromJSON(datei());

		const idsBeiderLaeufe = enqueueSpy.mock.calls.map((c) => c[3]);
		expect(idsBeiderLaeufe).toEqual(['stabile-id', 'stabile-id']);
	});

	it('überspringt Einträge ohne Barcode, statt sie kaputt weiterzureichen', async () => {
		const { offlineSync } = await import('./stores/offlineSync.svelte.js');

		const anzahl = await offlineSync.importQueueFromJSON(
			sicherungsdatei([
				{ id: 'a', action_type: 'checkin', barcode_id: 'B-1', timestamp: 1 },
				{ id: 'b', action_type: 'checkin', timestamp: 2 },
				{ id: 'c', barcode_id: 'B-3', timestamp: 3 }
			])
		);

		expect(anzahl).toBe(1);
		expect(enqueueSpy).toHaveBeenCalledTimes(1);
	});
});
