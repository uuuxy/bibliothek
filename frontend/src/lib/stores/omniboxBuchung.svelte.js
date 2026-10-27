// Die Buchung eines Scans über POST /api/action: Schnappschuss, vorher die Auflösung eines ohne
// Netz gemerkten Ausweises, Versand und Auswertung der Antwort. Der Versand an /api/action steht
// nur hier (frontend-hygiene-action-endpunkt.test.js); geladen wird die Datei nur vom Omnibox-Store
// (frontend-hygiene-omnibox-teile.test.js), der Zustand, Merker und Rückmeldung hereinreicht.

import { apiClient, leseApiFehler } from '../apiFetch.js';
import { spieleFehlerton } from '../utils/audio.js';
import { buchBarcodes } from './buchBarcodes.svelte.js';
import { ordneScanEin } from '../utils/scanEinordnen.js';
import { leserFuerVersand } from './offlineWarteschlange.js';

// 502 bis 504 und 423: Gebucht hat der Server nicht (Neustart, Sitzung nicht lesbar, gesperrt).
const nichtErreicht = (/** @type {Response} */ res) => [423, 502, 503, 504].includes(res.status);

/**
 * Ein Scan und was der Aufrufer dazu vorgibt.
 * @typedef {{ q: string, reloadProfileCb?: (() => void) | null, overrideBlock: boolean,
 *   confirmedChecklist: boolean, absicht: 'ausleihe' | 'rueckgabe' | null }} Auftrag
 */
/** @typedef {import('./offlineWarteschlange.js').OfflineEintrag} OfflineEintrag */

