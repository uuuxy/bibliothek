/**
 * Die Hinweise zu einem eben ausgegebenen Schulbuch an der Theke: ein Buch, das das Kind schon
 * hat, ein Buch, das sonst niemand in der Klasse hat, und eine andere Auflage als in der Klasse.
 * Der Server prüft es nach der Buchung (repository/buch_in_klasse.go, auflagen_klasse.go); die
 * Ausleihe ist gebucht. Die Zeilen stehen über dem Konto, bis die Theke einen anderen Leser lädt
 * (stores/omnibox.svelte.js); kommt das Buch zurück, fallen seine Zeilen weg.
 *
 * Satzzeichen nach M3 (Content design, Grammar and punctuation): ein einzelner Satz ohne Punkt,
 * mehrere Sätze mit Punkt; keine Klammer für einen Nachsatz, kein Gedankenstrich.
 */
import { auflagenHinweisText } from './auflagenText.js';

/** @typedef {{ exemplar: string, art: 'doppelt' | 'klasse' | 'auflagen', text: string }} AusgabeHinweis */

/**
 * Der Satz zu einem Buch, das sonst niemand in der Klasse hat. Die Zahlen nennen, woran es das
 * Programm misst: an den Kindern, die ihre Schulbücher für das Schuljahr schon haben.
 * @param {{ klasse: string, mit_buechern: number, kinder: number }} h
 * @param {string} titel
 */
export function klassenHinweisText(h, titel) {
	return `„${titel}" hat sonst niemand in der ${h.klasse}. ${h.mit_buechern} von ${h.kinder} Kindern haben ihre Schulbücher schon.`;
}

/**
 * Der Satz zu einem Buch, das das Kind schon hat.
 * @param {string} vorname
 * @param {string} titel
 */
export function doppeltHinweisText(vorname, titel) {
	return `${vorname || 'Das Kind'} hat „${titel}" jetzt zweimal`;
}

/**
 * Die Hinweise einer Ausleihe in der Antwort von POST /api/action, in der Reihenfolge, in der die
 * Theke sie zeigt: das sichere Versehen zuerst.
 * @param {any} daten
 * @param {string} [vorname] der Leser an der Theke
 * @returns {AusgabeHinweis[]}
 */
export function ausgabeHinweiseAus(daten, vorname) {
	const exemplar = daten.book?.id ?? '';
	const titel = daten.book?.titel ?? '';
	/** @type {AusgabeHinweis[]} */
	const liste = [];
	if (daten.doppelt_hinweis) {
		liste.push({ exemplar, art: 'doppelt', text: doppeltHinweisText(vorname ?? '', titel) });
	}
	if (daten.klassen_hinweis) {
		liste.push({ exemplar, art: 'klasse', text: klassenHinweisText(daten.klassen_hinweis, titel) });
	}
	if (daten.auflagen_hinweis) {
		liste.push({
			exemplar,
			art: 'auflagen',
			text: auflagenHinweisText(daten.auflagen_hinweis, titel)
		});
	}
	return liste;
}

/**
 * Die Hinweise ohne die eines Exemplars: Es kam zurück oder ging eben neu hinaus.
 * @param {AusgabeHinweis[]} liste
 * @param {string} [exemplar]
 */
export function ohneExemplar(liste, exemplar) {
	return exemplar ? liste.filter((h) => h.exemplar !== exemplar) : liste;
}
