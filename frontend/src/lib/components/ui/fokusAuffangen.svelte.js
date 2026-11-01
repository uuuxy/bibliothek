import { untrack } from 'svelte';

/**
 * Bringen neue Daten das Element zum Verschwinden, auf dem der Fokus steht, fiele er auf den Anfang
 * der Seite, und die nächste Tab-Taste begönne vor der Navigation. Er rückt dann auf das Ziel, ohne
 * eines auf den Anfang des Inhalts (Hauptbereich.svelte, wie der Skip-Link). Aufruf beim Erzeugen des
 * Bauteils.
 * @param {() => unknown} daten was das Element verschwinden lässt
 * @param {() => HTMLElement | null | undefined} wurzel worin der Fokus stehen muss
 * @param {() => HTMLElement | null | undefined} [ziel]
 */
export function fokusAuffangen(daten, wurzel, ziel) {
	let standDrin = false;
	$effect.pre(() => {
		daten();
		standDrin = untrack(wurzel)?.contains(document.activeElement) ?? false;
	});
	$effect(() => {
		daten();
		if (standDrin && !untrack(wurzel)?.contains(document.activeElement)) {
			(ziel ? untrack(ziel) : document.getElementById('hauptinhalt'))?.focus();
		}
		standDrin = false;
	});
}
