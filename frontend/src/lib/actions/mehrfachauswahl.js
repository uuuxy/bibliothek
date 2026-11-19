// Mehrfachauswahl wie am Desktop, M3, Desktop: „Select a range: Shift + click“ und „Select all:
// Control + A, or Cmd + A (Mac)“. Eine Liste mit Kästchen trägt use:mehrfachauswahl; ihr
// Kästchen oder Knopf für alle trägt data-auswahl-alle. Die Aktion klickt die Kästchen selbst,
// damit Bereich und Alles durch dieselben Handler gehen wie ein Klick von Hand und jede Liste
// ihren Zustand behält.

/** @type {Set<HTMLElement>} */
const listen = new Set();

/** Kästchen der Zeilen: jedes Kästchen der Liste außer dem für alle. @param {HTMLElement} liste */
function zeilenKaestchen(liste) {
	return /** @type {HTMLInputElement[]} */ ([
		...liste.querySelectorAll('input[type="checkbox"]:not([data-auswahl-alle])')
	]);
}

const TEXTFELD =
	/^(?:text|search|email|password|number|tel|url|date|time|datetime-local|month|week)$/;

/** Strg/Cmd+A markiert in einem Textfeld weiter den Text. @param {EventTarget | null} ziel */
function schreibtText(ziel) {
	if (!(ziel instanceof HTMLElement)) return false;
	if (ziel.isContentEditable || ziel instanceof HTMLTextAreaElement) return true;
	if (ziel instanceof HTMLSelectElement) return true;
	return ziel instanceof HTMLInputElement && TEXTFELD.test(ziel.type);
}

/** @param {HTMLElement} liste */
const sichtbar = (liste) => liste.isConnected && liste.getClientRects().length > 0;

/**
 * Die Liste, die Strg/Cmd+A meint: die mit dem Fokus, sonst die einzige sichtbare, bei offenem
 * Dialog nur in ihm. Bei zwei Kandidaten keine, dann bleibt es beim Browser.
 */
function gemeinteListe() {
	const offen = [...listen].filter(sichtbar);
	const mitFokus = offen.find((l) => l.contains(document.activeElement));
	if (mitFokus) return mitFokus;
	const offenerDialog = [...document.querySelectorAll('[aria-modal="true"]')].at(-1);
	const kandidaten = offenerDialog ? offen.filter((l) => offenerDialog.contains(l)) : offen;
	return kandidaten.length === 1 ? kandidaten[0] : null;
}

/** @param {HTMLElement} liste */
function alleWaehlen(liste) {
	const offen = zeilenKaestchen(liste).filter((k) => !k.disabled);
	if (offen.every((k) => k.checked)) return;
	const alle = /** @type {HTMLButtonElement | HTMLInputElement | null} */ (
		liste.querySelector('[data-auswahl-alle]')
	);
	if (alle && !alle.disabled) alle.click();
	else for (const k of offen) if (!k.checked) k.click();
}

/** @param {KeyboardEvent} e */
function beiTaste(e) {
	if (e.key.toLowerCase() !== 'a' || !(e.ctrlKey || e.metaKey) || e.altKey || e.shiftKey) return;
	if (schreibtText(e.target)) return;
	const liste = gemeinteListe();
	if (!liste) return;
	e.preventDefault();
	alleWaehlen(liste);
}

/** @param {HTMLElement} liste */
export function mehrfachauswahl(liste) {
	/** @type {HTMLInputElement | null} */
	let anker = null;
	let selbst = false;

	/** @param {EventTarget | null} ziel */
	const kaestchenVon = (ziel) => {
		const el = ziel instanceof Element ? ziel : null;
		const k = el?.closest('label')?.querySelector('input[type="checkbox"]') ?? el;
		return k instanceof HTMLInputElement && zeilenKaestchen(liste).includes(k) ? k : null;
	};

	// Umschalt beim Drücken zöge sonst eine Textmarkierung über die Zeilen.
	/** @param {MouseEvent} e */
	const beiDruck = (e) => {
		if (e.shiftKey && kaestchenVon(e.target)) e.preventDefault();
	};

	/** @param {HTMLInputElement} k */
	const klickeSelbst = (k) => {
		selbst = true;
		try {
			k.click();
		} finally {
			selbst = false;
		}
	};

	/** @param {MouseEvent} e */
	const beiKlick = (e) => {
		if (selbst) return;
		const k = kaestchenVon(e.target);
		if (!k) return;
		if (e.target !== k) {
			// Den Klick auf den Text der Zeile reicht der Browser selbst ans Kästchen weiter,
			// Firefox aber nicht mit Umschalt; dann übernimmt die Aktion ihn in jedem Browser.
			if (!e.shiftKey) return;
			e.preventDefault();
			klickeSelbst(k);
		}
		const reihe = zeilenKaestchen(liste);
		const von = anker ? reihe.indexOf(anker) : -1;
		if (e.shiftKey && von !== -1 && anker !== k) {
			const bis = reihe.indexOf(k);
			for (const z of reihe.slice(Math.min(von, bis), Math.max(von, bis) + 1))
				if (z !== k && !z.disabled && z.checked !== k.checked) klickeSelbst(z);
		}
		anker = k;
	};

	liste.addEventListener('mousedown', beiDruck);
	liste.addEventListener('click', beiKlick);
	if (listen.size === 0) document.addEventListener('keydown', beiTaste);
	listen.add(liste);

	return {
		destroy() {
			liste.removeEventListener('mousedown', beiDruck);
			liste.removeEventListener('click', beiKlick);
			listen.delete(liste);
			if (listen.size === 0) document.removeEventListener('keydown', beiTaste);
		}
	};
}
