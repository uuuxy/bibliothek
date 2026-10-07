// Was jsdom fehlt, aber jeder Browser hat — hier EINMAL nachgereicht.
//
// jsdom kennt kein `scrollIntoView`. Eine Komponente, die den aktiven Eintrag sichtbar
// hält (SelectListe, keyboardNav), wirft damit in einem `$effect` — und zwar an der
// Zusammenfassung vorbei: Der Lauf druckt „104 passed / 547 passed" und beendet sich
// trotzdem mit 1. Am 12.09.2026 stand `main` genau so rot, während die letzte Zeile
// grün aussah; gefunden wurde es erst, weil jemand auf den Exit-Code geschaut hat.
//
// Bis dahin stopfte jede Testdatei die Lücke selbst (`beforeAll` in
// LmfPlanReihenfolge.test.js). Das hält nur, solange jeder neue Test davon weiß — zwei
// Dateien desselben Tages wussten es nicht. Die Umgebung ist die richtige Stelle: Wer
// eine Komponente rendert, soll nicht wissen müssen, wo jsdom Lücken hat.
import { vi } from 'vitest';

if (!Element.prototype.scrollIntoView) {
	Element.prototype.scrollIntoView = vi.fn();
}

// jsdom kennt auch `animate` nicht: Jedes Bauteil mit `transition:` wirft sonst beim Rendern.
// Die Attrappe meldet die Animation gleich als beendet, der Übergang läuft damit durch.
if (!Element.prototype.animate) {
	/** @type {any} */ (Element.prototype).animate = () => {
		/** @type {{ onfinish: (() => void) | null, cancel: () => void, currentTime: number, playState: string }} */
		const animation = { onfinish: null, cancel: () => {}, currentTime: 0, playState: 'finished' };
		queueMicrotask(() => animation.onfinish?.());
		return animation;
	};
}

// jsdom kennt auch `matchMedia` nicht; die Seitenleiste fragt damit nach der Fensterbreite.
// Die Attrappe verneint jede Abfrage. Ein Test, der eine Breite braucht, setzt seine eigene.
if (!window.matchMedia) {
	window.matchMedia = (abfrage) =>
		/** @type {any} */ (Object.assign(new EventTarget(), { matches: false, media: abfrage }));
}

// jsdom kennt auch `ResizeObserver` nicht; actions/resthoehe.js beobachtet damit, ob sich über
// einem Element etwas verschiebt. Die Attrappe beobachtet nichts: jsdom rechnet kein Layout.
if (!globalThis.ResizeObserver) {
	globalThis.ResizeObserver = class {
		observe() {}
		unobserve() {}
		disconnect() {}
	};
}
