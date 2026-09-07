import { describe, it, expect } from 'vitest';
import { berechneMenueBox } from './menueGeometrie.js';

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
