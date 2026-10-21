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
 * Ein Eintrag der Warteschlange (Format 2): der Schnappschuss vom Scan.
 *
 * `id` ist der Idempotenz-Schlüssel, den der Server kennt (api/theke/eingabe.go); `art` die Absicht
 * beim Scan; `leser_id` die Person, die in diesem Moment geladen war; `gescannt_am` der
 * Scan-Zeitpunkt (ms). Format 1 (`{action_type, barcode_id, schueler_id, timestamp}`, auch
 * in alten Sicherungsdateien) wird beim Lesen übersetzt.
 *
 * Seit Migration 125 gibt es ein Personenfeld. Das alte `schueler_id` trug schon dieselbe
 * Kennung wie `leser_id` und wird übersetzt; ein altes `lehrer_id` dagegen zeigte auf ein
 * Konto und nicht auf einen Leser. Es wird verworfen, statt die Buchung einer falschen
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
 *   besuch_seit_ms?: number,
 *   mono?: number,
 *   ursprung?: number
 * }} OfflineEintrag
 *
 * Die vier optionalen Felder, und warum sie optional BLEIBEN muessen:
 *
 * `ausweis_barcode` traegt einen ohne Netz gescannten Ausweis, den erst der Server
 * aufloesen kann (die Nachbuch-Tuer kennt das Feld). Ein Eintrag ohne ihn ist der
 * Normalfall: Die Person war beim Scan schon geladen und steht in `leser_id`.
 *
 * `besuch_seit_ms` sagt bei einem Scan mit Person, wie lange sie schon an der Theke stand.
 * Daran nimmt der Server ein Buch zurück, das schon vor dem Besuch auf ihrem Konto lag, wie am
 * Netz. Fehlt es (Rückgabe, ältere Sicherung), bleibt ein Buch, das sie schon hat, bei ihr.
 *
 * `mono` und `ursprung` sind der Uhr-Anker: `performance.now()` beim Scan und der
 * Zeitursprung des Seitenaufrufs. Aus ihnen bestimmt der Sync den Scan-Zeitpunkt neu,
 * wenn der Eintrag aus DEMSELBEN Seitenaufruf stammt — die Wanduhr eines Theken-Rechners
 * wird gern in dem Moment korrigiert, in dem das Netz zurueckkommt, also zwischen Scan
 * und Versand. Fehlen sie (Eintrag aus einem frueheren Seitenaufruf, eingespielte
 * Sicherung, Format 1), gilt `gescannt_am` unveraendert — dort gibt es nichts Besseres.
 */

/**
 * Die Absicht in der früheren Schreibweise (Format 1).
 * @param {unknown} actionType
 */
function artAusActionType(actionType) {
	if (actionType === 'checkout') return 'ausleihe';
	if (actionType === 'checkin') return 'rueckgabe';
	return undefined;
}

/**
 * Übersetzt einen gespeicherten oder eingespielten Eintrag beider Formate in Format 2.
 * Liefert null, wenn Barcode oder Absicht fehlen — so ein Eintrag ließe sich nicht buchen.
 * @param {any} roh
 * @returns {OfflineEintrag | null}
 */
export function normalisiereEintrag(roh) {
	if (!roh || typeof roh !== 'object') return null;
	const barcode = roh.barcode ?? roh.barcode_id;
	const art = roh.art ?? artAusActionType(roh.action_type);
	if (!barcode || (art !== 'ausleihe' && art !== 'rueckgabe')) return null;
	// Ein unlesbarer Zeitpunkt wird zu jetzt, nicht zu NaN (`Number()` auf einem ISO-Text ist
	// NaN): Aus NaN baut der Sync `new Date(NaN).toISOString()`, und das wirft beim Bauen der
	// Portion, außerhalb jedes Fangnetzes. Die Runde bräche ab, `gleichtAb` bliebe auf
	// „läuft", und bis zum Neuladen liefe kein Sync mehr; ein einziger Eintrag einer fremden
	// oder von Hand gebauten Sicherungsdatei legte so die ganze Warteschlange still.
	//
	// Verworfen wird deshalb nicht: Der Scan ist eine echte Buchung, und der Server begrenzt
	// den Zeitpunkt ohnehin auf seine eigene Uhr. Lieber eine Buchung mit dem Zeitpunkt des
	// Einspielens als eine verlorene.
	const zeit = Number(roh.gescannt_am ?? roh.timestamp ?? Date.now());
	/** @type {OfflineEintrag} */
	const eintrag = {
		id: roh.id || crypto.randomUUID(),
		art,
		barcode: String(barcode),
		leser_id: roh.leser_id ?? roh.schueler_id ?? null,
		gescannt_am: Number.isFinite(zeit) ? zeit : Date.now()
	};
	// Nur uebernehmen, was wirklich dasteht: Ein `undefined` in IndexedDB waere ein Feld,
	// das es gibt und das nichts bedeutet.
	if (typeof roh.ausweis_barcode === 'string' && roh.ausweis_barcode !== '') {
		eintrag.ausweis_barcode = roh.ausweis_barcode;
	}
	if (Number.isFinite(roh.besuch_seit_ms) && roh.besuch_seit_ms >= 0) {
		eintrag.besuch_seit_ms = Number(roh.besuch_seit_ms);
	}
	if (Number.isFinite(roh.mono) && Number.isFinite(roh.ursprung)) {
		eintrag.mono = Number(roh.mono);
		eintrag.ursprung = Number(roh.ursprung);
	}
	return eintrag;
}

/**
 * Die Person, mit der ein Scan zum Server geht: nur bei einer Ausleihe. Eine Rückgabe trägt keine,
 * am Netz wie beim Nachbuchen: Mit Person läse der Server ein inzwischen freies Buch als Ausleihe
 * an sie (zweiter Klick auf „Buch zurückgeben", eine Akte, die nicht nachgeladen ist).
 * @param {Pick<OfflineEintrag, 'art' | 'leser_id'>} eintrag
 * @returns {string | undefined}
 */
export function leserFuerVersand(eintrag) {
	return eintrag.art === 'ausleihe' && eintrag.leser_id ? eintrag.leser_id : undefined;
}

/**
 * Lädt die Warteschlange aus IndexedDB, in Format 2, nach Scan-Zeitpunkt geordnet.
 * @returns {Promise<OfflineEintrag[]>}
 */
export async function ladeWarteschlange() {
	try {
		const db = await getDB();
		const roh = await db.getAll(STORE_NAME);
		return roh
			.map(normalisiereEintrag)
			.filter((e) => e !== null)
			.sort((a, b) => a.gescannt_am - b.gescannt_am);
	} catch (err) {
		// Weiterwerfen, nicht []: Ein leeres Ergebnis hieße „nichts offen", das Band zeigte 0,
		// und niemand wüsste, dass die Einträge unlesbar sind.
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
		// Weiterwerfen: Mit einem geschluckten IndexedDB-Fehler (privates Fenster, gesperrte
		// Website-Daten, voller Speicher) meldete die Theke „gespeichert" mit Erfolgston, und
		// der Scan wäre weg.
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
