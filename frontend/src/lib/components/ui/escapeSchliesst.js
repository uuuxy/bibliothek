/** @type {{ ruf: (() => void) | undefined }[]} Offene Dialoge in Einhängereihenfolge. */
const stapel = [];

/**
 * Ist gerade ein Overlay offen, dem Escape gehört? Der Router fragt das, bevor er Escape
 * als „zurück an die Theke" deutet (escapeRegel.js).
 *
 * Der Router hört auf window und ist als Erster eingehängt; ein Dialog hängt seinen
 * Escape-Lauscher später ein und kommt deshalb als Zweiter dran, sein preventDefault
 * erreicht den Router nicht mehr. Steht der Fokus nicht in einem Eingabefeld (Menüeintrag,
 * Knopf), spränge das Programm sonst mit demselben Tastendruck an die Theke und schlösse
 * den Dialog.
 */
export function escapeIstBelegt() {
	return stapel.length > 0;
}

/**
 * `use:escapeSchliesst={schliessen}` — Escape schließt diesen Dialog. An der Theke wird
 * schnell gearbeitet; ein Dialog, der nur mit der Maus zugeht, hält den Betrieb auf, und
 * mit der Tastatur allein wäre er gar nicht zu schließen (der Klick auf den Hintergrund ist
 * kein Tastaturweg).
 *
 * Eine Aktion und kein weiteres Bauteil: Dialoge mit eigenem Aufbau (dunkles Kamerafeld,
 * Alarmfläche ohne Kopfzeile, Formular mit eigener Fußzeile) bekommen dasselbe Verhalten,
 * ohne ihre Gestalt zu ändern, und es steht an einer Stelle. `Dialog.svelte` benutzt
 * dieselbe Aktion.
 *
 * Nur der oberste Dialog reagiert: Sonst schlösse ein Escape über einer Rückfrage („Gebühr
 * wirklich stornieren?") auch den Dialog darunter; der Anwender sieht einen Tastendruck,
 * das Programm täte zwei Dinge. Der Stapel führt Buch in
 * Einhängereihenfolge; zuletzt eingehängt heißt zuoberst.
 *
 * @param {HTMLElement} node
 * @param {(() => void) | undefined} schliessen
 */
export function escapeSchliesst(node, schliessen) {
	/** @type {{ ruf: (() => void) | undefined }} */
	const eintrag = { ruf: schliessen };
	stapel.push(eintrag);

	/** @param {KeyboardEvent} e */
	function beiTaste(e) {
		if (e.key !== 'Escape') return;
		// Hat ein Bauteil innerhalb des Dialogs die Taste schon verarbeitet, ist sie
		// verbraucht: Ein offenes Auswahlfeld (ui/Select) schließt mit Escape seine Liste und
		// ruft preventDefault, und dieser Lauscher am window käme danach trotzdem dran. Ein
		// Tastendruck schlösse die Liste und den ganzen Dialog, und die halb ausgefüllte
		// Eingabe wäre weg. Dieselbe Regel, mit der escapeIstBelegt() den Router vom Dialog
		// fernhält, gilt damit auch nach innen.
		if (e.defaultPrevented) return;
		if (stapel.at(-1) !== eintrag) return;
		e.preventDefault();
		e.stopPropagation();
		eintrag.ruf?.();
	}
	window.addEventListener('keydown', beiTaste);

	return {
		/** @param {(() => void) | undefined} neu */
		update(neu) {
			eintrag.ruf = neu;
		},
		destroy() {
			window.removeEventListener('keydown', beiTaste);
			const i = stapel.indexOf(eintrag);
			if (i >= 0) stapel.splice(i, 1);
		}
	};
}
