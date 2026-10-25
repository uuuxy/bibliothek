/**
 * @file exemplarErsatzwert.js
 * Die eine Regel, wann ein Ersatzwert anzuzeigen ist, geteilt von der Karte und vom
 * Zustands-Dialog.
 *
 * Sie hängt nicht am Betrag, denn 0,00 € heißt zweierlei. Bei einem Totalschaden (100 %
 * Wertverlust) ist die Null das Ergebnis und gehört auf den Bildschirm. Bei einem Titel ohne
 * Preis ist sie keine Aussage, und „Ersatzwert heute: 0,00 €" wäre eine falsche: Das Buch
 * ist nicht wertlos, sein Preis ist bloß nicht erfasst.
 *
 * Deshalb beantwortet der Server die Frage und schickt sie als eigenes Feld mit
 * (`ersatzwert_bekannt`, pkg/ersatzwert.Vorschlag.HatPreis), statt dass hier ein Satz der
 * Herleitung gelesen wird, der bei der ersten Umformulierung still bricht. Fehlt das Feld
 * (eine alte Antwort, ein Objekt aus einer anderen Quelle), gilt „nicht bekannt": Lieber
 * eine Zeile zu wenig als eine falsche Zahl über einem Buch.
 *
 * @param {{ ersatzwert?: number, ersatzwert_bekannt?: boolean }} ex
 * @returns {boolean}
 */
export function ersatzwertBekannt(ex) {
	return ex?.ersatzwert_bekannt === true;
}
