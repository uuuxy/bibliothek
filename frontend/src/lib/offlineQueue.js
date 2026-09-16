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
 * beim Scan; `leser_id` die Person, die in diesem Moment geladen war; `gescannt_am` der
 * Scan-Zeitpunkt (ms). Format 1 (`{action_type, barcode_id, schueler_id, timestamp}`, bis
 * 15.09.2026, auch in alten Sicherungsdateien) wird beim Lesen übersetzt.
 *
 * Seit Migration 125 gibt es EIN Personenfeld. Das alte `schueler_id` trug schon dieselbe
 * Kennung wie `leser_id` und wird übersetzt; ein altes `lehrer_id` dagegen zeigte auf ein
 * KONTO und nicht auf einen Leser — es wird verworfen, statt die Buchung einer falschen
 * Person zuzuschreiben. Der Eintrag bleibt als Rückgabe bzw. als „Ausweis unbekannt"
 * stehen und meldet sich, statt still danebenzugreifen.
 *
 * @typedef {{
 *   id: string,
 *   art: 'ausleihe' | 'rueckgabe',
 *   barcode: string,
 *   leser_id: string | null,
 *   gescannt_am: number,
 *   ausweis_barcode?: string,
 *   mono?: number,
 *   ursprung?: number
 * }} OfflineEintrag
 *
 * Die drei optionalen Felder, und warum sie optional BLEIBEN muessen:
 *
 * `ausweis_barcode` traegt einen ohne Netz gescannten Ausweis, den erst der Server
 * aufloesen kann (die Nachbuch-Tuer kennt das Feld). Ein Eintrag ohne ihn ist der
 * Normalfall: Die Person war beim Scan schon geladen und steht in `leser_id`.
 *
 * `mono` und `ursprung` sind der Uhr-Anker: `performance.now()` beim Scan und der
 * Zeitursprung des Seitenaufrufs. Aus ihnen bestimmt der Sync den Scan-Zeitpunkt neu,
 * wenn der Eintrag aus DEMSELBEN Seitenaufruf stammt — die Wanduhr eines Theken-Rechners
 * wird gern in dem Moment korrigiert, in dem das Netz zurueckkommt, also zwischen Scan
 * und Versand. Fehlen sie (Eintrag aus einem frueheren Seitenaufruf, eingespielte
 * Sicherung, Format 1), gilt `gescannt_am` unveraendert — dort gibt es nichts Besseres.
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
	/** @type {OfflineEintrag} */
	const eintrag = {
		id: roh.id || crypto.randomUUID(),
		art,
		barcode: String(barcode),
		leser_id: roh.leser_id ?? roh.schueler_id ?? null,
		gescannt_am: Number(roh.gescannt_am ?? roh.timestamp ?? Date.now())
	};
	// Nur uebernehmen, was wirklich dasteht: Ein `undefined` in IndexedDB waere ein Feld,
	// das es gibt und das nichts bedeutet.
	if (typeof roh.ausweis_barcode === 'string' && roh.ausweis_barcode !== '') {
		eintrag.ausweis_barcode = roh.ausweis_barcode;
	}
	if (Number.isFinite(roh.mono) && Number.isFinite(roh.ursprung)) {
		eintrag.mono = Number(roh.mono);
		eintrag.ursprung = Number(roh.ursprung);
	}
	return eintrag;
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
		// Weiterwerfen, nicht [] (Commit 5, 15.09.2026): Ein leeres Ergebnis hieße „nichts
		// offen" — das Band zeigte 0, und niemand hätte gewusst, dass die Einträge unlesbar sind.
		console.error('Offline-Warteschlange nicht lesbar:', err);
		throw err;
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
		// Weiterwerfen (Commit 5): Bis zum 15.09.2026 wurde jeder IndexedDB-Fehler geschluckt —
		// privates Fenster, gesperrte Website-Daten, voller Speicher — und die Theke meldete
		// „gespeichert" mit Erfolgston. Der Scan war weg.
		console.error('Offline-Eintrag nicht gespeichert:', err);
		throw err;
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
		// Weiterwerfen: Der Sync bricht die Runde ab und sendet den Eintrag später erneut —
		// derselbe Idempotenz-Schlüssel, also ohne Doppelbuchung.
		console.error(`Offline-Eintrag ${id} nicht ausgebucht:`, err);
		throw err;
	}
}
