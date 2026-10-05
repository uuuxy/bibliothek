// utils/fehlertext.js
// Der Text eines gefangenen Werts.

/**
 * Die Meldung eines Error, sonst der Wert als Zeichenkette. Geworfen wird meist ein Error;
 * Bibliotheken werfen auch Zeichenketten und eigene Ausnahme-Objekte, die sich selbst als
 * Text ausgeben („Name: Meldung"). Diese Annahme steht hier einmal, als Typ.
 *
 * @param {unknown} e
 * @returns {string}
 */
export function fehlertext(e) {
	if (e instanceof Error) return e.message;
	const wert = /** @type {{ toString(): string }} */ (e);
	return String(wert);
}
