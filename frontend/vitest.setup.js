// Was jsdom fehlt, aber jeder Browser hat — hier einmal nachgereicht.
//
// jsdom kennt kein `scrollIntoView`. Eine Komponente, die den aktiven Eintrag sichtbar
// hält (SelectListe, tastaturNavigation), wirft damit in einem `$effect` — und zwar an der
// Zusammenfassung vorbei: Der Lauf druckt „… passed" und beendet sich trotzdem mit 1.
//
// Die Umgebung ist die richtige Stelle, nicht jede Testdatei für sich: Das hielte nur,
// solange jeder neue Test davon weiß. Wer eine Komponente rendert, soll nicht wissen
// müssen, wo jsdom Lücken hat.
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
		observe = vi.fn();
		unobserve = vi.fn();
		disconnect = vi.fn();
	};
}
