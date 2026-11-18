// Die Welle beim Drücken, nach Googles md-ripple (material-web). M3, States: „A ripple overlay
// signifies a pressed state." Ein Kreis wächst vom Druckpunkt in 450 ms über die ganze Fläche
// (easing standard), blendet in 105 ms ein und nach dem Loslassen, frühestens 225 ms nach dem
// Druck, in 375 ms aus. Ein Klick über die Tastatur lässt ihn aus der Mitte wachsen, ein Finger
// erst nach 150 ms, damit Wischen keine Welle auslöst. Die Welle gilt für dieselben Elemente wie
// der State-Layer in styles/komponenten.css; ihre Farbe ist die Farbe des Inhalts.

const ZIEL = 'button:not(.no-state-layer), .m3-state';
const WACHSEN_MS = 450;
const MINDESTENS_MS = 225;
const EINBLENDEN_MS = 105;
const AUSBLENDEN_MS = 375;
const FINGER_MS = 150;
const START_ANTEIL = 0.2;
const RAND = 10;
const WEICHER_RAND_MIN = 75;
const WEICHER_RAND_ANTEIL = 0.35;
const DECKKRAFT = 0.1;
const KURVE = 'cubic-bezier(0.2, 0, 0, 1)';

/** @param {EventTarget | null} ziel */
function wirt(ziel) {
	const el = ziel instanceof Element ? ziel.closest(ZIEL) : null;
	if (!el || el.matches(':disabled, [aria-disabled="true"]')) return null;
	// Ein leeres Element nimmt keine Kinder auf.
	return /^(INPUT|SELECT|TEXTAREA|IMG)$/.test(el.tagName) ? null : el;
}

/**
 * Lässt eine Welle in `el` wachsen und gibt die Funktion zurück, die sie ausblendet.
 * @param {Element} el @param {{ x: number, y: number }} [punkt] Druckpunkt im Fenster
 */
function starteWelle(el, punkt) {
	const r = el.getBoundingClientRect();
	const kreis = document.createElement('span');
	if (!r.width || !r.height || typeof kreis.animate !== 'function') return () => {};
	const huelle = document.createElement('span');
	huelle.className = 'm3-welle';
	huelle.setAttribute('aria-hidden', 'true');
	huelle.append(kreis);
	el.append(huelle);

	const groesste = Math.max(r.width, r.height);
	const start = Math.max(1, Math.floor(groesste * START_ANTEIL));
	const weich = Math.max(WEICHER_RAND_ANTEIL * groesste, WEICHER_RAND_MIN);
	const skala = (Math.hypot(r.width, r.height) + RAND + weich) / start;
	const mitte = { x: (r.width - start) / 2, y: (r.height - start) / 2 };
	const von = punkt ? { x: punkt.x - r.left - start / 2, y: punkt.y - r.top - start / 2 } : mitte;
	kreis.style.width = kreis.style.height = `${start}px`;
	const wachsen = kreis.animate(
		[
			{ transform: `translate(${von.x}px, ${von.y}px) scale(1)` },
			{ transform: `translate(${mitte.x}px, ${mitte.y}px) scale(${skala})` }
		],
		{ duration: WACHSEN_MS, easing: KURVE, fill: 'forwards' }
	);
	kreis.animate([{ opacity: 0 }, { opacity: DECKKRAFT }], {
		duration: EINBLENDEN_MS,
		easing: 'linear',
		fill: 'forwards'
	});

	let beendet = false;
	return () => {
		if (beendet) return;
		beendet = true;
		const gelaufen = Number(wachsen.currentTime) || 0;
		setTimeout(
			() => {
				const aus = kreis.animate([{ opacity: DECKKRAFT }, { opacity: 0 }], {
					duration: AUSBLENDEN_MS,
					easing: 'linear',
					fill: 'forwards'
				});
				aus.onfinish = () => huelle.remove();
			},
			Math.max(0, MINDESTENS_MS - gelaufen)
		);
	};
}

/**
 * Hängt die Welle an alle Bedienelemente des Dokuments; gibt die Funktion zum Abmelden zurück.
 * @param {Document} doc
 */
export function richteWelleEin(doc) {
	/** @type {Element | null} */
	let gedrueckt = null;
	/** @type {(() => void) | null} */
	let beende = null;
	/** @type {{ el: Element, punkt: { x: number, y: number }, uhr: ReturnType<typeof setTimeout> } | null} */
	let finger = null;
	/** Bricht ab; ein Finger, der vor Ablauf der Wartezeit loslässt, bekommt die Welle kurz. */
	const ende = (/** @type {boolean} */ mitWelle) => {
		if (finger) {
			clearTimeout(finger.uhr);
			if (mitWelle) starteWelle(finger.el, finger.punkt)();
			finger = null;
		}
		beende?.();
		beende = null;
		gedrueckt = null;
	};
	/** @param {PointerEvent} e */
	const druck = (e) => {
		if (!e.isPrimary || (e.pointerType !== 'touch' && e.button !== 0)) return;
		const el = wirt(e.target);
		if (!el) return;
		ende(false);
		gedrueckt = el;
		const punkt = { x: e.clientX, y: e.clientY };
		if (e.pointerType !== 'touch') beende = starteWelle(el, punkt);
		else
			finger = {
				el,
				punkt,
				uhr: setTimeout(() => {
					finger = null;
					beende = starteWelle(el, punkt);
				}, FINGER_MS)
			};
	};
	/** @param {MouseEvent} e Ein Klick ohne Zeiger kommt von der Tastatur. */
	const klick = (e) => {
		if (e.detail !== 0) return;
		const el = wirt(e.target);
		if (el) starteWelle(el)();
	};
	const los = () => ende(true);
	const abbruch = () => ende(false);
	/** @param {PointerEvent} e Die Maus verlässt das gedrückte Element. */
	const raus = (e) => {
		if (e.target === gedrueckt && e.pointerType !== 'touch') ende(false);
	};
	/** @type {[string, (e: any) => void][]} */
	const hoerer = [
		['pointerdown', druck],
		['click', klick],
		['pointerup', los],
		['pointercancel', abbruch],
		['contextmenu', abbruch],
		['dragstart', abbruch],
		['pointerleave', raus]
	];
	for (const [art, f] of hoerer) doc.addEventListener(art, f, true);
	return () => {
		for (const [art, f] of hoerer) doc.removeEventListener(art, f, true);
	};
}
