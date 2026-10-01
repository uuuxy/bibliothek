/**
 * `use:ruhtWennTraege={{ anhalten, fortsetzen }}` — ein Bauteil mit laufender Kamera ruht,
 * solange es in einem trägen Bereich der Seite steht (inert).
 *
 * Anlass: Hinter dem Sperrbildschirm bleibt die Anwendung stehen und wird träge geschaltet
 * (Anwendungsrahmen.svelte). Eine Kamera darin liefe weiter: Ein vorgehaltener Barcode würde
 * gelesen, und ihr Licht brennt neben einem gesperrten Bildschirm. Das Bauteil richtet sich
 * nach dem Bereich, in dem es steht, und muss die Sperre dafür nicht kennen.
 *
 * @param {HTMLElement} node
 * @param {{ anhalten: () => void, fortsetzen: () => void }} ruf
 */
export function ruhtWennTraege(node, ruf) {
	let traege = !!node.closest('[inert]');
	const beobachter = new MutationObserver(() => {
		const jetzt = !!node.closest('[inert]');
		if (jetzt === traege) return;
		traege = jetzt;
		if (jetzt) ruf.anhalten();
		else ruf.fortsetzen();
	});
	// Träge wird ein Vorfahre, nicht das Bauteil selbst — deshalb das ganze Dokument, aber nur
	// dieses eine Attribut.
	beobachter.observe(document.documentElement, {
		attributes: true,
		attributeFilter: ['inert'],
		subtree: true
	});
	return {
		/** @param {{ anhalten: () => void, fortsetzen: () => void }} neu */
		update(neu) {
			ruf = neu;
		},
		destroy() {
			beobachter.disconnect();
		}
	};
}
