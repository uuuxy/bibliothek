import { snackbars } from '../stores/snackbars.svelte.js';

/**
 * Schreibt in die Zwischenablage und sagt, ob es gelungen ist. Ohne sichere Verbindung gibt es sie
 * nicht, und der Browser kann sie verweigern: Dann steht die Meldung da, und der Aufrufer behauptet
 * kein „Kopiert".
 * @param {string} text
 * @param {string} [hinweis] was statt dessen zu tun ist
 * @returns {Promise<boolean>}
 */
export async function kopiere(text, hinweis = '') {
	try {
		await navigator.clipboard.writeText(text);
		return true;
	} catch {
		snackbars.addToast(
			hinweis ? `Kopieren nicht möglich. ${hinweis}` : 'Kopieren nicht möglich.',
			'error'
		);
		return false;
	}
}
