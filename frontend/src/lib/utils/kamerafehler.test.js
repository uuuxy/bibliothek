import { describe, it, expect } from 'vitest';
import { kamerafehlertext } from './kamerafehler.js';

describe('kamerafehlertext', () => {
	// Die Wortlaute, mit denen die Browser die Kamera verweigern (gemessen mit Playwright).
	it.each([
		['Chromium', 'Permission denied'],
		[
			'WebKit',
			'The request is not allowed by the user agent or the platform in the current context, possibly because the user denied permission.'
		]
	])('nennt die Verweigerung in %s auf Deutsch', (_browser, text) => {
		const satz = kamerafehlertext(new DOMException(text, 'NotAllowedError'));
		expect(satz).toBe(
			'Der Browser hat den Zugriff auf die Kamera verweigert. Freigeben lässt er sich in den Website-Einstellungen des Browsers.'
		);
		expect(satz).not.toContain(text);
	});

	it('unterscheidet fehlende, belegte und nicht angebotene Kamera', () => {
		expect(kamerafehlertext(new DOMException('Requested device not found', 'NotFoundError'))).toBe(
			'Der Browser findet keine Kamera. Bitte eine anschließen und erneut versuchen.'
		);
		expect(
			kamerafehlertext(new DOMException('Could not start video source', 'NotReadableError'))
		).toBe(
			'Die Kamera lässt sich nicht starten, vermutlich nutzt sie ein anderes Programm. Bitte dieses schließen und erneut versuchen.'
		);
		expect(kamerafehlertext(new DOMException('Not supported', 'NotSupportedError'))).toBe(
			'Dieser Browser gibt hier keine Kamera frei.'
		);
	});

	it('sagt bei allem anderen, dass die Kamera nicht startet, ohne den Text des Browsers', () => {
		expect(kamerafehlertext(new DOMException('Starting videoinput failed', 'AbortError'))).toBe(
			'Die Kamera lässt sich nicht starten.'
		);
		expect(
			kamerafehlertext(
				new TypeError("Cannot read properties of undefined (reading 'getUserMedia')")
			)
		).toBe('Die Kamera lässt sich nicht starten.');
		expect(kamerafehlertext('kaputt')).toBe('Die Kamera lässt sich nicht starten.');
		expect(kamerafehlertext(null)).toBe('Die Kamera lässt sich nicht starten.');
	});
});
