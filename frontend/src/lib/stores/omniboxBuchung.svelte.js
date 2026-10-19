// Die Buchung eines Scans über POST /api/action: der Schnappschuss, vorher die Auflösung eines
// ohne Netz gemerkten Ausweises, der Versand und die Auswertung der Antwort. Der Versand an
// /api/action steht nur hier (frontend-hygiene-action-endpunkt.test.js). Geladen wird die Datei
// nur vom Omnibox-Store (frontend-hygiene-omnibox-teile.test.js); er reicht Zustand, Merker und
// Rückmeldung als Funktionen herein.

import { apiClient } from '../apiFetch.js';
import { spieleFehlerton } from '../audio.js';
import { buchBarcodes } from './buchBarcodes.svelte.js';
import { ordneScanEin } from '../scanEinordnen.js';

/**
 * Ein Scan und was der Aufrufer dazu vorgibt.
 * @typedef {{ q: string, reloadProfileCb?: (() => void) | null, overrideBlock: boolean,
 *   confirmedChecklist: boolean, absicht: 'ausleihe' | 'rueckgabe' | null }} Auftrag
 */
/** @typedef {import('../offlineWarteschlange.js').OfflineEintrag} OfflineEintrag */

/**
 * @param {{
 *   leseLeser: () => any,
 *   leseMerker: () => string,
 *   vergissMerker: () => void,
 *   leereHinweiszeilen: () => void,
 *   schnappschuss: (q: string, schluessel: string, absicht: Auftrag['absicht']) => OfflineEintrag,
 *   verarbeiteVersandfehler: (e: unknown, eintrag: OfflineEintrag) => Promise<boolean>,
 *   verarbeiteAktionsFehler: (res: Response, q: string) => Promise<void>,
 *   verarbeiteAktionsErgebnis: (daten: any, reloadProfileCb: any, q?: string,
 *     overrideBlock?: boolean) => boolean,
 *   verarbeiteAntwortfehler: (e: unknown) => void,
 *   leuchteBildschirm: (type: string) => void,
 *   leuchteRand: (farbe: string) => void,
 *   zeigeFehlerBanner: (message: string) => void,
 *   clearFehlerBanner: () => void
 * }} theke
 */
