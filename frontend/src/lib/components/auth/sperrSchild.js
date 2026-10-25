// Hinter dem Sperrbildschirm bleibt die Anwendung stehen, damit Ungespeichertes die Sperre
// überlebt (App.svelte). Sie ist ausgeblendet und träge (hidden, inert); ihre Bauteile hören
// aber auch auf Tasten und Zeiger der ganzen Seite. Escape am Sperrbildschirm führte den
// Router an die Theke und schlösse die offene Maske, ein Klick das offene Menü. Der Schild
// hält diese Ereignisse von der Anwendung fern, solange gesperrt ist.

/**
 * Bedienung, auf die Bauteile an window oder document hören. Welche Ereignisse dort ohne
 * Schild bleiben dürfen, begründet frontend-hygiene-seitenweite-zuhoerer.test.js.
 */
export const ABGESCHIRMT = [
	'keydown',
	'keyup',
	'keypress',
	'pointerdown',
	'pointerup',
	'pointermove',
	'pointercancel',
	'mousedown',
	'mouseup',
	'click',
	'dblclick',
	'auxclick',
	'contextmenu',
	'wheel',
	'touchstart',
	'touchmove',
	'touchend',
	'paste',
	'cut',
	'drop'
];

const SPERRBILDSCHIRM = '[data-sperrbildschirm]';

/**
 * Stellt den Schild auf — einmal beim Laden und vor den Bauteilen: Zuhörer am selben Ziel
 * laufen in der Reihenfolge ihrer Anmeldung, und der Schild muss der erste sein.
 * @param {() => boolean} istGesperrt
 * @returns {() => void} baut ihn wieder ab
 */
export function stelleSperrSchildAuf(istGesperrt) {
	/** @param {Event} e */
	function ausserhalb(e) {
		if (!istGesperrt()) return;
		const ziel = e.target;
		if (ziel instanceof Element && ziel.closest(SPERRBILDSCHIRM)) return;
		// Nicht am Sperrbildschirm: Das Ereignis erreicht niemanden. Was der Browser selbst
		// damit tut (Tab, Neuladen), bleibt.
		e.stopImmediatePropagation();
	}
	/** @param {Event} e */
	function nachDemSperrbildschirm(e) {
		// Am Sperrbildschirm: Seine eigenen Zuhörer sind durch. Weiter oben, an document und
		// window, hört nur noch die Anwendung dahinter.
		if (istGesperrt()) e.stopPropagation();
	}
	const wurzel = document.documentElement;
	for (const name of ABGESCHIRMT) {
		window.addEventListener(name, ausserhalb, { capture: true, passive: true });
		wurzel.addEventListener(name, nachDemSperrbildschirm, { passive: true });
	}
	return () => {
		for (const name of ABGESCHIRMT) {
			window.removeEventListener(name, ausserhalb, { capture: true });
			wurzel.removeEventListener(name, nachDemSperrbildschirm);
		}
	};
}
