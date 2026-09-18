/**
 * Dieselbe ISBN, andere Schreibweise.
 *
 * Ein Strichcode auf dem Buchrücken ist immer eine EAN-13 („9783060130764"). Im Bestand
 * steht dieselbe ISBN je nach Herkunft anders: mit Bindestrichen aus einer Katalogabfrage,
 * mit Leerzeichen aus einer Liste, und bei Altbeständen zehnstellig — die Littera-Übernahme
 * behält eine gültige ISBN-10 unverändert bei (internal/uebernahme/isbn.go). Ein
 * Zeichenvergleich findet davon nur den Zufallstreffer.
 *
 * Nicht geraten wird dabei nichts: Zwischen ISBN-10 und ISBN-13 mit 978-Präfix liegt eine
 * feste Rechnung — gleicher Kern, neu berechnete Prüfziffer. Beide Formen bezeichnen
 * dasselbe Buch.
 */

/** Nur die bedeutungstragenden Zeichen, Prüfzeichen groß. @param {unknown} roh */
export function normalisiereIsbn(roh) {
	return String(roh ?? '')
		.replace(/[^0-9xX]/g, '')
		.toUpperCase();
}

/** @param {string} neun Die ersten neun Ziffern einer ISBN-10 */
function pruefzeichen10(neun) {
	let summe = 0;
	for (let i = 0; i < 9; i++) summe += Number(neun[i]) * (10 - i);
	const rest = (11 - (summe % 11)) % 11;
	return rest === 10 ? 'X' : String(rest);
}

/** @param {string} zwoelf Die ersten zwölf Ziffern einer ISBN-13 */
function pruefziffer13(zwoelf) {
	let summe = 0;
	for (let i = 0; i < 12; i++) summe += Number(zwoelf[i]) * (i % 2 === 0 ? 1 : 3);
	return String((10 - (summe % 10)) % 10);
}

/**
 * Alle Schreibweisen, unter denen dieselbe ISBN im Bestand stehen kann — normalisiert.
 * Leere Liste heißt: Das ist keine ISBN, es bleibt bei der gewöhnlichen Textsuche.
 *
 * Bewusst OHNE Prüfziffernkontrolle der Eingabe: Ein falsch gelesener Code trifft dann
 * eben nichts. Ein Bestand mit einer krummen, aber eingetippten ISBN wäre dagegen
 * unauffindbar, wenn diese Funktion ihn vorher aussortierte.
 *
 * @param {string} roh
 * @returns {string[]}
 */
export function isbnFormen(roh) {
	const n = normalisiereIsbn(roh);
	if (n.length === 10) return [n, '978' + n.slice(0, 9) + pruefziffer13('978' + n.slice(0, 9))];
	if (n.length === 13 && (n.startsWith('978') || n.startsWith('979'))) {
		// Nur 978 lässt sich zurückrechnen: Der 979-Bereich hat keine ISBN-10-Entsprechung.
		if (!n.startsWith('978')) return [n];
		const kern = n.slice(3, 12);
		return [n, kern + pruefzeichen10(kern)];
	}
	return [];
}
