// utils/kamerafehler.js
// Was die Seite sagt, wenn der Browser die Kamera nicht hergibt.

/**
 * Der Satz zu einem Fehler von getUserMedia. Der Browser nennt den Grund am Namen der Ausnahme
 * (NotAllowedError, NotFoundError, …), sein Text dazu ist englisch und je Browser anders:
 * „Permission denied" in Chromium, „The request is not allowed by the user agent …" in WebKit.
 *
 * @param {unknown} e
 * @returns {string}
 */
export function kamerafehlertext(e) {
	const name = /** @type {{ name?: unknown }} */ (e)?.name;
	switch (name) {
		case 'NotAllowedError':
		case 'SecurityError':
			return 'Der Browser hat den Zugriff auf die Kamera verweigert. Freigeben lässt er sich in den Website-Einstellungen des Browsers.';
		case 'NotFoundError':
		case 'OverconstrainedError':
			return 'Der Browser findet keine Kamera. Bitte eine anschließen und erneut versuchen.';
		case 'NotReadableError':
			return 'Die Kamera lässt sich nicht starten, vermutlich nutzt sie ein anderes Programm. Bitte dieses schließen und erneut versuchen.';
		case 'NotSupportedError':
			return 'Dieser Browser gibt hier keine Kamera frei.';
		default:
			return 'Die Kamera lässt sich nicht starten.';
	}
}
