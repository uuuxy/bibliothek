import { SvelteMap } from 'svelte/reactivity';

/**
 * 'warning' kam dazu, als das eigene Toast-Markup der Omnibox hier aufging: Der
 * Kiosk unterscheidet zwischen "hat geklappt", "hat geklappt, aber anders als
 * erwartet" (Fremdrückgabe, Offline-Puffer) und "Fehler". Ohne die dritte Stufe
 * wäre die mittlere still zum Fehler geworden.
 * @typedef {'success' | 'error' | 'warning' | 'info'} ToastTyp
 */

/**
 * Eine Folgehandlung im Toast. M3 nennt das den Action-Slot der Snackbar und erlaubt
 * GENAU EINE — sie ist der Anschluss an das, was gerade passiert ist ("eingebucht →
 * Etiketten drucken"), nicht ein zweites Menü.
 * @typedef {{ label: string, onClick: () => void }} ToastAktion
 */

export const snackbars = new (class {
	/**
	 * Höchstens eine Meldung (Material 3, Snackbar: „Only one snackbar may be displayed at a
	 * time"); siehe addToast.
	 * @type {{id: number, message: string, type: ToastTyp, aktion?: ToastAktion}[]}
	 */
	toasts = $state([]);

	/**
	 * Was die Meldungen nicht verdecken dürfen: Eine Leiste am unteren Rand hebt sie an, die
	 * Navigation links rückt ihre Mitte nach rechts (SnackbarAnzeige, meldungsPlatz.js).
	 * Eingetragen wird nur über `use:meldungenWeichenAus`.
	 * @type {Map<HTMLElement, 'unten' | 'links'>}
	 */
	ausweichen = new SvelteMap();

	#counter = 0;
	/** @type {Map<number, ReturnType<typeof setTimeout>>} */
	#timers = new SvelteMap();

	/**
	 * Die neue Meldung ersetzt die stehende, auch eine mit Folgehandlung. So macht es Googles
	 * Snackbar für Android (SnackbarManager.show, DISMISS_EVENT_CONSECUTIVE). In einer
	 * Warteschlange erschiene an der Theke die Meldung zum sechsten Buch erst 25 s nach dem Scan.
	 * @param {string} message
	 * @param {ToastTyp} [type='info']
	 * @param {ToastAktion} [aktion] Folgehandlung; verlängert die Standzeit auf 10 s,
	 *   weil eine Meldung, die man BEDIENEN soll, nicht nach fünf Sekunden wegfliegen darf.
	 */
	addToast(message, type = 'info', aktion = undefined) {
		for (const timer of this.#timers.values()) clearTimeout(timer);
		this.#timers.clear();
		const id = this.#counter++;
		this.toasts = [{ id, message, type, aktion }];
		this.#starte(id);
	}

	/** @param {number} id */
	#starte(id) {
		const meldung = this.toasts.find((t) => t.id === id);
		if (!meldung) return;
		this.#timers.set(
			id,
			setTimeout(() => this.removeToast(id), meldung.aktion ? 10000 : 5000)
		);
	}

	/**
	 * Die Standzeit hält an, solange Maus oder Tastaturfokus auf dem Toast liegen
	 * (WCAG 2.2.1: eine Zeitgrenze muss sich verlängern lassen). Wer noch liest oder
	 * die Folgehandlung anpeilt, verliert die Meldung nicht unter der Hand.
	 * @param {number} id
	 */
	pausieren(id) {
		const timer = this.#timers.get(id);
		if (timer === undefined) return;
		clearTimeout(timer);
		this.#timers.delete(id);
	}

	/** Nach dem Verlassen läuft die volle Standzeit noch einmal. @param {number} id */
	fortsetzen(id) {
		if (!this.#timers.has(id)) this.#starte(id);
	}

	/**
	 * @param {number} id
	 */
	removeToast(id) {
		const timer = this.#timers.get(id);
		if (timer !== undefined) {
			clearTimeout(timer);
			this.#timers.delete(id);
		}
		this.toasts = this.toasts.filter((t) => t.id !== id);
	}
})();
