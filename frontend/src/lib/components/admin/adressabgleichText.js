// Die Sätze zum Abgleich der Adressen mit dem Webmail, getrennt vom Abschnitt, damit sie ohne
// Oberfläche prüfbar sind (adressabgleichText.test.js).
import { formatZeitpunkt, verbindeSaetze } from '../../utils/format.js';

/**
 * @typedef {{ zeitpunkt: string, eintraege: number, uebernommen: number, uebergangen: number, fehler?: string }} AbgleichStand
 */

/**
 * @param {number} n
 * @param {string} eins
 * @param {string} viele
 */
function anzahl(n, eins, viele) {
	return `${n} ${n === 1 ? eins : viele}`;
}

/**
 * Die Zeile zum letzten Lauf, der eintragen durfte (von selbst oder auf Knopfdruck).
 * @param {AbgleichStand | null | undefined} stand
 */
export function letzterLaufText(stand) {
	if (!stand) return 'Noch kein Abgleich gelaufen';
	const wann = `Zuletzt am ${formatZeitpunkt(stand.zeitpunkt)}`;
	if (stand.fehler) return verbindeSaetze(`${wann}: gescheitert`, stand.fehler);
	return `${wann}: ${ergebnisZahlen(stand, false)}`;
}

/**
 * Die Zahlen eines Laufs in einem Satz; der Probelauf spricht im Konjunktiv.
 * @param {AbgleichStand} stand
 * @param {boolean} probelauf
 */
export function ergebnisZahlen(stand, probelauf) {
	const uebernommen = probelauf
		? `${anzahl(stand.uebernommen, 'Konto bekäme', 'Konten bekämen')} eine Adresse`
		: `${anzahl(stand.uebernommen, 'Adresse', 'Adressen')} übernommen`;
	return `${uebernommen}, ${anzahl(stand.uebergangen, 'Konto', 'Konten')} ohne eindeutigen Treffer`;
}
