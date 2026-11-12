import { describe, it, expect } from 'vitest';
import { berechneMenueBox, berechneVorschauBox } from './menueGeometrie.js';

const fenster = { breite: 1200, hoehe: 800 };

describe('berechneMenueBox', () => {
	it('legt das Menü rechtsbündig unter den Knopf, wenn Platz ist', () => {
		const box = berechneMenueBox(
			{ left: 500, right: 600, top: 100, bottom: 136 },
			200,
			256,
			'rechts',
			fenster
		);
		expect(box).toEqual({ left: 344, top: 140 });
	});
	it('linksbündig trifft die linke Kante des Knopfs', () => {
		const box = berechneMenueBox(
			{ left: 500, right: 600, top: 100, bottom: 136 },
			200,
			256,
			'links',
			fenster
		);
		expect(box.left).toBe(500);
	});
	it('klappt nach oben, wenn unten kein Platz ist und oben mehr', () => {
		const box = berechneMenueBox(
			{ left: 500, right: 600, top: 700, bottom: 736 },
			200,
			256,
			'rechts',
			fenster
		);
		expect(box.top).toBe(700 - 200 - 4);
	});
	it('rückt ins Fenster, wenn es weder darunter noch darüber passt', () => {
		// Knopf etwa in der Mitte, Menü 580 px hoch: unten 384 px Platz, oben 380 px. Nach unten
		// geöffnet lief es 200 px über den Fensterrand, der letzte Eintrag war nicht zu sehen.
		const anker = { left: 500, right: 600, top: 380, bottom: 416 };
		expect(berechneMenueBox(anker, 580, 256, 'rechts', fenster).top).toBe(800 - 580 - 8);
		expect(berechneMenueBox(anker, 900, 256, 'rechts', fenster).top).toBe(8);
	});
	it('bleibt 8 px vom Fensterrand weg', () => {
		const box = berechneMenueBox(
			{ left: 0, right: 40, top: 100, bottom: 136 },
			200,
			256,
			'rechts',
			fenster
		);
		expect(box.left).toBe(8);
	});
});

// Die Vorschau am Menüeintrag geht auf wie ein Untermenü: oben bündig mit dem Eintrag, an der
// hinteren Kante der Menüfläche, links davon, wenn rechts der Platz fehlt (das Menü der
// Signatur im Bestellfenster steht am rechten Rand).
describe('berechneVorschauBox', () => {
	const menue = { left: 700, right: 1000 };
	const flaeche = { breite: 248, hoehe: 184 };

	it('legt die Vorschau rechts an die Menüfläche, oben bündig mit dem Eintrag', () => {
		expect(berechneVorschauBox({ top: 300 }, { left: 300, right: 600 }, flaeche, fenster)).toEqual({
			left: 600,
			top: 300
		});
	});
	it('kippt nach links, wenn rechts der Platz fehlt', () => {
		expect(berechneVorschauBox({ top: 300 }, menue, flaeche, fenster)).toEqual({
			left: 452,
			top: 300
		});
	});
	it('hält 8 px Abstand zum Fensterrand, unten wie links', () => {
		expect(berechneVorschauBox({ top: 700 }, menue, flaeche, fenster).top).toBe(608);
		expect(berechneVorschauBox({ top: 0 }, { left: 100, right: 1100 }, flaeche, fenster)).toEqual({
			left: 8,
			top: 8
		});
	});
});
