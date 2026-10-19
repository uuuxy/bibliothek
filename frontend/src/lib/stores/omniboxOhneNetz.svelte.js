// Was die Theke ohne Netz tut: den Scan als Schnappschuss in die Warteschlange legen, einen
// Ausweis nur als Nummer merken und einen Scan verwerfen, der ohne Netz nicht geht. Den Merker
// hält dieser Teil. Nur der Omnibox-Store lädt die Datei (frontend-hygiene-omnibox-teile.test.js);
// den Leser an der Theke und die Rückmeldung behält er und reicht sie als Funktionen herein.

import { spieleErfolgston, spieleFehlerton } from '../audio.js';
import { enqueueOfflineAction } from '../offlineWarteschlange.js';
import { offlineSync } from './offlineSync.svelte.js';
import { buchBarcodes } from './buchBarcodes.svelte.js';
import { ordneScanEin } from '../scanEinordnen.js';

/**
 * @param {{
 *   leseLeser: () => any,
 *   beendeSchnellrueckgabe: () => void,
 *   entferneLeser: () => void,
 *   leuchteBildschirm: (type: string) => void,
 *   zeigeFehlerBanner: (message: string) => void,
 *   melde: (message: string, type?: import('./snackbars.svelte.js').ToastTyp) => void
 * }} theke
 */