export function erzeugeOmniboxBuchung({
	leseLeser,
	leseMerker,
	vergissMerker,
	leereHinweiszeilen,
	schnappschuss,
	verarbeiteVersandfehler,
	verarbeiteAktionsFehler,
	verarbeiteAktionsErgebnis,
	verarbeiteAntwortfehler,
	leuchteBildschirm,
	leuchteRand,
	zeigeFehlerBanner,
	clearFehlerBanner
}) {
	/**
	 * Löst einen ohne Netz gemerkten Ausweis auf, bevor das nächste Buch hinausgeht: Ohne
	 * Leser-Kennung läse der Server den Scan als Rückgabe.
	 *
	 * Entschieden wird an der Wirkung eines Ausweisscans, der nichts bucht, nicht an
	 * `navigator.onLine`, das bei einem Ausfall des Servers weiter „online" meldet. Lädt er die
	 * Person, geht das Buch normal hinaus. Scheitert der Versand, bleibt der Merker, und das Buch
	 * geht mit `ausweis_barcode` in die Warteschlange. Antwortet der Server ohne Person, ist
	 * unklar, wem das Buch gehört: Es wird nicht gebucht, und der Merker fällt.
	 *
	 * @param {string} q der rohe Scan
	 * @param {(() => void) | null} [reloadProfileCb]
	 * @returns {Promise<boolean>} false = dieser Scan wurde bewusst nicht ausgeführt
	 */
	async function merkerAufgeloest(q, reloadProfileCb) {
		if (!leseMerker()) return true;
		// Ein Ausweis selbst darf durch — er ist der Ausweg und ersetzt den Merker.
		if (ordneScanEin(q, buchBarcodes.istBuch).art === 'ausweis') {
			vergissMerker();
			return true;
		}

		const gemerkt = leseMerker();
		let res;
		try {
			res = await apiClient.post('/api/action', {
				query: gemerkt,
				idempotency_key: crypto.randomUUID()
			});
		} catch (e) {
			// Keine Verbindung — der erwartete Fall, kein Fehler. Der Merker bleibt.
			console.warn('Ausweis-Merker nicht auflösbar, bleibt stehen:', e);
			return true;
		}

		try {
			if (!res.ok) await verarbeiteAktionsFehler(res, gemerkt); // wirft immer
			verarbeiteAktionsErgebnis(await res.json(), reloadProfileCb, gemerkt);
		} catch (e) {
			verarbeiteAntwortfehler(e);
		}

		vergissMerker();
		if (leseLeser()?.id) return true;

		leuchteBildschirm('error');
		spieleFehlerton();
		zeigeFehlerBanner(
			`Der ohne Netz gemerkte Ausweis \u201e${gemerkt}\u201c ließ sich nicht laden. Dieser Scan wurde NICHT gebucht — ` +
				`bitte den Ausweis erneut scannen.`
		);
		return false;
	}

	// Bucht einen Scan. Das Ergebnis sagt, ob er glatt durchging: Leser geladen, gebucht oder
	// für später abgelegt.
	/**
	 * @param {Auftrag} auftrag
	 * @returns {Promise<boolean>}
	 */
	async function bucheScan({ q, reloadProfileCb, overrideBlock, confirmedChecklist, absicht }) {
		leereHinweiszeilen();
		clearFehlerBanner();

		// Der Schnappschuss entsteht vor allem Weiteren, damit Escape oder „Theke leeren" die
		// Absicht des Scans nicht mehr ändern.
		let eintrag = schnappschuss(q, crypto.randomUUID(), absicht);

		// Steht ein ohne Netz gemerkter Ausweis, gehört ihm das nächste Buch: erst auflösen, dann
		// buchen (merkerAufgeloest).
		if (leseMerker()) {
			if (!(await merkerAufgeloest(q, reloadProfileCb))) return false;
			// Die Auflösung hat die Person geladen: Sie gehört in den Schnappschuss, der Merker
			// nicht mehr. Trüge der Eintrag beides, entschiede beim Nachbuchen die Reihenfolge der
			// Felder, wem das Buch gehört.
			if (eintrag.ausweis_barcode && leseLeser()?.id) {
				const ohneMerker = { ...eintrag };
				delete ohneMerker.ausweis_barcode;
				eintrag = { ...ohneMerker, leser_id: leseLeser().id, art: absicht ?? 'ausleihe' };
			}
		}

		// Scheitert der Versand, hat der Server nichts gesehen: Der Schnappschuss geht in die
		// Warteschlange. Kam eine Antwort an, wird ihr Fehler gezeigt und nicht eingereiht; nach
		// Ablauf seines Idempotenz-Caches (24 h) buchte der Server sonst ein zweites Mal.
		//
		// Rot nur im Fehlerfall: Sonst überschriebe es das Grün des Erfolgs und das Orange der
		// Fremdrückgabe, und niemand sähe mehr auf die Farbe.
		let res;
		try {
			res = await apiClient.post('/api/action', {
				query: q,
				active_leser_id: eintrag.leser_id ?? undefined,
				confirmed_checklist: confirmedChecklist,
				override_block: overrideBlock,
				idempotency_key: eintrag.id
			});
		} catch (e) {
			leuchteRand('red');
			return verarbeiteVersandfehler(e, eintrag);
		}

		try {
			if (!res.ok) {
				await verarbeiteAktionsFehler(res, q); // wirft immer
			}
			const daten = await res.json();
			return verarbeiteAktionsErgebnis(daten, reloadProfileCb, q, overrideBlock);
		} catch (e) {
			leuchteRand('red');
			verarbeiteAntwortfehler(e);
			return false;
		}
	}

	return /** @satisfies {object} */ ({ bucheScan });
}
