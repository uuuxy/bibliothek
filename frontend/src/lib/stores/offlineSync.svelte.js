import { loadQueue, dequeueOfflineAction } from '../offlineQueue.js';
import { apiClient } from '../apiFetch.js';
import { playSoundSuccess } from '../audio.js';

// Baut das Batch-Payload; nur Checkouts mit Schüler-ID tragen active_student_id.
function baueBatchPayload(batchItems) {
	return batchItems.map((item) => {
		const req = {
			query: item.barcode_id,
			idempotency_key: item.id
		};
		if (item.action_type === 'checkout' && item.schueler_id) {
			req.active_student_id = item.schueler_id;
		}
		return req;
	});
}

// Bucht je Item aus, wenn der Server Erfolg oder einen permanenten 4xx-Fehler
// (außer 429 Too Many Requests) meldet.
async function verarbeiteBatchErgebnisse(data, batchItems) {
	for (let i = 0; i < batchItems.length; i++) {
		const item = batchItems[i];
		const result = data.results?.find((r) => r.index === i);

		// Dequeue on success, on a permanent client error (4xx except 429), or as
		// failsafe when the backend returned no index for this item (overall 200 OK).
		if (
			!result ||
			result.success ||
			(result.status >= 400 && result.status < 500 && result.status !== 429)
		) {
			await dequeueOfflineAction(item.id);
		}
	}
}

async function exportQueueAsJSON() {
	const q = await loadQueue();
	if (q.length === 0) return;
	const blob = new Blob([JSON.stringify(q, null, 2)], { type: 'application/json' });
	const url = URL.createObjectURL(blob);
	const a = document.createElement('a');
	a.href = url;
	a.download = `offline_scans_backup_${new Date().toISOString().slice(0, 10)}.json`;
	document.body.appendChild(a);
	a.click();
	a.remove();
	URL.revokeObjectURL(url);
}

function createOfflineSyncStore() {
	let pendingCount = $state(0);
	let isSyncing = $state(false);
	let isOffline = $state(typeof navigator !== 'undefined' ? !navigator.onLine : false);

	async function updateCount() {
		const q = await loadQueue();
		pendingCount = q.length;
	}

	// Verschickt einen Batch und verarbeitet dessen Ergebnisse. Liefert false, wenn der
	// Sync abbrechen soll (kompletter Batch-Fehler wie 502, oder Netzwerkfehler).
	async function sendeBatch(payload, batchItems, queueLength) {
		try {
			const res = await apiClient.post('/api/action/batch', payload);

			if (!res.ok) {
				// Batch request failed completely (e.g. 502 Bad Gateway), stop syncing
				return false;
			}

			const data = await res.json();
			await verarbeiteBatchErgebnisse(data, batchItems);
			await updateCount();

			// Network Jitter: 200-500 ms Pause vor dem nächsten Batch, damit mehrere
			// Geräte nach einer Offline-Phase nicht im Gleichtakt auf den Server laufen.
			//
			// Math.random() ist hier bewusst richtig und kein Sicherheitsmangel
			// (SonarQube javascript:S2245 meldet jede Verwendung): Das Ergebnis ist eine
			// Wartedauer, kein Geheimnis. Es schützt nichts, identifiziert nichts und ist
			// für niemanden von Vorteil, wenn er es vorhersagt. Eine kryptografische
			// Quelle brächte hier keinerlei Schutz — nur Aufwand.
			if (queueLength > 50) {
				const jitter = 200 + Math.random() * 300; // NOSONAR — Jitter, kein Sicherheitskontext (siehe Kommentar oben; S2245 ist hier ein False Positive)
				await new Promise((resolve) => setTimeout(resolve, jitter));
			}
			return true;
		} catch (err) {
			console.warn('Offline-Sync: Netzwerkfehler beim Batch-Versand:', err);
			return false;
		}
	}

	async function startSync() {
		if (isSyncing || !navigator.onLine) return;
		isSyncing = true;

		let syncedAny = false;

		while (navigator.onLine) {
			const q = await loadQueue();
			if (q.length === 0) break;

			// Ensure they are processed in order of creation
			q.sort((a, b) => a.timestamp - b.timestamp);

			const batchItems = q.slice(0, 50);
			const payload = baueBatchPayload(batchItems);

			const ok = await sendeBatch(payload, batchItems, q.length);
			if (!ok) break;
			syncedAny = true;
		}

		if (syncedAny && pendingCount === 0) {
			playSoundSuccess();
		}
		isSyncing = false;
	}

	async function importQueueFromJSON(file) {
		try {
			const text = await file.text();
			const items = JSON.parse(text);
			if (!Array.isArray(items)) throw new Error('Invalid format');

			const { enqueueOfflineAction } = await import('../offlineQueue.js');
			let importedCount = 0;
			for (const item of items) {
				if (!item.action_type || !item.barcode_id) continue;
				await enqueueOfflineAction(item.action_type, item.barcode_id, item.schueler_id || null);
				importedCount++;
			}

			await updateCount();
			startSync();
			return importedCount;
		} catch (e) {
			console.error(e);
			throw new Error('Fehler beim Einlesen der Backup-Datei.');
		}
	}

	function handleBeforeUnload(e) {
		if (pendingCount > 0) {
			e.preventDefault();
			const msg =
				'Es gibt noch ungespeicherte Daten (Offline-Queue). Datenverlust droht! Bitte Browser nicht schließen.';
			e.returnValue = msg;
			return msg;
		}
	}

	function init() {
		if (typeof window !== 'undefined') {
			isOffline = !navigator.onLine;
			updateCount();

			window.addEventListener('online', () => {
				isOffline = false;
				startSync();
			});

			window.addEventListener('offline', () => {
				isOffline = true;
			});

			window.addEventListener('beforeunload', handleBeforeUnload);
			// Periodic check every 30s just in case online event missed or transient 5xx errors
			setInterval(() => {
				if (pendingCount > 0 && !isOffline) startSync();
			}, 30000);
		}
	}

	return {
		get isOffline() {
			return isOffline;
		},
		get pendingCount() {
			return pendingCount;
		},
		get isSyncing() {
			return isSyncing;
		},
		updateCount,
		startSync,
		exportQueueAsJSON,
		importQueueFromJSON,
		init
	};
}

export const offlineSync = createOfflineSyncStore();