export function erzeugeOmniboxOhneNetz({
	leseLeser,
	beendeSchnellrueckgabe,
	entferneLeser,
	leuchteBildschirm,
	zeigeFehlerBanner,
	melde
}) {
	// Der ohne Netz gescannte Ausweis, nur seine Nummer: Auflösen kann ihn nur der Server, bis
	// dahin tragen die folgenden Bücher sie mit (`ausweis_barcode` in der Warteschlange).
	let offlineAusweis = $state('');

	// Der Schnappschuss des Scans, bevor die Anfrage hinausgeht: Escape oder „Theke leeren"
	// während einer hängenden Anfrage ändern weder Person noch Absicht.
	//
	// Ohne Vorgabe ist der Scan eine Ausleihe, wenn ein Leser geladen oder ein Ausweis gemerkt
	// ist, sonst eine Rückgabe; „Buch zurückgeben" in der Akte gibt die Rückgabe vor.
	/**
	 * @param {string} q
	 * @param {string} idempotencyKey
	 * @param {'ausleihe' | 'rueckgabe' | null} absicht
	 * @returns {import('../offlineWarteschlange.js').OfflineEintrag}
	 */
	function schnappschuss(q, idempotencyKey, absicht) {
		const aktiverLeser = leseLeser();
		// Ohne geladenen Leser, aber mit gemerktem Ausweis geht das Buch an die Person dahinter.
		const merker = !aktiverLeser?.id && offlineAusweis ? offlineAusweis : '';
		return {
			id: idempotencyKey,
			art: absicht ?? (aktiverLeser?.id || merker ? 'ausleihe' : 'rueckgabe'),
			barcode: q,
			leser_id: aktiverLeser?.id ?? null,
			...(merker ? { ausweis_barcode: merker } : {}),
			gescannt_am: Date.now(),
			// Die Wanduhr springt gern, wenn das Netz zurückkommt, performance.now() nicht. Daraus
			// rechnet das Nachbuchen den Zeitpunkt des Scans neu (offlineWarteschlange.js).
			mono: Math.round(performance.now()),
			ursprung: Math.round(performance.timeOrigin ?? 0)
		};
	}

	// Legt einen Scan ohne Netz ab. Eingeordnet wird wie am Server (scanEinordnen.js), mit der
	// Buchliste dieses Rechners; abgelegt wird die Nummer aus der Einordnung, weil ein
	// Littera-Etikett eine EAN-13 trägt und der Server nur die Nummer darin kennt.
	/**
	 * @param {import('../offlineWarteschlange.js').OfflineEintrag} eintrag
	 * @returns {Promise<boolean>} true: gespeichert oder als Ausweis gemerkt
	 */
	async function speichereOfflineAktion(eintrag) {
		const einordnung = ordneScanEin(eintrag.barcode, buchBarcodes.istBuch);
		if (einordnung.art === 'ausweis') {
			merkeOfflineAusweis(einordnung.nummer);
			return true;
		}
		if (einordnung.art !== 'buch') {
			// Nach einem unklaren Scan fällt der Merker: Das nächste Buch soll keiner Person
			// zugeschrieben werden, bei der niemand mehr sicher ist.
			if (einordnung.art === 'unklar') offlineAusweis = '';
			verwirfOfflineScan(einordnung);
			return false;
		}
		eintrag = { ...eintrag, barcode: einordnung.nummer };
		try {
			await enqueueOfflineAction(eintrag);
		} catch (err) {
			// Laut, nicht „gespeichert": Ohne Warteschlange ist der Scan verloren, das Buch muss
			// zurück ins Regal oder auf den Zettel.
			console.error('Offline-Eintrag nicht gespeichert:', err);
			leuchteBildschirm('error');
			spieleFehlerton();
			zeigeFehlerBanner(
				`NICHT gespeichert: „${eintrag.barcode}“ konnte nicht auf diesem Rechner abgelegt werden — Buch zurücklegen und den Vorgang notieren.`
			);
			offlineSync.zaehleNeu();
			return false;
		}
		offlineSync.zaehleNeu();
		leuchteBildschirm('warning');
		spieleErfolgston();
		melde(`Offline: Aktion für „${eintrag.barcode}“ gespeichert.`, 'warning');
		return true;
	}

	// Ohne Netz merkt sich die Theke vom Ausweis nur die Nummer, weil auf dem Rechner keine
	// Personendaten liegen sollen. Der Server löst sie beim Nachbuchen auf.
	/** @param {string} nummer */
	function merkeOfflineAusweis(nummer) {
		beendeSchnellrueckgabe();
		offlineAusweis = nummer;
		// Die zuvor geladene Person weicht: Sonst zeigte die Theke einen Namen, während die
		// folgenden Bücher an eine andere Person gingen.
		entferneLeser();
		leuchteBildschirm('warning');
		spieleErfolgston();
		melde(
			`Ohne Netz gemerkt: Ausweis „${nummer}". Die folgenden Bücher werden ihm zugeordnet.`,
			'warning'
		);
	}

	/** @param {import('../scanEinordnen.js').ScanEinordnung} einordnung */
	function verwirfOfflineScan(einordnung) {
		// Den Ausweis merkt speichereOfflineAktion vorher; hier kommt er nicht an.
		const meldungen = {
			// Geräte gehen ohne Netz nicht: Sie hängen an einer Checkliste, die es nur online gibt.
			geraet:
				`Geräte lassen sich ohne Netz nicht ausgeben oder zurücknehmen — die Checkliste dazu gibt es nur online. ` +
				`\u201e${einordnung.nummer}\u201c wurde NICHT gebucht.`,
			// Ohne Netz gibt es keine Namenssuche, weil auf dem Rechner der Theke keine
			// Personendaten liegen sollen.
			suche:
				`Ohne Netz lässt sich nicht nach Namen suchen — auf diesem Rechner stehen keine Personendaten. ` +
				`Bitte den Ausweis scannen oder den Vorgang notieren.`,
			// Die sichere Seite: Wer hier rät, schreibt das nächste Buch einer fremden Person zu.
			unklar:
				`\u201e${einordnung.nummer}\u201c ist ohne Netz nicht eindeutig — die Nummer steht nicht in der Buchliste dieses Rechners. ` +
				`NICHT gebucht; bitte notieren.`
		};
		leuchteBildschirm('error');
		spieleFehlerton();
		zeigeFehlerBanner(meldungen[einordnung.art] ?? 'Dieser Scan wurde ohne Netz NICHT gebucht.');
	}

	// Der Versand ist gescheitert (Netzfehler, Timeout, CSRF-Bootstrap ohne Netz): Der
	// Server hat den Scan nicht gesehen, der Schnappschuss geht in die Warteschlange.
	/**
	 * @param {unknown} e @param {import('../offlineWarteschlange.js').OfflineEintrag} eintrag
	 * @returns {Promise<boolean>} true: gespeichert oder als Ausweis gemerkt
	 */
	function verarbeiteVersandfehler(e, eintrag) {
		console.warn('Scan nicht zugestellt, wird eingereiht:', e);
		return speichereOfflineAktion(eintrag);
	}

	return /** @satisfies {object} */ ({
		get offlineAusweis() {
			return offlineAusweis;
		},
		set offlineAusweis(v) {
			offlineAusweis = v;
		},
		schnappschuss,
		verarbeiteVersandfehler
	});
}