/**
 * @param {{
 *   leseLeser: () => any,
 *   leseMerker: () => string,
 *   vergissMerker: () => void,
 *   leereHinweiszeilen: () => void,
 *   schnappschuss: (q: string, schluessel: string, absicht: Auftrag['absicht']) => OfflineEintrag,
 *   verarbeiteVersandfehler: (e: unknown, eintrag: OfflineEintrag) => Promise<boolean>,
 *   verarbeiteAktionsFehler: (res: Response, meldung: string, q: string) => never,
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
	// Zählt, wie oft die Theke geleert wurde (thekeLeeren.js): Eine Antwort von vorher lädt keinen
	// Leser, öffnet keine Rückfrage und merkt keinen Ausweis mehr (docs/OFFEN.md 5.74).
	let stand = 0;
	function verwirfLaufendeAntworten() {
		stand += 1;
	}

	// Erst nach dem Lesen fragt die Buchung, ob die Theke geleert wurde: Auch das Lesen dauert.
	const leseRumpf = (/** @type {Response} */ res) => (res.ok ? res.json() : leseApiFehler(res));

	/**
	 * Löst einen ohne Netz gemerkten Ausweis auf, bevor das nächste Buch hinausgeht: Ohne
	 * Leser-Kennung läse der Server den Scan als Rückgabe.
	 *
	 * Entschieden wird an der Wirkung eines Ausweisscans, der nichts bucht, nicht an `navigator.onLine`.
	 * Lädt er die Person, geht das Buch normal hinaus. Scheitert der Versand (auch 502 bis 504, 423),
	 * bleibt der Merker, und das Buch geht mit `ausweis_barcode` in die Warteschlange. Antwortet der
	 * Server ohne Person, ist unklar, wem das Buch gehört: Es wird nicht gebucht, der Merker fällt.
	 *
	 * @param {string} q der rohe Scan
	 * @param {(() => void) | null | undefined} reloadProfileCb
	 * @param {() => boolean} ueberholt die Theke wurde seit dem Scan geleert
	 * @returns {Promise<boolean>} false = dieser Scan wurde bewusst nicht ausgeführt
	 * @throws wenn der Versand scheitert; bucheScan legt das Buch dann mit dem Merker ab
	 */
	async function merkerAufgeloest(q, reloadProfileCb, ueberholt) {
		if (!leseMerker()) return true;
		// Ein Ausweis selbst darf durch — er ist der Ausweg und ersetzt den Merker.
		if (ordneScanEin(q, buchBarcodes.istBuch).art === 'ausweis') {
			vergissMerker();
			return true;
		}

		const gemerkt = leseMerker();
		const res = await apiClient.post('/api/action', {
			query: gemerkt,
			idempotency_key: crypto.randomUUID()
		});
		if (nichtErreicht(res)) throw new Error('Server antwortet nicht');

		try {
			const rumpf = await leseRumpf(res);
			if (ueberholt()) return false;
			if (!res.ok) verarbeiteAktionsFehler(res, rumpf, gemerkt); // wirft immer
			verarbeiteAktionsErgebnis(rumpf, reloadProfileCb, gemerkt);
		} catch (e) {
			if (ueberholt()) return false;
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

	// Bucht einen Scan. Das Ergebnis sagt, ob er glatt durchging: Leser geladen, gebucht, für
	// später abgelegt oder nach dem Leeren der Theke ohne Wirkung auf sie.
	/**
	 * @param {Auftrag} auftrag
	 * @returns {Promise<boolean>}
	 */
	async function bucheScan({ q, reloadProfileCb, overrideBlock, confirmedChecklist, absicht }) {
		const standBeimScan = stand;
		const ueberholt = () => stand !== standBeimScan;
		leereHinweiszeilen();
		clearFehlerBanner();

		// Der Schnappschuss entsteht vor allem Weiteren, damit Escape oder „Theke leeren" die
		// Absicht des Scans nicht mehr ändern.
		let eintrag = schnappschuss(q, crypto.randomUUID(), absicht);

		// Der Scan erreicht den Server nicht: Der Schnappschuss geht in die Warteschlange. Nach dem
		// Leeren zählt dabei nur ein Buch, denn ein Ausweis würde gemerkt und bekäme die nächsten.
		/** @param {unknown} grund */
		async function legeAb(grund) {
			if (!ueberholt()) {
				leuchteRand('red');
				return verarbeiteVersandfehler(grund, eintrag);
			}
			// Die Reihe geht weiter (true): Was dort noch wartet, wurde erst nach dem Leeren gescannt.
			const buch = ordneScanEin(q, buchBarcodes.istBuch).art === 'buch';
			return buch ? verarbeiteVersandfehler(grund, eintrag) : true;
		}

		// Steht ein ohne Netz gemerkter Ausweis, gehört ihm das nächste Buch: erst auflösen, dann
		// buchen (merkerAufgeloest). Geht das nicht, trägt der Schnappschuss den Ausweis in die
		// Warteschlange; online ginge das Buch ohne Leser hinaus und wäre am Server eine Rückgabe.
		if (leseMerker()) {
			/** @type {boolean} */
			let aufgeloest;
			try {
				aufgeloest = await merkerAufgeloest(q, reloadProfileCb, ueberholt);
			} catch (e) {
				return legeAb(e);
			}
			if (ueberholt()) return legeAb(new Error('Theke geleert'));
			if (!aufgeloest) return false;
			// Die Auflösung hat die Person geladen: Sie gehört in den Schnappschuss, der Merker nicht
			// mehr; trüge er beides, entschiede beim Nachbuchen die Reihenfolge der Felder.
			if (eintrag.ausweis_barcode && leseLeser()?.id) {
				const ohneMerker = { ...eintrag };
				delete ohneMerker.ausweis_barcode;
				eintrag = { ...ohneMerker, leser_id: leseLeser().id, art: absicht ?? 'ausleihe' };
			}
		}

		// Scheitert der Versand oder arbeitet der Server nicht (nichtErreicht), geht der Schnappschuss
		// in die Warteschlange. Eine Antwort des Servers zeigt ihren Fehler und reiht nicht ein: Nach
		// Ablauf des Idempotenz-Caches (24 h) buchte er sonst zweimal. Rot nur im Fehlerfall, sonst
		// überschriebe es das Grün des Erfolgs und das Orange der Fremdrückgabe.
		let res;
		try {
			res = await apiClient.post('/api/action', {
				query: q,
				active_leser_id: leserFuerVersand(eintrag),
				confirmed_checklist: confirmedChecklist,
				override_block: overrideBlock,
				idempotency_key: eintrag.id
			});
			if (nichtErreicht(res)) throw new Error('Server antwortet nicht');
		} catch (e) {
			return legeAb(e);
		}

		try {
			const rumpf = await leseRumpf(res);
			if (ueberholt()) return true;
			if (!res.ok) verarbeiteAktionsFehler(res, rumpf, q); // wirft immer
			return verarbeiteAktionsErgebnis(rumpf, reloadProfileCb, q, overrideBlock);
		} catch (e) {
			if (ueberholt()) return true;
			leuchteRand('red');
			verarbeiteAntwortfehler(e);
			return false;
		}
	}

	return /** @satisfies {object} */ ({ bucheScan, verwirfLaufendeAntworten });
}
