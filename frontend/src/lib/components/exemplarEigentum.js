// Das Eigentum eines Exemplars, wie die Exemplarkarte es nennt (docs/OFFEN.md 4.24, Stufe 3).
//
// Wert und Herkunft kommen fertig vom Server (repository.ExemplarTopfSQL und
// ExemplarTopfHerkunftSQL) — hier wird nichts abgeleitet, nur benannt. Dieselben Wörter wie im
// Warenkorb und im Topf-Dialog der Bestellung (bestellungen/mittel.js).

import { MITTEL } from './bestellungen/mittel.js';

/**
 * @typedef {{ eigentum?: string, eigentum_herkunft?: string, littera_eigentumsvermerk?: string }} MitEigentum
 */

/**
 * Die Zeile „Eigentum: …" der Karte, zerlegt in Wert und Herkunft. Leer, wenn der Server kein
 * Eigentum geliefert hat (ältere Antwort) — dann zeigt die Karte keine Zeile statt einer
 * erfundenen.
 * @param {MitEigentum} ex
 * @returns {{ wer: string, herkunft: string } | null}
 */
export function eigentumZeile(ex) {
	const eintrag = MITTEL[/** @type {'land' | 'schultraeger'} */ (ex.eigentum)];
	if (!eintrag) return null;
	const vermerk = ex.littera_eigentumsvermerk ?? '';
	let herkunft;
	switch (ex.eigentum_herkunft) {
		case 'littera':
			herkunft = vermerk ? `laut Littera („${vermerk}“)` : 'laut Littera';
			break;
		case 'hand':
			herkunft = 'von Hand gesetzt';
			break;
		case 'bestellung':
			herkunft = 'aus der Bestellung';
			break;
		default:
			herkunft = ex.eigentum === 'land' ? 'Vorgabe (Lernmittel)' : 'Vorgabe (kein Lernmittel)';
	}
	// Ein Littera-Wortlaut, der das Eigentum nicht gesetzt hat (oder später von Hand geändert
	// wurde), steht dahinter: Die Bücherei sieht, was in Littera stand.
	if (vermerk && ex.eigentum_herkunft !== 'littera') {
		herkunft += ` · Littera: „${vermerk}“`;
	}
	return { wer: eintrag.traeger, herkunft };
}
