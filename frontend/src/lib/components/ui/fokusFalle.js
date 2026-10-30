/** @type {HTMLElement[]} Offene Fokusfallen in Einhängereihenfolge — zuletzt = zuoberst. */
const stapel = [];

const FOKUSSIERBAR =
	'a[href], button:not([disabled]), input:not([disabled]):not([type="hidden"]), ' +
	'select:not([disabled]), textarea:not([disabled]), summary, ' +
	'[tabindex]:not([tabindex="-1"]), [contenteditable="true"]';

/**
 * Sichtbare, fokussierbare Nachkommen in Dokumentreihenfolge. getClientRects statt
 * offsetParent: Letzteres ist bei `position: fixed` immer null, und genau so liegen
 * Dialoge.
 * @param {HTMLElement} node
 */
function kandidaten(node) {
	return /** @type {HTMLElement[]} */ ([...node.querySelectorAll(FOKUSSIERBAR)]).filter(
		(el) => el.getClientRects().length > 0
	);
}

/**
 * Die Halte der Tab-Taste: Eine Radiogruppe ist ein Halt, ihr gewählter Knopf; innerhalb der
 * Gruppe wandern die Pfeiltasten.
 * @param {HTMLElement} node
 */
function halteDerTaste(node) {
	const liste = kandidaten(node);
	const istRadio = (/** @type {HTMLElement} */ el) =>
		el instanceof HTMLInputElement && el.type === 'radio';
	return liste.filter(
		(el) =>
			!istRadio(el) ||
			/** @type {HTMLInputElement} */ (el).checked ||
			!liste.some(
				(a) =>
					istRadio(a) &&
					/** @type {HTMLInputElement} */ (a).checked &&
					/** @type {HTMLInputElement} */ (a).name === /** @type {HTMLInputElement} */ (el).name
			)
	);
}

/**
 * `use:fokusFalle` — ein Dialog nimmt den Fokus, hält ihn und gibt ihn zurück. Sonst
 * bliebe der Fokus auf dem Knopf, der den Dialog geöffnet hat, und ein Tab wanderte durch
 * die Seite dahinter; für Tastatur- und Screenreader-Nutzung wäre der Dialog nur ein Bild.
 * Die Aktion steht in Dialog.svelte und gilt für jeden Dialog daran (WCAG 2.4.3
 * Fokus-Reihenfolge, 2.1.2 keine Tastaturfalle; die hier ist die erlaubte: Escape führt
 * heraus, siehe escapeSchliesst).
 *
 * Drei Zusicherungen:
 *   1. Beim Öffnen wandert der Fokus auf das erste fokussierbare Element im Dialog —
 *      es sei denn, etwas darin hat ihn schon (ein Feld mit `autofocus`).
 *   2. Tab und Shift+Tab kreisen innerhalb des Dialogs. Die Falle geht selbst weiter, statt
 *      das dem Browser zu überlassen: Safari lässt beim Tab Knöpfe und Links aus. Der Fokus
 *      spränge vom letzten Feld am letzten Knopf vorbei aus dem Dialog.
 *   3. Beim Schließen kehrt der Fokus zum Auslöser zurück — aber nur, wenn ihn
 *      inzwischen niemand anders genommen hat (nach „Schüler anlegen" springt die
 *      Anwendung ins Profil; dem Auslöser den Fokus aufzuzwingen wäre falsch).
 *
 * Nur die oberste Falle reagiert (Stapel wie bei escapeSchliesst): Öffnet ein Dialog eine
 * Rückfrage (BestaetigungsDialog liegt im DOM außerhalb des fragenden Dialogs),
 * gehört der Fokus der Rückfrage; die äußere Falle hält still, bis die innere wieder
 * weg ist, und bekommt den Fokus dann von deren Rückgabe.
 *
 * @param {HTMLElement} node das Element mit role="dialog"
 */
export function fokusFalle(node) {
	const ausloeser = document.activeElement;
	stapel.push(node);

	queueMicrotask(() => {
		if (stapel.at(-1) !== node || node.contains(document.activeElement)) return;
		(kandidaten(node)[0] ?? node).focus({ preventScroll: true });
	});

	/** @param {KeyboardEvent} e */
	function beiTaste(e) {
		// Ein Bauteil im Dialog kann die Taste für sich verbraucht haben (Vorschlag übernehmen).
		if (e.key !== 'Tab' || e.defaultPrevented || stapel.at(-1) !== node) return;
		e.preventDefault();
		const liste = halteDerTaste(node);
		const aktiv = document.activeElement;
		const drinnen = aktiv instanceof HTMLElement && node.contains(aktiv) && aktiv !== node;
		const folgt = liste.filter(
			(el) => drinnen && aktiv.compareDocumentPosition(el) & Node.DOCUMENT_POSITION_FOLLOWING
		);
		const davor = liste.filter((el) => !folgt.includes(el) && el !== aktiv);
		const reihe = e.shiftKey ? [...davor.reverse(), ...folgt.reverse()] : [...folgt, ...davor];
		// Ein Element, das den Fokus nicht nimmt (verborgen, gesperrte Gruppe), hält die Runde nicht auf.
		for (const el of reihe) {
			el.focus();
			if (document.activeElement === el) return;
		}
		if (!drinnen) node.focus();
	}
	node.addEventListener('keydown', beiTaste);

	return {
		destroy() {
			node.removeEventListener('keydown', beiTaste);
			const i = stapel.lastIndexOf(node);
			if (i >= 0) stapel.splice(i, 1);
			const aktiv = document.activeElement;
			const verwaist = !aktiv || aktiv === document.body || node.contains(aktiv);
			if (verwaist && ausloeser instanceof HTMLElement && document.contains(ausloeser)) {
				ausloeser.focus({ preventScroll: true });
			}
		}
	};
}
