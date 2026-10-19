/** Abstand der Meldungen zum unteren Fensterrand in px, wie bei der Leiste der Leserdatei. */
export const RAND = 24;
/** Abstand über einer Leiste in px, wie der Abstand der Auswahlleiste zum Rand. */
export const ABSTAND = 16;

/** @typedef {{ top: number, bottom: number, right: number, width: number, height: number }} Kasten */

/**
 * Wo die Meldungen stehen: unten und mittig über der Fläche rechts der Navigation. Material 3,
 * Snackbar, „Placement": „Snackbars should be placed at the bottom of a UI, in front of the main
 * content. In some cases, snackbars can be nudged upwards to avoid overlapping with other UI
 * elements near the bottom"; „Avoid placing snackbars in front of navigation components".
 *
 * Eine Leiste rückt die Meldungen nur nach oben, wenn sie dort liegt, wo sie stünden: Eine
 * Auswahlleiste unter einer kurzen Liste steht weiter oben im Fluss und stört nicht.
 *
 * @param {{ fensterHoehe: number, hoehe: number, navigation: Kasten[], leisten: Kasten[] }} lage
 *   hoehe: die der Meldungen; die Kästen wie aus getBoundingClientRect
 * @returns {{ links: number, unten: number }} Abstand zum linken und zum unteren Fensterrand
 */
export function meldungsPlatz({ fensterHoehe, hoehe, navigation, leisten }) {
	const links = Math.max(0, ...navigation.filter(sichtbar).map((k) => k.right));
	let unten = RAND;
	// Von der untersten Leiste aufwärts: Wer über eine Leiste rückt, kann in die nächste geraten.
	for (const k of leisten.filter(sichtbar).sort((a, b) => b.bottom - a.bottom)) {
		const unterkante = fensterHoehe - unten;
		if (k.top < unterkante && k.bottom > unterkante - hoehe) {
			unten = fensterHoehe - k.top + ABSTAND;
		}
	}
	return { links: Math.round(links), unten: Math.round(unten) };
}

/** Ein ausgeblendetes Element misst 0 × 0. @param {Kasten} k */
const sichtbar = (k) => k.width > 0 && k.height > 0;
