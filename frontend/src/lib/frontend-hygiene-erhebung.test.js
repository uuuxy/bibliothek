import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Ratsche: Erhebung nur in Googles Stufen (M3, Elevation; material-web
// elevation/internal/_elevation.scss). styles/theme-mass.css führt level1 bis level5 als
// shadow-level1 … shadow-level5; eine andere Schatten-Klasse oder ein eigener box-shadow in einem
// Stil zeichnete einen Schatten, den Google nicht kennt. Ein Schatten nach innen (inset) ist keine
// Erhebung und bleibt erlaubt, ebenso `none` für den Druck.
//
// Sieht nicht: Klassen aus Ausdrücken, die erst zur Laufzeit entstehen, und ob ein Bauteil die
// Stufe seines Tokens trägt; das misst e2e/erhebung.spec.js an Dialog, Kalender, Menü und
// Snackbar.

const KLASSE = /(?<![\w-])((?:[\w-]+:)*)shadow-(?!level[1-5]\b|none\b)([\w[\]/.-]+)/g;
const STIL = /box-shadow:\s*([^;}]+)/g;

describe('Erhebung nach M3', () => {
	const dateien = sammleQuelldateien(srcRoot).filter((f) => /\.(svelte|js)$/.test(f));

	it('nimmt nur Googles Stufen als Schatten-Klasse', () => {
		/** @type {string[]} */
		const funde = [];
		for (const f of dateien) {
			if (f.endsWith('.test.js')) continue;
			const quelle = ohneKommentare(readFileSync(f, 'utf8'));
			for (const [ganz] of quelle.matchAll(KLASSE)) funde.push(`${relPfad(f)}: ${ganz}`);
		}
		expect(funde, 'Schatten abseits von Googles Stufen: shadow-level1 … shadow-level5').toEqual([]);
	});

	it('zeichnet in Stilen keinen eigenen Schatten nach außen', () => {
		/** @type {string[]} */
		const funde = [];
		for (const f of dateien.filter((f) => f.endsWith('.svelte'))) {
			const quelle = ohneKommentare(readFileSync(f, 'utf8'));
			for (const [, wert] of quelle.matchAll(STIL)) {
				const teile = wert.split(/,(?![^(]*\))/).map((t) => t.trim());
				if (teile.every((t) => /^none\b/.test(t) || /\binset\b/.test(t))) continue;
				funde.push(`${relPfad(f)}: box-shadow: ${wert.trim().slice(0, 60)}`);
			}
		}
		expect(funde, 'eigener Schatten in einem Stil: shadow-level… nehmen').toEqual([]);
	});

	it('findet die Stufen und ihre Verwendung', () => {
		const css = readFileSync(`${srcRoot}/styles/theme-mass.css`, 'utf8');
		for (const n of [1, 2, 3, 4, 5]) expect(css).toContain(`--shadow-level${n}:`);
		const verwendet = dateien.filter((f) => /shadow-level[1-3]/.test(readFileSync(f, 'utf8')));
		expect(verwendet.length).toBeGreaterThan(10);
	});
});
