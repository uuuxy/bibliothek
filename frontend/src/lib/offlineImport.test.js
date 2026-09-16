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

vi.mock('./offlineQueue.js', async (importOriginal) => ({
	// normalisiereEintrag bleibt echt: Der Import übersetzt damit beide Formate.
	.../** @type {any} */ (await importOriginal()),
	enqueueOfflineAction: (...args) => {
		enqueueSpy(...args);
		return Promise.resolve();
	},
	loadQueue: () => Promise.resolve([]),
	dequeueOfflineAction: () => Promise.resolve()
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
					leser_id: null,
					timestamp: 1
				}
			])
		);

		expect(anzahl).toBe(1);
		// Format 1 aus der Datei kommt als Format-2-Objekt in die Warteschlange (15.09.2026).
		expect(enqueueSpy).toHaveBeenCalledWith({
			id: '11111111-2222-3333-4444-555555555555',
			art: 'rueckgabe',
			barcode: 'B-4711',
			leser_id: null,
			gescannt_am: 1
		});
	});

	it('liest eine Sicherung in Format 2 (Absicht, Person, Scan-Zeitpunkt) unverändert', async () => {
		const { offlineSync } = await import('./stores/offlineSync.svelte.js');
		const eintrag = {
			id: 'f2-1',
			art: 'ausleihe',
			barcode: 'B-4712',
			leser_id: 'schueler-7',
			gescannt_am: 1757900000000
		};
		await offlineSync.importQueueFromJSON(sicherungsdatei([eintrag]));
		expect(enqueueSpy).toHaveBeenCalledWith(eintrag);
	});

	it('vergibt beim zweiten Einspielen derselben Datei KEINE neue ID', async () => {
		const { offlineSync } = await import('./stores/offlineSync.svelte.js');
		const datei = () =>
			sicherungsdatei([
				{ id: 'stabile-id', action_type: 'checkin', barcode_id: 'B-4711', timestamp: 1 }
			]);

		await offlineSync.importQueueFromJSON(datei());
		await offlineSync.importQueueFromJSON(datei());

		const idsBeiderLaeufe = enqueueSpy.mock.calls.map((c) => c[0].id);
		expect(idsBeiderLaeufe).toEqual(['stabile-id', 'stabile-id']);
	});

	// Ein unlesbarer Zeitpunkt wurde zu NaN, und aus NaN baute der Sync
	// `new Date(NaN).toISOString()` — das wirft, und zwar beim Bauen der Portion, also
	// ausserhalb jedes Fangnetzes: Die Runde brach ab, `isSyncing` blieb auf „läuft", und
	// danach lief kein Sync mehr bis zum Neuladen der Seite. Ein einziger Eintrag einer
	// fremden Datei legte so die ganze Warteschlange still.
	it('macht aus einem unlesbaren Zeitpunkt JETZT, nicht NaN', async () => {
		const { offlineSync } = await import('./stores/offlineSync.svelte.js');
		const vorher = Date.now();

		const anzahl = await offlineSync.importQueueFromJSON(
			sicherungsdatei([
				{ id: 'iso', action_type: 'checkout', barcode_id: 'B-9', timestamp: '2026-09-16T10:00:00Z' }
			])
		);

		expect(anzahl, 'die Buchung geht nicht verloren').toBe(1);
		const gescannt = enqueueSpy.mock.calls[0][0].gescannt_am;
		expect(Number.isFinite(gescannt), 'kein NaN in der Warteschlange').toBe(true);
		expect(gescannt).toBeGreaterThanOrEqual(vorher);
		// Und die Portion laesst sich daraus bauen, statt zu werfen.
		expect(() => new Date(gescannt).toISOString()).not.toThrow();
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
