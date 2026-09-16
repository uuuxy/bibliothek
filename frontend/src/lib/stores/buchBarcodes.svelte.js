import { openDB } from 'idb';
import { apiFetch } from '../apiFetch.js';

/**
 * Die Buch-Barcode-Liste auf dem Theken-Rechner (Stufe 1 des Offline-Baus).
 *
 * Wozu: Ohne Netz muss die Theke eine nackte Ziffernfolge selbst einordnen — Buch oder
 * Ausweis? Die Vorsilben sagen es, die Littera-Etiketten des Altbestands und die alten
 * Ausweise sagen es nicht. Der Server liefert dafuer alle Exemplar-Barcodes
 * (GET /api/action/buchbarcodes, mit ETag und gepackt); hier liegen sie, damit sie ein
 * Neuladen ohne Netz ueberstehen.
 *
 * EIGENE Datenbank, nicht die der Warteschlange. Der Unterschied ist Absicht: In der
 * Warteschlange liegen Vorgaenge, die es sonst nirgends gibt — ein Schema-Wechsel dort
 * ist riskant. Diese Liste ist ein Abbild und jederzeit neu holbar; geht sie verloren,
 * kostet das eine Abfrage. Zwei Sorten Haltbarkeit, zwei Ablagen.
 *
 * KEINE Personendaten: nur Buchnummern (so auch der Server im Kopfkommentar der Tuer).
 *
 * Fehler sind hier NIE laut: Die Liste ist eine Verbesserung der Einordnung, keine
 * Voraussetzung fuers Scannen. Faellt sie aus, gelten `B-` und `LMF-` weiter (siehe
 * scanEinordnen.js), und nackte Nummern werden „unklar" — das ist die sichere Seite.
 */

const DB_NAME = 'bibliothek-barcodes-db';
const STORE = 'liste';
const SCHLUESSEL = 'aktuell';

async function db() {
	return openDB(DB_NAME, 1, {
		upgrade(d) {
			if (!d.objectStoreNames.contains(STORE)) d.createObjectStore(STORE);
		}
	});
}

class BuchBarcodes {
	/** @type {Set<string>} */
	#nummern = new Set();
	/** Merker der Fassung — geht als If-None-Match hinaus, damit der Server 304 antworten darf. */
	#stand = '';
	/** Anzahl der gehaltenen Nummern; fuer die Anzeige und die Selbstpruefung. */
	anzahl = $state(0);
	/** Wann zuletzt erfolgreich geholt oder bestaetigt (ms), 0 = nie. */
	geholtAm = $state(0);
	/** Handle des stuendlichen Abgleichs; null = laeuft nicht. @type {ReturnType<typeof setInterval> | null} */
	#zeitgeber = null;

	/** Steht diese Nummer als Exemplar-Barcode auf der Liste? */
	istBuch = (nummer) => this.#nummern.has(nummer);

	/**
	 * Liste aus der Ablage dieses Rechners in den Speicher holen. Nach jedem Neuladen
	 * noetig — und gerade dann, wenn kein Netz da ist, ist sie die einzige Quelle.
	 */
	async laden() {
		try {
			const eintrag = await (await db()).get(STORE, SCHLUESSEL);
			if (!eintrag || !Array.isArray(eintrag.barcodes)) return;
			this.#nummern = new Set(eintrag.barcodes);
			this.#stand = typeof eintrag.stand === 'string' ? eintrag.stand : '';
			this.anzahl = this.#nummern.size;
			this.geholtAm = Number(eintrag.geholtAm) || 0;
		} catch (err) {
			// Privates Fenster, gesperrte Website-Daten, voller Speicher: Die Theke
			// arbeitet weiter, nackte Nummern gelten dann als unklar.
			console.warn('Barcode-Liste nicht lesbar:', err);
		}
	}

