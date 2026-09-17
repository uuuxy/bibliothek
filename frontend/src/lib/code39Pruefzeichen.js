/**
 * Das Prüfzeichen, das bis zum 17.09.2026 auf jedem Ausweis und Buchetikett stand — der
 * Zwilling von pkg/code39/pruefzeichen.go für die Theke OHNE NETZ.
 *
 * Die Geschichte steht ausführlich im Go-Paket. Kurz: Der Drucker hängte ein Mod-43-
 * Prüfzeichen an, das IN den Strichcode-Daten steht. Unter der Karte steht „A-10003", das
 * Lesegerät liefert „A-100037". Gedruckt wird seither Code 128 ohne Prüfzeichen; die
 * Karten von vorher sind im Umlauf und sollen weiter funktionieren.
 *
 * Beide Seiten lesen dieselben Prüffälle (code39.faelle.json) — rechnen sie verschieden,
 * wird der Go- oder der Vitest rot. Dasselbe Muster wie bei litteraEtikett.
 */

/** Die Wertetabelle von Code 39: Index = Wert des Zeichens. '*' ist nur Start/Stopp. */
const ZEICHENSATZ = '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ-. $/+%';

/**
 * Das Mod-43-Prüfzeichen eines Inhalts.
 *
 * @param {string} inhalt
 * @returns {string | null} das Zeichen, oder null bei einem Zeichen, das Code 39 nicht kennt
 */
export function pruefzeichen(inhalt) {
	let summe = 0;
	for (const z of inhalt) {
		const wert = ZEICHENSATZ.indexOf(z);
		if (wert < 0) return null;
		summe += wert;
	}
	return ZEICHENSATZ[summe % 43];
}

/**
 * Trennt ein angehängtes Mod-43-Prüfzeichen ab.
 *
 * Gibt `null` zurück, wenn das letzte Zeichen NICHT das Prüfzeichen des Restes ist — der
 * Aufrufer behält dann seinen Scan.
 *
 * WICHTIG: Das ist kein Beweis, dass ein Prüfzeichen gemeint war. Bei 43 möglichen
 * Zeichen sieht im Schnitt jeder 43. gültige Code zufällig so aus. Nur als ZWEITER
 * Versuch benutzen, nachdem der Scan so, wie er kam, nichts gefunden hat.
 *
 * @param {string} scan
 * @returns {string | null}
 */
export function ohnePruefzeichen(scan) {
	if (typeof scan !== 'string' || scan.length < 3) return null;
	const kern = scan.slice(0, -1);
	const erwartet = pruefzeichen(kern);
	if (erwartet === null || erwartet !== scan[scan.length - 1]) return null;
	return kern;
}
