import {
	loadQueue,
	dequeueOfflineAction,
	enqueueOfflineAction,
	normalisiereEintrag
} from '../offlineQueue.js';
import { apiClient } from '../apiFetch.js';
import { playSoundSuccess } from '../audio.js';
import { showToast } from '../../inventur/lib/store.svelte.js';

// Baut das Batch-Payload. Eine Ausleihe trägt ihre Person: Schüler als active_student_id,
// Lehrkraft als active_teacher_id (Handapparat; der Stapel-Endpunkt kennt das Feld seit
// jeher, geschickt wurde es bis zum 15.09.2026 nie). Eine Rückgabe trägt keine Person.
/** @param {import('../offlineQueue.js').OfflineEintrag[]} batchItems */
function baueBatchPayload(batchItems) {
	return batchItems.map((item) => {
		/** @type {{ query: string, idempotency_key: string, active_student_id?: string, active_teacher_id?: string }} */
		const req = {
			query: item.barcode,
			idempotency_key: item.id
		};
		if (item.art === 'ausleihe' && item.schueler_id) {
			req.active_student_id = item.schueler_id;
		} else if (item.art === 'ausleihe' && item.lehrer_id) {
			req.active_teacher_id = item.lehrer_id;
		}
		return req;
	});
}

/** @type {Record<string, string>} */
const TYPNAME = { ausleihe: 'Ausleihe', rueckgabe: 'Rückgabe' };
/** @param {string | undefined} typ */
const nenne = (typ) => TYPNAME[typ ?? ''] ?? (typ ? `„${typ}“` : 'nichts');

// Erledigt ist nur, was der Server WIE GESCANNT gebucht hat (OFFEN.md 2.2, Commit 6).
// Gibt zurück, was der Bediener prüfen muss, und ob die Runde weitergehen darf.
//
// - Erfolg mit passendem Typ: still ausgebucht.
// - Erfolg mit anderem Typ (Ausleihe gescannt, Rückgabe gebucht — das Buch war schon bei
//   diesem Kind): ausgebucht UND gemeldet, mit Barcode und beiden Typen. Blockiert nicht;
//   das kommt erst mit der Nachbuch-Tür in Stufe 3, die den Fall benennen kann.
// - 4xx außer 429: ausgebucht und gemeldet. Der Server hat fachlich entschieden (Buch nicht
//   gefunden, Schüler gesperrt). Bis zum Rasterdurchgang am 06.09.2026 erfuhr das niemand:
//   Eine Klasse gibt 18 Bücher offline zurück, vier werden abgelehnt, und die Ausleihen
//   laufen ins Mahnwesen bis zur Rechnung an die Eltern.
// - 5xx, 429 und ein Index, den der Server nicht beantwortet hat (Schweigen ist kein
//   Erfolg): bleibt liegen, die Runde endet. Bis zum 15.09.2026 galt „kein Ergebnis" als
//   erledigt, und ein liegengebliebener Eintrag wurde ohne Pause sofort erneut gesendet.
/**
 * @param {any} data
 * @param {import('../offlineQueue.js').OfflineEintrag[]} batchItems
 * @returns {Promise<{ pruefen: { barcode: string, meldung: string }[], weiter: boolean }>}
 */