	/**
	 * Beim Server nachfragen. 304 heisst „unveraendert" — der Normalfall, denn der
	 * Bestand aendert sich selten und angemeldet wird taeglich.
	 *
	 * @returns {Promise<'neu' | 'unveraendert' | 'fehlgeschlagen'>}
	 */
	async auffrischen() {
		try {
			const kopf = this.#stand ? { 'If-None-Match': `"${this.#stand}"` } : undefined;
			const res = await apiFetch('/api/action/buchbarcodes', { headers: kopf });
			if (res.status === 304) {
				this.geholtAm = Date.now();
				return 'unveraendert';
			}
			if (!res.ok) return 'fehlgeschlagen';
			const daten = await res.json();
			if (!Array.isArray(daten?.barcodes)) return 'fehlgeschlagen';
			this.#nummern = new Set(daten.barcodes.map(String));
			this.#stand = typeof daten.stand === 'string' ? daten.stand : '';
			this.anzahl = this.#nummern.size;
			this.geholtAm = Date.now();
			await this.#ablegen(daten.barcodes);
			return 'neu';
		} catch (err) {
			// Kein Netz ist der erwartete Fall, nicht der Ausnahmefall: Die Liste vom
			// letzten Mal gilt weiter.
			console.warn('Barcode-Liste nicht geholt:', err);
			return 'fehlgeschlagen';
		}
	}

	/** @param {string[]} barcodes */
	async #ablegen(barcodes) {
		try {
			await (
				await db()
			).put(STORE, { stand: this.#stand, barcodes, geholtAm: this.geholtAm }, SCHLUESSEL);
		} catch (err) {
			// Nicht ablegen zu koennen ist kein Grund, die Liste im Speicher wegzuwerfen —
			// sie haelt dann bis zum naechsten Neuladen.
			console.warn('Barcode-Liste nicht abgelegt:', err);
		}
	}

	/**
	 * Stuendlich nachfassen, solange Netz da ist (entschieden am 16.09.2026, OFFEN.md 5.19).
	 *
	 * Geholt wird die Liste beim Anmelden — und ein Kiosk-Tab steht zwoelf Stunden offen.
	 * Was am Vormittag neu inventarisiert wurde, war am Nachmittag ohne Netz eine
	 * „unklare" Nummer, und niemand konnte das wissen. Der Abgleich kostet fast nichts:
	 * Der Server antwortet im Normalfall 304, weil sich der Bestand selten aendert.
	 *
	 * Ohne Netz wird gar nicht erst gefragt — das waere ein Fehler pro Stunde im Protokoll
	 * fuer nichts. Das Handle liegt am Store, damit der Aufrufer ihn wegraeumen kann; ein
	 * ueberlebender Zeitgeber faerbt sonst einen ganzen Testlauf rot
	 * (frontend-hygiene-thekenzeitgeber.test.js).
	 *
	 * @param {number} intervallMs Abstand zweier Abgleiche; die Vorgabe ist eine Stunde.
	 */
	starteAbgleich(intervallMs = 60 * 60 * 1000) {
		this.stoppeZeitgeber();
		this.#zeitgeber = setInterval(() => {
			if (typeof navigator === 'undefined' || navigator.onLine) void this.auffrischen();
		}, intervallMs);
	}

	/** Den stuendlichen Abgleich beenden (Abmelden, Abbau der Seite, Test-Ende). */
	stoppeZeitgeber() {
		if (this.#zeitgeber !== null) {
			clearInterval(this.#zeitgeber);
			this.#zeitgeber = null;
		}
	}

	/**
	 * Beim Anmelden: Liste bereitstellen.
	 *
	 * Erst aus der eigenen Ablage laden — das ist die Fassung, die ein Neuladen OHNE Netz
	 * ueberlebt und dann die einzige Quelle ist. Danach beim Server nachfragen; der
	 * antwortet im Normalfall 304 „unveraendert", weil sich der Bestand selten aendert und
	 * taeglich angemeldet wird.
	 *
	 * Bewusst leise und ohne Rueckgabe: Ohne Liste gelten `B-` und `LMF-` weiter, und
	 * nackte Nummern werden „unklar" — die sichere Seite. Eine Fehlermeldung waere Laerm
	 * ueber etwas, das die Theke nicht anhaelt.
	 */
	async bereitstellen() {
		await this.laden();
		await this.auffrischen();
		this.starteAbgleich();
	}

	/** Nur fuer Tests: Speicher und Ablage leeren. */
	async _leeren() {
		this.stoppeZeitgeber();
		this.#nummern = new Set();
		this.#stand = '';
		this.anzahl = 0;
		this.geholtAm = 0;
		try {
			await (await db()).delete(STORE, SCHLUESSEL);
		} catch {
			/* egal */
		}
	}
}

export const buchBarcodes = new BuchBarcodes();
