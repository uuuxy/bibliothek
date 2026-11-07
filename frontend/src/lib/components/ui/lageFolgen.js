/** Eine schwebende Fläche (Menü, Vorschau daneben) folgt ihrem Auslöser.
 *
 *  Ereignisse für Bildlauf und Fenstergröße reichen dafür nicht: Rückt der Auslöser, weil
 *  daneben etwas einfährt und ein zentrierter Dialog dabei wächst, feuert keins, und die Fläche
 *  bliebe an der alten Stelle stehen, im Dialog „Inventur starten" über dem Knopf „Starten".
 *  Deshalb wird in jedem Bild verglichen und nur gemessen, wenn sich Auslöser oder Fenster
 *  geändert haben. */

/** @type {(f: () => void) => number} */
const naechstesBild = (f) =>
	typeof requestAnimationFrame === 'function'
		? requestAnimationFrame(f)
		: /** @type {number} */ (/** @type {unknown} */ (setTimeout(f, 16)));

/** @param {number} kennung */
const bildAbbrechen = (kennung) =>
	typeof cancelAnimationFrame === 'function'
		? cancelAnimationFrame(kennung)
		: clearTimeout(kennung);

/**
 * Misst sofort und danach in jedem Bild, in dem sich das Rechteck des Auslösers oder die Größe
 * des Fensters geändert hat.
 * @param {() => Element | null | undefined} ausloeser
 * @param {() => void} messen
 * @returns {() => void} das Aufräumen
 */
export function folgeLage(ausloeser, messen) {
	let zuletzt = '';
	let kennung = 0;
	const pruefe = () => {
		const el = ausloeser();
		if (el) {
			const r = el.getBoundingClientRect();
			const jetzt = `${r.left},${r.top},${r.width},${r.height},${window.innerWidth},${window.innerHeight}`;
			if (jetzt !== zuletzt) {
				zuletzt = jetzt;
				messen();
			}
		}
		kennung = naechstesBild(pruefe);
	};
	pruefe();
	return () => bildAbbrechen(kennung);
}
