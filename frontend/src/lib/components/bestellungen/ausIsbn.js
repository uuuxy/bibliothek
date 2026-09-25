/**
 * Die Tür der Titelsuche beim Bestellen: POST /api/buecher/aus-isbn — aus dem Katalog oder neu
 * aus der DNB. Zwei Stellen fragen sie, die Titelsuche (OrderSearch) und „Neue Auflage
 * bestellen" (NeueAuflageDialog); beide lesen die Antwort nach derselben Regel.
 *
 * Steht die ISBN nur in der anderen Länge im Katalog (ISBN-10 ↔ ISBN-13, docs/OFFEN.md 4.18
 * Stufe 4), legt die Tür nichts an: Die Antwort trägt keine titel_id, dafür andere_form. Dann
 * fragt AndereIsbnFormWahl, und „Neu anlegen" schickt dieselbe ISBN mit neu_anlegen.
 */

/**
 * Der Rumpf der Anfrage.
 * @param {string} isbn
 * @param {boolean} [neuAnlegen] true nach „Neu anlegen" in der Frage — dann sucht die Tür die
 *   andere Form nicht mehr.
 */
export function ausIsbnRumpf(isbn, neuAnlegen = false) {
	return neuAnlegen ? { isbn, neu_anlegen: true } : { isbn };
}

/**
 * Fragt die Antwort, ob der Titel unter der anderen Form gemeint ist? Dann ist nichts angelegt.
 * @param {any} antwort
 */
export function istAndereFormFrage(antwort) {
	return Boolean(antwort?.andere_form) && !antwort?.titel_id;
}
