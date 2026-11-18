import { describe, it, expect } from 'vitest';
import {
	standardDecelerate,
	standardAccelerate,
	emphasizedDecelerate,
	emphasizedAccelerate
} from './bewegung.js';

const KURVEN = {
	standardDecelerate,
	standardAccelerate,
	emphasizedDecelerate,
	emphasizedAccelerate
};

describe('M3-Kurven für Svelte-Übergänge', () => {
	for (const [name, kurve] of Object.entries(KURVEN)) {
		it(`${name} beginnt bei 0, endet bei 1 und steigt dazwischen`, () => {
			expect(kurve(0)).toBe(0);
			expect(kurve(1)).toBe(1);
			let vorher = 0;
			for (let i = 1; i < 100; i++) {
				const wert = kurve(i / 100);
				expect(wert).toBeGreaterThanOrEqual(vorher);
				vorher = wert;
			}
		});
	}

	it('bremst beim Eintreten und beschleunigt beim Austreten', () => {
		expect(emphasizedDecelerate(0.5)).toBeGreaterThan(0.9);
		expect(standardDecelerate(0.5)).toBeGreaterThan(0.8);
		expect(emphasizedAccelerate(0.5)).toBeLessThan(0.2);
		expect(standardAccelerate(0.5)).toBeLessThan(0.4);
	});

	it('trifft die Kurve, die CSS mit denselben Werten zeichnet', () => {
		// cubic-bezier(0, 0, 0, 1) bei s = 0.5: x = s³ = 0.125, y = 3s²(1 − s) + s³ = 0.5.
		expect(standardDecelerate(0.125)).toBeCloseTo(0.5, 6);
	});
});
