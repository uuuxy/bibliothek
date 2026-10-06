import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { srcRoot } from './hygiene-quellen.js';

// Das dunkle Schema (.schema-dunkel in styles/rollen.css) überschreibt die Rollen für einen
// Bereich. Fehlt dort eine Rolle, behält sie im dunklen Bereich ihren hellen Wert: dunkle
// Schrift auf dunklem Grund oder eine weiße Fläche mitten im Monitor.

const css = readFileSync(join(srcRoot, 'styles', 'rollen.css'), 'utf8');

/** @param {string} anfang */
function rollenImBlock(anfang) {
	const start = css.indexOf(anfang);
	if (start < 0) return [];
	const ende = css.indexOf('\n}', start);
	return [...css.slice(start, ende).matchAll(/^\s*(--color-[a-z-]+):/gm)].map((m) => m[1]).sort();
}

describe('Rollen: helles und dunkles Schema', () => {
	const hell = rollenImBlock('@theme static {');
	const dunkel = rollenImBlock('.schema-dunkel {');

	it('findet beide Blöcke — sonst vergliche der Test zwei leere Listen', () => {
		expect(hell.length).toBeGreaterThan(30);
		expect(dunkel.length).toBeGreaterThan(30);
	});

	it('gibt jeder Rolle des hellen Schemas einen Wert im dunklen', () => {
		expect(
			hell.filter((r) => !dunkel.includes(r)),
			'Diese Rollen fehlen in .schema-dunkel und behielten dort ihren hellen Wert.'
		).toEqual([]);
	});

	it('führt im dunklen Schema keine Rolle, die es im hellen nicht gibt', () => {
		expect(dunkel.filter((r) => !hell.includes(r))).toEqual([]);
	});
});
