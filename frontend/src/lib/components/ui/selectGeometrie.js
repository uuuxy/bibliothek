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
	const r = ausloeser.getBoundingClientRect();
	// Nach unten, wenn Platz ist; sonst nach oben. Sonst hinge die Liste bei
	// Feldern am unteren Bildrand unerreichbar außerhalb des Fensters.
	const untenPlatz = window.innerHeight - r.bottom;
	const hoehe = Math.min(LISTE_MAX, anzahl * ZEILE + 16);
	const nachOben = untenPlatz < hoehe && r.top > untenPlatz;
	return {
		left: r.left,
		top: nachOben ? Math.max(8, r.top - hoehe - 4) : r.bottom + 4,
		breite: r.width
	};
}
