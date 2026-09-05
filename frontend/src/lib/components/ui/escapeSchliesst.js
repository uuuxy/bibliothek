/** @type {{ ruf: (() => void) | undefined }[]} Offene Dialoge in Einhängereihenfolge. */
const stapel = [];

/**
 * Ist gerade ein Overlay offen, dem Escape gehört? Der Router fragt das, BEVOR er
 * Escape als „zurück an die Theke" deutet (escapeRegel.js).
 *
 * Anlass (06.09.2026, am Überlaufmenü des LMF-Planers gesehen): Der Router hört auf
 * window und ist als Erster eingehängt; ein Dialog hängt seinen Escape-Lauscher später
 * ein und kommt deshalb ZWEITER dran. Sein preventDefault erreicht den Router nicht mehr.
 * Steht der Fokus nicht in einem Eingabefeld (Menüeintrag, Knopf), sprang das Programm
 * mit demselben Tastendruck an die Theke UND schloss den Dialog — seit dem 05.09. für
 * jedes Overlay mit dieser Aktion, auch Modal.svelte.
 */
export function escapeIstBelegt() {
	return stapel.length > 0;
}

/**
 * `use:escapeSchliesst={schliessen}` — Escape schließt diesen Dialog.
 *
 * Anlass (05.09.2026): KEINES der elf selbstgebauten Overlays reagierte auf Escape,
 * `Modal.svelte` als einziges schon. An der Theke wird schnell gearbeitet; ein Dialog,
 * der nur mit der Maus zugeht, hält den Betrieb auf — und mit der Tastatur allein war
 * er gar nicht zu schließen (der Klick auf den Hintergrund ist kein Tastaturweg).
 *
 * Warum eine Aktion und kein weiteres Bauteil: Die elf Dialoge unterscheiden sich in
 * ihrem Aufbau (dunkles Kamerafeld, Alarmfläche ohne Kopfzeile, Formular mit eigener
 * Fußzeile). Sie alle auf `Modal.svelte` zu ziehen, ist ein eigener Durchgang mit
 * sichtbaren Folgen. Das VERHALTEN lässt sich vorher zusammenführen, ohne die Gestalt
 * anzufassen — und es gibt danach genau eine Stelle, an der es steht. `Modal.svelte`
 * benutzt dieselbe Aktion.
 *
 * NUR DER OBERSTE Dialog reagiert. Ohne diese Regel schlösse ein Escape über einer
 * Rückfrage („Gebühr wirklich stornieren?") auch den Dialog darunter — der Anwender
 * sieht einen Tastendruck, das Programm täte zwei Dinge. Der Stapel führt Buch in
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
		if (stapel[stapel.length - 1] !== eintrag) return;
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
