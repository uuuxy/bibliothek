import { isbnFormen, normalisiereIsbn } from '../../utils/isbnFormen.js';

/**
 * Der Schnellfilter des Bestellbedarfs: Titel, ISBN, Verlag und Signatur als Teilstring. Die
 * ISBN jeder Auflage zählt mit: Wer die alte Auflage scannt, findet die Zeile des Buchs, die
 * die neueste zeigt. Eine ISBN trifft auch in der anderen Länge und mit Bindestrichen — getippt
 * wird die zehnstellige vom Titelblatt, der Katalog führt die dreizehnstellige.
 *
 * @param {any[]} zeilen
 * @param {string} filter
 * @returns {any[]}
 */
export function filtereBedarf(zeilen, filter) {
	const q = filter.trim().toLowerCase();
	if (!q) return zeilen;
	const formen = isbnFormen(q);
	const isbnTrifft = (/** @type {string | undefined} */ isbn) =>
		(isbn || '').toLowerCase().includes(q) ||
		(formen.length > 0 && formen.includes(normalisiereIsbn(isbn)));
	return zeilen.filter(
		(r) =>
			(r.titel || '').toLowerCase().includes(q) ||
			isbnTrifft(r.isbn) ||
			(r.auflagen ?? []).some((/** @type {any} */ a) => isbnTrifft(a.isbn)) ||
			(r.verlag || '').toLowerCase().includes(q) ||
			(r.signatur || '').toLowerCase().includes(q)
	);
}
