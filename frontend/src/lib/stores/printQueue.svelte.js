// stores/printQueue.svelte.js
// Warteschlange für den Etikettendruck: Bestellungen/Wareneingang legen
// Exemplare hierher, das DruckCenter (labels) konsumiert sie.
// Entkoppelt die Bestell-Domäne vom Inventur-Store.

/** @typedef {{ barcode_id: string, titel: string, autor?: string }} PrintCopy */

export const printQueue = $state({
	/** @type {PrintCopy[] | null} */
	copies: null
});

export function clearPrintQueue() {
	printQueue.copies = null;
}
