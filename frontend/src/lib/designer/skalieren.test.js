import { describe, it, expect } from 'vitest';
import { skaliere } from './skalieren.js';

// Jeder Griff zieht seine Ecke; die gegenüberliegende bleibt stehen.
const START = { x: 10, y: 10, width: 20, height: 10 };

/** @param {{ x: number, y: number, width: number, height: number }} m */
const ecken = (m) => ({
	nw: [m.x, m.y],
	ne: [m.x + m.width, m.y],
	sw: [m.x, m.y + m.height],
	se: [m.x + m.width, m.y + m.height]
});
const GEGENUEBER = /** @type {const} */ ({ nw: 'se', ne: 'sw', sw: 'ne', se: 'nw' });

describe('skaliere: die gegenüberliegende Ecke bleibt stehen', () => {
	it.each(/** @type {const} */ (['nw', 'ne', 'sw', 'se']))('frei, Griff %s', (ecke) => {
		const neu = skaliere(ecke, START, ecke.endsWith('e') ? 4 : -4, ecke.startsWith('s') ? 2 : -2);
		expect([neu.width, neu.height]).toEqual([24, 12]);
		expect(ecken(neu)[GEGENUEBER[ecke]]).toEqual(ecken(START)[GEGENUEBER[ecke]]);
	});

	it.each(/** @type {const} */ (['nw', 'ne', 'sw', 'se']))(
		'proportional, Griff %s: die Höhe folgt der Breite',
		(ecke) => {
			const neu = skaliere(ecke, { ...START, proportional: true }, ecke.endsWith('e') ? 4 : -4, 7);
			expect([neu.width, neu.height]).toEqual([24, 12]);
			expect(ecken(neu)[GEGENUEBER[ecke]]).toEqual(ecken(START)[GEGENUEBER[ecke]]);
		}
	);

	it('höchstens bis 5 mm Breite und 3 mm Höhe', () => {
		expect(skaliere('se', START, -30, -30)).toEqual({ x: 10, y: 10, width: 5, height: 3 });
	});

	it('links und oben nicht über den Kartenrand hinaus', () => {
		expect(skaliere('nw', START, -40, -40)).toMatchObject({ x: 0, y: 0 });
	});
});