async function verarbeiteBatchErgebnisse(data, batchItems) {
	/** @type {{ barcode: string, meldung: string }[]} */
	const pruefen = [];
	let weiter = true;
	for (let i = 0; i < batchItems.length; i++) {
		const item = batchItems[i];
		const result = data.results?.find((/** @type {any} */ r) => r.index === i);
		if (!result) {
			weiter = false;
			continue;
		}
		if (result.success) {
			const gebucht = result.data?.type;
			if (gebucht !== item.art) {
				pruefen.push({
					barcode: item.barcode,
					meldung: `als ${nenne(item.art)} gescannt, der Server buchte ${nenne(gebucht)}`
				});
			}
			await dequeueOfflineAction(item.id);
			continue;
		}
		if (result.status >= 400 && result.status < 500 && result.status !== 429) {
			pruefen.push({
				barcode: item.barcode,
				meldung: `nicht angenommen${result.error ? ` (${result.error})` : ''}`
			});
			await dequeueOfflineAction(item.id);
			continue;
		}
		weiter = false;
	}
	return { pruefen, weiter };
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

// Sagt dem Bediener, was er von Hand klären muss — mit Barcode und Grund je Eintrag. Das
// Muster steht im Haus schon fertig (useFehlbestand: „n gelöscht, m übersprungen").
/** @param {{ barcode: string, meldung: string }[]} pruefen */
function meldeZuPruefende(pruefen) {
	if (pruefen.length === 0) return;
	const liste = pruefen.map((p) => `„${p.barcode}“: ${p.meldung}`).join('; ');
	showToast(
		pruefen.length === 1
			? `Offline-Scan ${liste} — bitte von Hand prüfen.`
			: `${pruefen.length} Offline-Scans brauchen Prüfung: ${liste} — bitte von Hand prüfen.`,
		'error'
	);
}

function createOfflineSyncStore() {
	let pendingCount = $state(0);
	let isSyncing = $state(false);
	let isOffline = $state(typeof navigator !== 'undefined' ? !navigator.onLine : false);
	// Die Warteschlange ließ sich nicht lesen (Commit 5): Das Band sagt es, statt 0 zu zeigen.
	let warteschlangeFehler = $state(false);

	async function updateCount() {
		try {
			const q = await loadQueue();
			pendingCount = q.length;
			warteschlangeFehler = false;
		} catch {
			warteschlangeFehler = true;
		}
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
			const { pruefen, weiter } = await verarbeiteBatchErgebnisse(data, batchItems);
			meldeZuPruefende(pruefen);
			await updateCount();
			if (!weiter) return false;

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
			/** @type {import('../offlineQueue.js').OfflineEintrag[]} */
			let q;
			try {
				q = await loadQueue();
				warteschlangeFehler = false;
			} catch {
				warteschlangeFehler = true;
				break;
			}
			if (q.length === 0) break;

			// loadQueue liefert nach Scan-Zeitpunkt geordnet.
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

	/**
	 * Spielt eine Notfall-Sicherung zurück in die lokale Warteschlange; der Auto-Sync
	 * schiebt sie danach zum Server.
	 *
	 * item.id MUSS mitwandern: Diese ID ist der Idempotenz-Schlüssel, den der Server
	 * kennt (siehe baueBatchPayload und idempotency_keys in api/action.go). Ohne sie
	 * vergibt normalisiereEintrag eine frische UUID — und dieselbe Datei zweimal
	 * eingespielt würde jede Aktion ZWEIMAL ausführen. Bei zehn Kiosk-Rechnern mit
	 * einem gemeinsamen Sicherungsordner ist doppeltes Einspielen der Normalfall,
	 * nicht der Ausnahmefall: Zwei Admins, oder einer, der unsicher ist, ob er es
	 * schon getan hat. Mit dem Schlüssel ist der zweite Durchlauf wirkungslos.
	 *
	 * Sicherungen in Format 1 (vor dem 15.09.2026, `action_type`/`barcode_id`) und
	 * Format 2 (`art`/`barcode`, mit Scan-Zeitpunkt) werden gleich behandelt.
	 * @param {File} file
	 */
	async function importQueueFromJSON(file) {
		try {
			const text = await file.text();
			const items = JSON.parse(text);
			if (!Array.isArray(items)) throw new Error('Invalid format');

			let importedCount = 0;
			const promises = [];
			for (const roh of items) {
				const eintrag = normalisiereEintrag(roh);
				if (!eintrag) continue;
				promises.push(enqueueOfflineAction(eintrag));
				importedCount++;
			}
			await Promise.all(promises);

			await updateCount();
			startSync();
			return importedCount;
		} catch (e) {
			console.error(e);
			throw new Error('Fehler beim Einlesen der Backup-Datei.');
		}
	}

	// Der Dialog nennt den Ausweg, nicht nur die Gefahr: "Nicht ausschalten" ist an
	// einem Schulrechner keine Handlungsanweisung, die jemand bis Feierabend
	// durchhalten kann. Verlassen darf man sich darauf ohnehin nicht — beim
	// Herunterfahren von Windows bekommt der Browser oft keine Gelegenheit mehr,
	// diesen Dialog zu zeigen. Deshalb steht dieselbe Anweisung dauerhaft im Banner.
	function handleBeforeUnload(e) {
		if (pendingCount > 0) {
			e.preventDefault();
			const msg = `${pendingCount} Vorgang/Vorgänge sind nur auf diesem Rechner gespeichert. Bitte zuerst "Sicherung speichern" — danach darf der Rechner aus.`;
			e.returnValue = msg;
			return msg;
		}
	}

	// Einmalig je Seite: Jede Anmeldung ruft init() (hintergrundAbrufe.svelte.js); ohne
	// Sperre stapelten Logout→Login Listener und 30-s-Intervalle, und startSync() lief
	// mehrfach parallel gegen dieselbe Warteschlange.
	let initialisiert = false;
	function init() {
		if (initialisiert) return;
		initialisiert = true;
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
			// Jede Minute ein Anlauf, falls das online-Ereignis fehlte oder ein Eintrag nach
			// 5xx/429 liegen blieb (die Runde endet dort, statt sofort erneut zu senden).
			setInterval(() => {
				if (pendingCount > 0 && !isOffline) startSync();
			}, 60000);
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
		get warteschlangeFehler() {
			return warteschlangeFehler;
		},
		updateCount,
		startSync,
		exportQueueAsJSON,
		importQueueFromJSON,
		init
	};
}

export const offlineSync = createOfflineSyncStore();
