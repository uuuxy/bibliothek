import { normalisiereIsbn } from '../../utils/isbnFormen.js';

// Bestellbedarf und Bestellung stehen in einer Liste: Was schon in der Bestellung steht, fällt
// aus dem Bestellbedarf heraus und trägt dort die Marke „Bestellbedarf N". Gezählt wird am Buch
// wie im Bestellbedarf selbst, also in jeder Auflage; dieselbe Position erkennt der Warenkorb an
// der Kennung oder der ISBN (bestellStore.addToCart).

/**
 * Steht die Position für das Buch dieser Zeile, in irgendeiner Auflage?
 * @param {any} zeile eine Zeile des Bestellbedarfs
 * @param {{ id: string, isbn?: string }} position
 */
export function gehoertZu(zeile, position) {
	const auflagen = [zeile, ...(zeile.auflagen ?? [])];
	if (auflagen.some((a) => a.id === position.id)) return true;
	const isbn = normalisiereIsbn(position.isbn);
	return isbn !== '' && auflagen.some((a) => normalisiereIsbn(a.isbn) === isbn);
}

/**
 * Die Zeilen des Bestellbedarfs, deren Buch noch nicht in der Bestellung steht.
 * @param {any[]} zeilen
 * @param {{ id: string, isbn?: string }[]} positionen
 */
export function offenerBedarf(zeilen, positionen) {
	return zeilen.filter((z) => !positionen.some((p) => gehoertZu(z, p)));
}

/**
 * Die Zeile des Bestellbedarfs zu einer Position, sonst null.
 * @param {any[]} zeilen
 * @param {{ id: string, isbn?: string }} position
 */
export function bedarfZu(zeilen, position) {
	return zeilen.find((z) => gehoertZu(z, position)) ?? null;
}
