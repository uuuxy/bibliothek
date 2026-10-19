import { snackbars } from '../stores/snackbars.svelte.js';

/**
 * `use:meldungenWeichenAus` — die Meldungen der Anwendung meiden dieses Element. Eine Leiste am
 * unteren Rand (`'unten'`) hebt sie an, die Navigation (`'links'`) rückt ihre Mitte nach rechts.
 * Wie `setAnchorView` der Snackbar bei Android; Material 3, Snackbar: „snackbars can be nudged
 * upwards to avoid overlapping with other UI elements near the bottom".
 *
 * @param {HTMLElement} node
 * @param {'unten' | 'links'} [seite='unten']
 */
export function meldungenWeichenAus(node, seite = 'unten') {
	snackbars.ausweichen.set(node, seite);
	return {
		/** @param {'unten' | 'links'} [neu='unten'] */
		update(neu = 'unten') {
			snackbars.ausweichen.set(node, neu);
		},
		destroy() {
			snackbars.ausweichen.delete(node);
		}
	};
}
