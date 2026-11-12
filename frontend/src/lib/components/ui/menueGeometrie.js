// Lage eines Menüs am Knopf — dieselbe Regel wie selectGeometrie.berechneBox, nur mit
// gemessener statt gerechneter Größe: Breite und Höhe folgen den Einträgen (Menue.svelte)
// und stehen erst nach dem Rendern fest.
//
// Rechts- oder linksbündig am Knopf, nach unten, wenn Platz ist — sonst nach oben, wenn
// dort mehr Platz ist. Immer 8 px vom Fensterrand weg.

/**
 * Ein Eintrag des Menüs. Hier und nicht im Bauteil, weil ein .svelte seine JSDoc-Typen
 * nicht exportiert — Aufrufer (Mahnwesen, Ausweis) holen ihn von hier.
 * @typedef {{ id: string, text: string, icon?: any, disabled?: boolean, trennerDavor?: boolean }} Eintrag
 */

const RAND = 8;
const ABSTAND = 4;

/**
 * @param {{ left: number, right: number, top: number, bottom: number }} anker Rechteck des Knopfs
 * @param {number} hoehe Gemessene Höhe des Menüs
 * @param {number} breite Gemessene Breite des Menüs
 * @param {'rechts' | 'links'} ausrichtung Welche Kante des Knopfs das Menü trifft
 * @param {{ breite: number, hoehe: number }} fenster
 * @returns {{ left: number, top: number }}
 */
export function berechneMenueBox(anker, hoehe, breite, ausrichtung, fenster) {
	const untenPlatz = fenster.hoehe - anker.bottom;
	const nachOben = untenPlatz < hoehe && anker.top > untenPlatz;
	const links = ausrichtung === 'links' ? anker.left : anker.right - breite;
	return {
		left: Math.max(RAND, Math.min(links, fenster.breite - breite - RAND)),
		top: nachOben ? Math.max(RAND, anker.top - hoehe - ABSTAND) : anker.bottom + ABSTAND
	};
}

/**
 * Lage einer Vorschau neben dem markierten Menüeintrag, wie ein Untermenü: oben bündig mit dem
 * Eintrag an der hinteren Kante der Menüfläche (material-web, md-sub-menu: anchor-corner
 * „start-end“, menu-corner „start-start“). Reicht rechts der Platz nicht, liegt sie links an der
 * Menüfläche; senkrecht bleibt sie im Fenster.
 *
 * @param {{ top: number }} eintrag Rechteck des markierten Eintrags
 * @param {{ left: number, right: number }} menue Rechteck der Menüfläche
 * @param {{ breite: number, hoehe: number }} flaeche Gemessene Maße der Vorschau
 * @param {{ breite: number, hoehe: number }} fenster
 * @returns {{ left: number, top: number }}
 */
export function berechneVorschauBox(eintrag, menue, flaeche, fenster) {
	const passtRechts = menue.right + flaeche.breite <= fenster.breite - RAND;
	return {
		left: passtRechts ? menue.right : Math.max(RAND, menue.left - flaeche.breite),
		top: Math.max(RAND, Math.min(eintrag.top, fenster.hoehe - flaeche.hoehe - RAND))
	};
}
