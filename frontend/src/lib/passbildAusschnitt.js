/** Form des gespeicherten Passbilds, Breite zu Höhe. */
export const PASSBILD_FORM = 3 / 4;

/**
 * Der größte mittige Ausschnitt eines Kamerabilds in der Form des Passbilds, in Bildpunkten.
 * Der Sucher zeichnet seinen hellen Bereich nach derselben Regel: Was dort hell ist, wird
 * gespeichert.
 *
 * @param {number} breite Breite des Kamerabilds
 * @param {number} hoehe Höhe des Kamerabilds
 * @returns {{ x: number, y: number, breite: number, hoehe: number }}
 */
export function passbildAusschnitt(breite, hoehe) {
	const hochkant = breite / hoehe < PASSBILD_FORM;
	const b = hochkant ? breite : Math.round(hoehe * PASSBILD_FORM);
	const h = hochkant ? Math.round(breite / PASSBILD_FORM) : hoehe;
	return { x: Math.round((breite - b) / 2), y: Math.round((hoehe - h) / 2), breite: b, hoehe: h };
}
