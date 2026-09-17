/**
 * @file exemplarErsatzwert.js
 * Die EINE Regel, wann ein Ersatzwert anzuzeigen ist — geteilt von der Karte und vom
 * Zustands-Dialog.
 *
 * Sie hängt nicht am Betrag, und das ist der Kern: 0,00 € heißt zweierlei. Bei einem
 * Totalschaden (100 % Wertverlust) ist die Null das ERGEBNIS und gehört auf den
 * Bildschirm — gerade dort. Bei einem Titel ohne Preis ist sie dagegen keine Aussage,
 * und „Ersatzwert heute: 0,00 €" wäre eine falsche: Das Buch ist nicht wertlos, sein
 * Preis ist bloß nicht erfasst.
 *
 * Bis zum 17.09.2026 stand hier `ersatzwert > 0` — und die Zeile verschwand ausgerechnet
 * beim Totalschaden. Danach las diese Funktion den deutschen Herleitungssatz des Servers
 * („beginnt mit ‚kein Preis hinterlegt'"). Das war die zweite Fassung desselben Fehlers:
 * ein SATZ als Schnittstelle. Er hält bis zur ersten Umformulierung — der Go-Test wäre
 * dabei rot geworden, jemand hätte ihn nachgezogen, und hier hätte niemand etwas gemerkt.
 * Die Karte zeigte fortan still 0,00 € über einem Buch ohne Preis (Rasterdurchgang
 * 17.09.2026, Frage 3).
 *
 * Seither beantwortet der SERVER die Frage und schickt sie als eigenes Feld mit
 * (`ersatzwert_bekannt`, pkg/ersatzwert.Vorschlag.PreisBekannt). Fehlt das Feld — eine
 * alte Antwort, ein Objekt aus einer anderen Quelle —, gilt „nicht bekannt": Lieber eine
 * Zeile zu wenig als eine falsche Zahl über einem Buch.
 *
 * @param {{ ersatzwert?: number, ersatzwert_bekannt?: boolean }} ex
 * @returns {boolean}
 */
export function ersatzwertBekannt(ex) {
	return ex?.ersatzwert_bekannt === true;
}
