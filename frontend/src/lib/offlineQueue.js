import { openDB } from 'idb';

const DB_NAME = 'bibliothek-offline-db';
const STORE_NAME = 'offline_actions';

async function getDB() {
	return openDB(DB_NAME, 3, {
		upgrade(db, oldVersion, newVersion, transaction) {
			if (oldVersion < 2) {
				if (db.objectStoreNames.contains('scans')) {
					db.deleteObjectStore('scans');
				}
			}
			let store;
			if (!db.objectStoreNames.contains(STORE_NAME)) {
				store = db.createObjectStore(STORE_NAME, { keyPath: 'id' });
			} else {
				store = transaction.objectStore(STORE_NAME);
			}
			if (oldVersion < 3) {
				if (!store.indexNames.contains('timestamp')) {
					store.createIndex('timestamp', 'timestamp');
				}
			}
		}
	});
}

/**
 * Ein Eintrag der Warteschlange (Format 2, seit dem 15.09.2026): der Schnappschuss vom Scan.
 *
 * `id` ist der Idempotenz-Schlüssel, den der Server kennt (api/action.go); `art` die Absicht
 * beim Scan; `schueler_id` ODER `lehrer_id` die Person, die in diesem Moment geladen war;
 * `gescannt_am` der Scan-Zeitpunkt (ms). Format 1 (`{action_type, barcode_id, schueler_id,
 * timestamp}`, bis 15.09.2026, auch in alten Sicherungsdateien) wird beim Lesen übersetzt.
 *
 * @typedef {{
 *   id: string,
 *   art: 'ausleihe' | 'rueckgabe',
 *   barcode: string,
 *   schueler_id: string | null,
 *   lehrer_id: string | null,
 *   gescannt_am: number
 * }} OfflineEintrag
 */

/**
 * Übersetzt einen gespeicherten oder eingespielten Eintrag beider Formate in Format 2.
 * Liefert null, wenn Barcode oder Absicht fehlen — so ein Eintrag ließe sich nicht buchen.
 * @param {any} roh
 * @returns {OfflineEintrag | null}
 */
export function normalisiereEintrag(roh) {
	if (!roh || typeof roh !== 'object') return null;
	const barcode = roh.barcode ?? roh.barcode_id;
	const art =
		roh.art ??
		(roh.action_type === 'checkout'
			? 'ausleihe'
			: roh.action_type === 'checkin'
				? 'rueckgabe'
				: undefined);
	if (!barcode || (art !== 'ausleihe' && art !== 'rueckgabe')) return null;
	return {
		id: roh.id || crypto.randomUUID(),
		art,
		barcode: String(barcode),
		schueler_id: roh.schueler_id ?? null,
		lehrer_id: roh.lehrer_id ?? null,
		gescannt_am: Number(roh.gescannt_am ?? roh.timestamp ?? Date.now())
	};
}

/**
 * Lädt die Warteschlange aus IndexedDB, in Format 2, nach Scan-Zeitpunkt geordnet.
 * @returns {Promise<OfflineEintrag[]>}
 */
export async function loadQueue() {
	try {
		const db = await getDB();
		const roh = await db.getAll(STORE_NAME);
		return roh
			.map(normalisiereEintrag)
			.filter((e) => e !== null)
			.sort((a, b) => a.gescannt_am - b.gescannt_am);
	} catch (err) {
		console.error('Failed to load offline queue from IndexedDB:', err);
		return [];
	}
}

/**
 * Reiht den Schnappschuss eines Scans ein. Der Index `timestamp` (Schema-Version 3) wird
 * weiter befüllt, damit alte und neue Einträge in derselben Ordnung liegen.
 * @param {OfflineEintrag} eintrag
 * @returns {Promise<void>}
 */
export async function enqueueOfflineAction(eintrag) {
	try {
		const db = await getDB();
		await db.add(STORE_NAME, { ...eintrag, timestamp: eintrag.gescannt_am });
	} catch (err) {
		console.error('Failed to enqueue offline action to IndexedDB:', err);
	}
}

/**
 * Deletes an action from the queue.
 * @param {string} id
 * @returns {Promise<void>}
 */
export async function dequeueOfflineAction(id) {
	try {
		const db = await getDB();
		await db.delete(STORE_NAME, id);
	} catch (err) {
		console.error(`Failed to dequeue offline action ${id}:`, err);
	}
}
