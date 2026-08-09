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

export const toastStore = new (class {
	/** @type {{id: number, message: string, type: ToastTyp, aktion?: ToastAktion}[]} */
	toasts = $state([]);

	#counter = 0;
	/** @type {Map<number, ReturnType<typeof setTimeout>>} */
	#timers = new SvelteMap();

	/**
	 * @param {string} message
	 * @param {ToastTyp} [type='info']
	 * @param {ToastAktion} [aktion] Folgehandlung; verlängert die Standzeit auf 10 s,
	 *   weil eine Meldung, die man BEDIENEN soll, nicht nach fünf Sekunden wegfliegen darf.
	 */
	addToast(message, type = 'info', aktion = undefined) {
		const id = this.#counter++;
		this.toasts.push({ id, message, type, aktion });
		this.#timers.set(
			id,
			setTimeout(() => this.removeToast(id), aktion ? 10000 : 5000)
		);
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
