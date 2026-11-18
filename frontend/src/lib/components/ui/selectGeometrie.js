/** Geometrie des M3-Auswahlfelds — wo die Menüfläche aufgeht.
 *
 *  Eigene Datei, weil das reine Rechnerei ist: keine Svelte-Zustände, dafür
 *  testbar ohne Browser. Die Liste liegt `position: fixed` (sonst schneidet
 *  ein overflow-Container sie ab), also sind das Fensterkoordinaten aus
 *  getBoundingClientRect. */

/** Zeilenhöhe eines Menüeintrags in px (Material 3: 48). */
export const ZEILE = 48;
/** Höchsthöhe der Liste in px — darüber wird gescrollt (max-h-80). */
export const LISTE_MAX = 320;

/**
 * @param {HTMLElement} ausloeser
 * @param {number} anzahl Zahl der Einträge
 * @returns {{ left: number, top: number, breite: number }}
 */
export function berechneBox(ausloeser, anzahl) {
	return lageUnter(ausloeser, Math.min(LISTE_MAX, anzahl * ZEILE + 16));
}

/**
 * Wo eine Fläche der Höhe hoehe am Auslöser aufgeht. Mit breite (der Kalender ist breiter als
 * sein Feld) bleibt sie auch rechts im Fenster.
 * @param {HTMLElement} ausloeser
 * @param {number} hoehe
 * @param {number} [breite]
 * @returns {{ left: number, top: number, breite: number }}
 */
export function lageUnter(ausloeser, hoehe, breite) {
	const r = ausloeser.getBoundingClientRect();
	// Nach unten, wenn Platz ist; sonst nach oben. Sonst hinge die Liste bei
	// Feldern am unteren Bildrand unerreichbar außerhalb des Fensters.
	const untenPlatz = window.innerHeight - r.bottom;
	const nachOben = untenPlatz < hoehe && r.top > untenPlatz;
	// Passt sie auf keiner Seite, rückt sie ins Fenster und deckt das Feld zum Teil, wie das
	// Menü (menueGeometrie.js); sonst lägen die letzten Einträge unter dem Fensterrand.
	const unten = Math.min(r.bottom + 4, Math.max(8, window.innerHeight - hoehe - 8));
	return {
		left:
			breite === undefined ? r.left : Math.max(8, Math.min(r.left, window.innerWidth - breite - 8)),
		top: nachOben ? Math.max(8, r.top - hoehe - 4) : unten,
		breite: breite ?? r.width
	};
}
