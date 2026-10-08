/**
 * `use:resthoehe` — setzt am Element `--resthoehe`: die Höhe von seiner Oberkante bis zur
 * Unterkante des Bereichs, der die Seite scrollt. Mit `max-h-(--resthoehe)` und
 * `overflow-y-auto` scrollt das Element in sich und endet spätestens am Rand der Seite; die
 * Seite selbst läuft nicht über. Mit `h-(--resthoehe)` reicht es immer bis dorthin.
 *
 * Gemessen statt gerechnet: Ein `calc(100vh - fester Abzug)` stimmt nicht, weil die Oberkante
 * wandert — mit einem Hinweisband darüber, mit einer Kopfzeile, die umbricht, beim Scrollen.
 *
 * `haftet`: Das Element klebt beim Scrollen (sticky). Dann zählt seine Oberkante im Fenster,
 * es wächst, bis es klebt. Sonst zählt seine Lage im Bereich, gleich wie weit gescrollt ist:
 * Ein Element im Fluss, das beim Scrollen wüchse, verlängerte die Seite unter sich selbst.
 *
 * @param {HTMLElement} node
 * @param {{ haftet?: boolean, mindestens?: number }} [wahl]
 */
export function resthoehe(node, wahl = {}) {
	const bereich = scrollbereich(node);
	const messen = () => {
		const b = bereich?.getBoundingClientRect();
		const hoehe = resthoeheAus(
			{
				bereichUnten: b ? b.bottom : window.innerHeight,
				gescrollt: bereich ? bereich.scrollTop : window.scrollY,
				oben: node.getBoundingClientRect().top
			},
			wahl
		);
		node.style.setProperty('--resthoehe', `${hoehe}px`);
	};
	// Höchstens einmal je Bild: Die Beobachter melden auch die Größe, die diese Messung setzt.
	let geplant = 0;
	const planen = () => {
		if (!geplant)
			geplant = requestAnimationFrame(() => {
				geplant = 0;
				messen();
			});
	};
	messen();
	window.addEventListener('scroll', planen, true);
	window.addEventListener('resize', planen);
	// Die Oberkante wandert auch, wenn sich über dem Element etwas ändert: ein Band, das später
	// erscheint, eine Liste, die geladen ist. Das zeigt sich an der Höhe des Inhalts im Bereich.
	const beobachter = new ResizeObserver(planen);
	for (const el of [document.body, node, bereich, ...(bereich?.children ?? [])]) {
		if (el) beobachter.observe(el);
	}
	return {
		/** @param {{ haftet?: boolean, mindestens?: number }} [neu] */
		update(neu = {}) {
			wahl = neu;
			planen();
		},
		destroy() {
			window.removeEventListener('scroll', planen, true);
			window.removeEventListener('resize', planen);
			beobachter.disconnect();
			cancelAnimationFrame(geplant);
		}
	};
}

/**
 * Der nächste Vorfahre, der senkrecht scrollt.
 * @param {HTMLElement} node
 * @returns {HTMLElement | null}
 */
function scrollbereich(node) {
	for (let e = node.parentElement; e; e = e.parentElement) {
		if (/(auto|scroll)/.test(getComputedStyle(e).overflowY)) return e;
	}
	return null;
}

/**
 * Kleiner wird das Element nicht: Bleibt darunter weniger Platz, etwa in einem kurzen Fenster
 * mit Hinweisbändern über der Liste, behält es diese Höhe, und die Seite scrollt um den Rest.
 */
export const MINDESTHOEHE = 240;

/**
 * Die Rechnung ohne Dokument. Kanten in Fensterkoordinaten.
 * @param {{ bereichUnten: number, gescrollt: number, oben: number }} lage
 * @param {{ haftet?: boolean, mindestens?: number }} [wahl]
 * @returns {number}
 */
export function resthoeheAus(lage, { haftet = false, mindestens = MINDESTHOEHE } = {}) {
	const oben = haftet ? lage.oben : lage.oben + lage.gescrollt;
	return Math.max(mindestens, Math.floor(lage.bereichUnten - oben));
}
