/** @type {HTMLElement[]} Offene Fokusfallen in Einhängereihenfolge — zuletzt = zuoberst. */
const stapel = [];

const FOKUSSIERBAR =
	'a[href], button:not([disabled]), input:not([disabled]):not([type="hidden"]), ' +
	'select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"]), ' +
	'[contenteditable="true"]';

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
 * `use:fokusFalle` — ein Dialog nimmt den Fokus, hält ihn und gibt ihn zurück.
 *
 * Anlass (Prüfung 08.09.2026, gemessen im Browser): EIN Tab verließ jeden Dialog. Der
 * Fokus blieb auf dem Knopf, der den Dialog geöffnet hatte, und wanderte von dort
 * durch die Seite dahinter — für Tastatur- und Screenreader-Nutzung war der Dialog
 * damit nur ein Bild. 22 von 24 Dialogen hängen an Modal.svelte; die Aktion steht dort
 * EINMAL und gilt für alle (WCAG 2.4.3 Fokus-Reihenfolge, 2.1.2 keine Tastaturfalle
 * — die hier ist die erlaubte: Escape führt heraus, siehe escapeSchliesst).
 *
 * Drei Zusicherungen:
 *   1. Beim Öffnen wandert der Fokus auf das erste fokussierbare Element im Dialog —
 *      es sei denn, etwas darin hat ihn schon (ein Feld mit `autofocus`).
 *   2. Tab und Shift+Tab kreisen innerhalb des Dialogs.
 *   3. Beim Schließen kehrt der Fokus zum Auslöser zurück — aber nur, wenn ihn
 *      inzwischen niemand anders genommen hat (nach „Schüler anlegen" springt die
 *      Anwendung ins Profil; dem Auslöser den Fokus aufzuzwingen wäre falsch).
 *
 * NUR DIE OBERSTE Falle reagiert (Stapel wie bei escapeSchliesst): Öffnet ein Dialog
 * eine Rückfrage (BestaetigungsDialog liegt im DOM AUSSERHALB des fragenden Dialogs),
 * gehört der Fokus der Rückfrage; die äußere Falle hält still, bis die innere wieder
 * weg ist, und bekommt den Fokus dann von deren Rückgabe.
 *
 * @param {HTMLElement} node das Element mit role="dialog"
 */
export function fokusFalle(node) {
	const ausloeser = document.activeElement;
	stapel.push(node);

	queueMicrotask(() => {
		if (stapel[stapel.length - 1] !== node || node.contains(document.activeElement)) return;
		(kandidaten(node)[0] ?? node).focus({ preventScroll: true });
	});

	/** @param {KeyboardEvent} e */
	function beiTaste(e) {
		if (e.key !== 'Tab' || stapel[stapel.length - 1] !== node) return;
		const liste = kandidaten(node);
		if (liste.length === 0) {
			e.preventDefault();
			node.focus();
			return;
		}
		const erstes = liste[0];
		const letztes = liste[liste.length - 1];
		const aktiv = document.activeElement;
		const drinnen = aktiv instanceof HTMLElement && node.contains(aktiv) && aktiv !== node;
		if (e.shiftKey && (!drinnen || aktiv === erstes)) {
			e.preventDefault();
			letztes.focus();
		} else if (!e.shiftKey && (!drinnen || aktiv === letztes)) {
			e.preventDefault();
			erstes.focus();
		}
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
