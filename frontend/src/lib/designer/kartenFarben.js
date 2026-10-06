/**
 * @file kartenFarben.js
 * Die festen Farben der Ausweiskarte: ihre Hintergründe und die Nummer unter dem Strichcode.
 *
 * Die Karte wird gedruckt. Ihre Farben sind Werte des Entwurfs wie die Farbe einer Fläche
 * oder eines Textes, keine Rollen der Oberfläche: Ein anderes Farbschema der Anwendung
 * darf den Ausweis nicht umfärben. Im zentral gespeicherten Entwurf steht die Kennung des
 * Hintergrunds (`value`); die Werte dazu stehen nur hier.
 */

/** Die Nummer unter dem Strichcode, auf der Leinwand wie auf dem Papier. */
export const NUMMER_FARBE = '#2f3033';

/** @typedef {{ value: string, name: string, flaeche: string, schrift: string }} KartenHintergrund */

/** @type {KartenHintergrund[]} */
export const KARTEN_HINTERGRUENDE = [
	{ value: 'weiss', name: 'Weiß', flaeche: '#ffffff', schrift: '#000000' },
	{ value: 'grau', name: 'Grau', flaeche: '#f1f0f4', schrift: '#1a1c1e' },
	{ value: 'smaragd', name: 'Smaragd', flaeche: '#b7f5b9', schrift: '#002c22' },
	{ value: 'blau', name: 'Blau', flaeche: '#d0e4ff', schrift: '#1e1a4d' },
	{ value: 'bernstein', name: 'Bernstein', flaeche: '#ffdcc2', schrift: '#461901' },
	{
		value: 'waldgruen',
		name: 'Waldgrün',
		flaeche: 'linear-gradient(to right top in oklab, #16330a 0%, #2d5a12 50%, #3f7418 100%)',
		schrift: '#ffffff'
	}
];

export const HINTERGRUND_VORDERSEITE = 'weiss';
export const HINTERGRUND_RUECKSEITE = 'grau';

/**
 * Der Stil der Kartenfläche zu einer Kennung. Eine unbekannte Kennung zeichnet die weiße
 * Karte.
 * @param {string | null | undefined} kennung
 */
export function kartenStil(kennung) {
	const h = KARTEN_HINTERGRUENDE.find((k) => k.value === kennung) ?? KARTEN_HINTERGRUENDE[0];
	return `background: ${h.flaeche}; color: ${h.schrift};`;
}

// Merkmale der Klassenlisten, unter denen der Hintergrund früher gespeichert wurde: der
// Anfang der Klasse, die die Fläche färbte. Die weiße Karte trägt keines, ebenso die
// älteren, fast weißen Fassungen von Grau, Smaragd und Blau; sie sind der Rückfall.
const ALTE_MERKMALE = [
	['from-emerald-1', 'smaragd'],
	['from-sky-1', 'blau'],
	['from-amber-1', 'bernstein'],
	['16330a', 'waldgruen'],
	['bg-slate-1', 'grau']
];

/**
 * Liest den gespeicherten Hintergrund einer Kartenseite. Ein Entwurf aus der Zeit der
 * Klassenlisten wird an seinem Merkmal erkannt und auf die Kennung übersetzt; was sich
 * nicht zuordnen lässt, ist die weiße Karte.
 * @param {unknown} gespeichert
 * @returns {string}
 */
export function heileKartenHintergrund(gespeichert) {
	const wert = typeof gespeichert === 'string' ? gespeichert.trim() : '';
	if (KARTEN_HINTERGRUENDE.some((k) => k.value === wert)) return wert;
	const treffer = ALTE_MERKMALE.find(([merkmal]) => wert.includes(merkmal));
	return treffer ? treffer[1] : HINTERGRUND_VORDERSEITE;
}
