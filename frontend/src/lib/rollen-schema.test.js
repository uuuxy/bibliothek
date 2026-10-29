import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { srcRoot } from './hygiene-quellen.js';

// Das dunkle Schema (.schema-dunkel in styles/rollen.css) überschreibt die Rollen für die ganze
// Seite oder einen Bereich. Fehlt dort eine Rolle, behält sie im Dunkeln ihren hellen Wert:
// dunkle Schrift auf dunklem Grund oder eine weiße Fläche mitten im Monitor. Das helle Schema
// für Papier (.schema-hell) wiederholt die Werte von @theme; weicht es ab, sähe die Vorschau
// anders aus als die Seite im hellen Schema.

const css = readFileSync(join(srcRoot, 'styles', 'rollen.css'), 'utf8');

/** @param {string} anfang */
function werteImBlock(anfang) {
	const start = css.indexOf(anfang);
	if (start < 0) return {};
	const ende = css.indexOf('}', start);
	return Object.fromEntries(
		[...css.slice(start, ende).matchAll(/^\s*(--color-[a-z-]+):\s*([^;]+);/gm)]
			.map((m) => [m[1], m[2].trim()])
			.sort(([a], [b]) => a.localeCompare(b))
	);
}

describe('Rollen: helles und dunkles Schema', () => {
	const hell = werteImBlock('@theme static {');
	const dunkel = werteImBlock('.schema-dunkel {');
	const papier = werteImBlock('.schema-hell {');
	const rollen = (/** @type {object} */ block) => Object.keys(block);

	it('findet alle drei Blöcke — sonst vergliche der Test leere Listen', () => {
		expect(rollen(hell).length).toBeGreaterThan(30);
		expect(rollen(dunkel).length).toBeGreaterThan(30);
		expect(rollen(papier).length).toBeGreaterThan(30);
	});

	it('gibt jeder Rolle des hellen Schemas einen Wert im dunklen', () => {
		expect(
			rollen(hell).filter((r) => !rollen(dunkel).includes(r)),
			'Diese Rollen fehlen in .schema-dunkel und behielten dort ihren hellen Wert.'
		).toEqual([]);
	});

	it('führt im dunklen Schema keine Rolle, die es im hellen nicht gibt', () => {
		expect(rollen(dunkel).filter((r) => !rollen(hell).includes(r))).toEqual([]);
	});

	it('hält das Papier-Schema auf den Werten des hellen', () => {
		expect(papier).toEqual(hell);
	});
});
