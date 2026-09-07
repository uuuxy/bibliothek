// Lage eines Menüs am Knopf — dieselbe Regel wie selectGeometrie.berechneBox, nur mit
// gemessener statt gerechneter Höhe: Seit das Menü einen Kopf tragen darf (Auswahlfeld im
// Mahnwesen), steht seine Höhe erst nach dem Rendern fest.
//
// Rechts- oder linksbündig am Knopf, nach unten, wenn Platz ist — sonst nach oben, wenn
// dort mehr Platz ist. Immer 8 px vom Fensterrand weg.

/**
 * Ein Eintrag des Menüs. Hier und nicht im Bauteil, weil ein .svelte seine JSDoc-Typen
 * nicht exportiert — Aufrufer (Mahnwesen, Ausweis) holen ihn von hier.
 * @typedef {{ id: string, text: string, icon?: any, disabled?: boolean, trennerDavor?: boolean, ueberschriftDavor?: string }} Eintrag
 */

const RAND = 8;
const ABSTAND = 4;

/**
 * @param {{ left: number, right: number, top: number, bottom: number }} anker Rechteck des Knopfs
 * @param {number} hoehe Gemessene Höhe des Menüs
 * @param {number} breite Breite des Menüs
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
