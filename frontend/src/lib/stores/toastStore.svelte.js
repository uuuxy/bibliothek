import { SvelteMap } from 'svelte/reactivity';

/**
 * 'warning' kam dazu, als das eigene Toast-Markup der Omnibox hier aufging: Der
 * Kiosk unterscheidet zwischen "hat geklappt", "hat geklappt, aber anders als
 * erwartet" (Fremdrückgabe, Offline-Puffer) und "Fehler". Ohne die dritte Stufe
 * wäre die mittlere still zum Fehler geworden.
 * @typedef {'success' | 'error' | 'warning' | 'info'} ToastTyp
 */

export const toastStore = new (class {
	/** @type {{id: number, message: string, type: ToastTyp}[]} */
	toasts = $state([]);

	#counter = 0;
	/** @type {Map<number, ReturnType<typeof setTimeout>>} */
	#timers = new SvelteMap();

	/**
	 * @param {string} message
	 * @param {ToastTyp} [type='info']
	 */
	addToast(message, type = 'info') {
		const id = this.#counter++;
		this.toasts.push({ id, message, type });
		this.#timers.set(
			id,
			setTimeout(() => this.removeToast(id), 5000)
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
