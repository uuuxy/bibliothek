import { describe, it, expect, beforeEach } from 'vitest';
import { berechneBox, LISTE_MAX } from './selectGeometrie.js';

/** Ein Feld an dieser Stelle, so wie getBoundingClientRect es liefert. @param {number} top */
const feld = (top) =>
	/** @type {HTMLElement} */ (
		/** @type {unknown} */ ({
			getBoundingClientRect: () => ({ left: 100, top, bottom: top + 36, width: 240 })
		})
	);

describe('berechneBox', () => {
	beforeEach(() => {
		Object.defineProperty(window, 'innerHeight', { value: 600, configurable: true });
	});

	it('öffnet unter dem Feld, wenn die Liste dort Platz hat', () => {
		expect(berechneBox(feld(100), 3)).toEqual({ left: 100, top: 140, breite: 240 });
	});

	it('öffnet über dem Feld, wenn unten kein Platz ist und oben mehr', () => {
		// Drei Einträge sind 160 px hoch, unter dem Feld bleiben 64 px.
		expect(berechneBox(feld(500), 3).top).toBe(500 - 160 - 4);
	});

	it('rückt ins Fenster, wenn die Liste weder darunter noch darüber passt', () => {
		// Zehn Einträge: 320 px. Unter dem Feld 314 px, darüber 250 px. Nach unten geöffnet lag
		// der letzte Eintrag 10 px unter dem Fensterrand.
		expect(berechneBox(feld(250), 10).top).toBe(600 - LISTE_MAX - 8);
	});
});
