import { describe, it, expect } from 'vitest';
import { passbildAusschnitt, PASSBILD_FORM } from './passbildAusschnitt.js';

describe('passbildAusschnitt', () => {
	it('nimmt aus einem breiten Kamerabild die volle Höhe und die Mitte', () => {
		expect(passbildAusschnitt(1280, 720)).toEqual({ x: 370, y: 0, breite: 540, hoehe: 720 });
		expect(passbildAusschnitt(1920, 1080)).toEqual({ x: 555, y: 0, breite: 810, hoehe: 1080 });
		expect(passbildAusschnitt(640, 480)).toEqual({ x: 140, y: 0, breite: 360, hoehe: 480 });
	});

	// Eine hochkant gehaltene Kamera liefert ein Bild, das schmaler ist als das Passbild. Der
	// Ausschnitt darf dann nicht breiter sein als das Bild, sonst stünde ein schwarzer Rand im Foto.
	it('nimmt aus einem schmalen Kamerabild die volle Breite und die Mitte', () => {
		expect(passbildAusschnitt(720, 1280)).toEqual({ x: 0, y: 160, breite: 720, hoehe: 960 });
	});

	it('nimmt ein Bild, das schon die Form hat, ganz', () => {
		expect(passbildAusschnitt(600, 800)).toEqual({ x: 0, y: 0, breite: 600, hoehe: 800 });
	});

	it('liefert für jede Kamera die Form 3:4 innerhalb des Bilds', () => {
		for (const [b, h] of [
			[1280, 720],
			[640, 480],
			[720, 1280],
			[1000, 1000],
			[333, 777]
		]) {
			const a = passbildAusschnitt(b, h);
			expect(Math.abs(a.breite / a.hoehe - PASSBILD_FORM)).toBeLessThan(0.002);
			expect(a.x >= 0 && a.y >= 0 && a.x + a.breite <= b && a.y + a.hoehe <= h).toBe(true);
		}
	});
});
