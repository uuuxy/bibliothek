// Die Reihe der Scans: der Doppelscan, die wartenden Scans und der Halt nach einem Scan, der
// nicht glatt durchging. Nur der Omnibox-Store lädt diese Datei
// (frontend-hygiene-omnibox-teile.test.js); die Buchung, die Rückfrage und die Rückmeldung
// reicht er als Funktionen herein.

import { spieleFehlerton } from '../audio.js';

/** @typedef {import('./omniboxBuchung.svelte.js').Auftrag} Auftrag */

/**
 * @param {{
 *   bucheScan: (auftrag: Auftrag) => Promise<boolean>,
 *   rueckfrageOffen: () => boolean,
 *   ungespeichert: () => boolean,
 *   leereEingabe: () => void,
 *   verarbeiteAntwortfehler: (e: unknown) => void,
 *   scanfeldWiederScharfstellen: () => void,
 *   leuchteBildschirm: (type: string) => void,
 *   leuchteRand: (farbe: string) => void,
 *   zeigeFehlerBanner: (meldung: string) => void,
 *   nenneNichtGebucht: (nummern: string[]) => void
 * }} theke
 */
export function erzeugeOmniboxReihe({
	bucheScan,
	rueckfrageOffen,
	ungespeichert,
	leereEingabe,
	verarbeiteAntwortfehler,
	scanfeldWiederScharfstellen,
	leuchteBildschirm,
	leuchteRand,
	zeigeFehlerBanner,
	nenneNichtGebucht
}) {
	// Die Reihe der Scans. Ein Handscanner wartet nicht auf die Antwort: Ein Scan, der
	// eintrifft, solange eine Buchung läuft, wartet hier und wird danach gebucht. Verworfen
	// bliebe er unbemerkt, in der Schnellrückgabe bliebe das Buch verliehen.
	let buchungLaeuft = false;
	/** @type {Auftrag[]} */
	const wartend = [];
	// Die Nummer, die gerade gebucht wird.
	let laufendeNummer = '';
	// Löst sich auf, sobald kein Scan mehr unterwegs ist oder wartet.
	let reiheGebucht = Promise.resolve();

	// Die Reihe liegt nur im Speicher. Ein Neuladen oder Schließen, solange sie bucht (ein Scan
	// wartet nur, solange ein anderer gebucht wird), verwürfe die Scans ohne Meldung; der Browser
	// fragt deshalb vorher, wie bei der Warteschlange ohne Netz (offlineSync.svelte.js).
	if (typeof window !== 'undefined') {
		window.addEventListener('beforeunload', (e) => {
			if (buchungLaeuft) e.preventDefault();
		});
	}

	// Ein Scan kommt in die Reihe: Läuft keine Buchung, beginnt sie mit ihm, sonst wartet er.
	/** @param {Auftrag} auftrag */
	function reiheEin(auftrag) {
		// Hält die Akte Ungespeichertes („Stammdaten bearbeiten"), bucht die Theke nichts: Ob ein
		// Scan ein Ausweis ist, sagt erst der Server, und ein Ausweis verwürfe die Eingaben.
		if (ungespeichert()) {
			leuchteBildschirm('error');
			spieleFehlerton();
			zeigeFehlerBanner(
				'Nicht gebucht: Erst die Eingaben in der Maske speichern oder verwerfen, dann erneut scannen.'
			);
			return reiheGebucht;
		}
		if (buchungLaeuft) {
			// Dieselbe Nummer, bevor sie gebucht ist, ist der Doppelscan: Er fällt weg. Gebucht
			// gäbe er das eben geliehene Buch gleich wieder zurück.
			if (!istUnterwegs(auftrag.q)) wartend.push(auftrag);
			return reiheGebucht;
		}
		buchungLaeuft = true;
		reiheGebucht = bucheReihe(auftrag);
		return reiheGebucht;
	}

	/** Wird die Nummer gerade gebucht oder wartet sie? @param {string} q */
	const istUnterwegs = (q) => q === laufendeNummer || wartend.some((w) => w.q === q);

	// Bucht den Scan und danach, der Reihe nach, was inzwischen gescannt wurde. Jeder Scan
	// findet vor, was der vorige hinterlassen hat: Nach einem Ausweis gehen die Bücher an
	// diesen Leser.
	//
	// Weiter geht es nur nach einer glatten Buchung. Scheitert ein Scan, kann er ein Ausweis
	// gewesen sein, und die folgenden Bücher gingen an den Leser davor; nach einer Rückfrage
	// oder bei Ungespeichertem entscheidet ein Mensch, was gilt. Was dann noch wartet, wird
	// nicht gebucht und genannt.
	/** @param {Auftrag} erster */
	async function bucheReihe(erster) {
		const haelt = () => rueckfrageOffen() || ungespeichert();
		try {
			/** @type {Auftrag | undefined} */
			let auftrag = erster;
			while (auftrag) {
				laufendeNummer = auftrag.q;
				const glatt = await bucheScan(auftrag);
				laufendeNummer = '';
				if (haelt()) {
					// Die Rückfrage hat jetzt die Tastatur. Was vom nächsten Scan schon im Feld
					// steht, stünde sonst vor dem übernächsten.
					leereEingabe();
				}
				if (!glatt || haelt()) break;
				auftrag = wartend.shift();
			}
		} catch (e) {
			// Unerwartet abgebrochen: Ob der laufende Scan gebucht ist, weiß hier niemand.
			console.error('Buchung abgebrochen:', e);
			leuchteRand('red');
			verarbeiteAntwortfehler(e);
		} finally {
			laufendeNummer = '';
			verwirfWartende();
			buchungLaeuft = false;
			scanfeldWiederScharfstellen();
		}
	}

	// Die wartenden Scans werden nicht gebucht. Das Banner nennt sie, bis der nächste Scan
	// kommt; der Ton sagt es dem, der nicht hinsieht.
	function verwirfWartende() {
		if (wartend.length === 0) return;
		const nummern = wartend.splice(0).map((w) => w.q);
		leuchteBildschirm('error');
		spieleFehlerton();
		leuchteRand('red');
		nenneNichtGebucht(nummern);
	}

	return /** @satisfies {object} */ ({ reiheEin, verwirfWartende });
}
