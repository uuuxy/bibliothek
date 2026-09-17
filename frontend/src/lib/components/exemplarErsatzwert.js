/**
 * @file exemplarErsatzwert.js
 * Die EINE Regel, wann ein Ersatzwert anzuzeigen ist — geteilt von der Karte und vom
 * Zustands-Dialog.
 *
 * Sie hängt nicht am Betrag, und das ist der Kern: 0,00 € heißt zweierlei. Bei einem
 * Totalschaden (100 % Wertverlust) ist die Null das ERGEBNIS und gehört auf den
 * Bildschirm — gerade dort. Bei einem Titel ohne Preis ist sie dagegen keine Aussage,
 * und „Ersatzwert heute: 0,00 €" wäre eine falsche: Das Buch ist nicht wertlos, sein
 * Preis ist bloß nicht erfasst. Der Server unterscheidet beides in der Herleitung.
 *
 * Bis zum 17.09.2026 stand hier `ersatzwert > 0` — und die Zeile verschwand ausgerechnet
 * beim Totalschaden.
 *
 * @param {{ ersatzwert?: number, ersatzwert_herleitung?: string }} ex
 * @returns {boolean}
 */
export function ersatzwertBekannt(ex) {
	const satz = ex?.ersatzwert_herleitung ?? '';
	if (!satz) return false;
	return !satz.startsWith('kein Preis hinterlegt');
}
