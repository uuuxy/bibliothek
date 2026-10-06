import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { srcRoot, sammleQuelldateien, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Farben kommen aus den M3-Rollen (styles/rollen.css), nicht aus der Tailwind-Palette und
// nicht als Weiß oder Schwarz: Nur für Fundstellen, die die Rollen benutzen, ist ein
// Farbwechsel eine Zeile und ein zweites Farbschema erreichbar.
//
// Mit Rahmenseite: `border-l-amber-500` färbt eine Kante und ist dieselbe Sache wie
// `border-amber-500`.
//
// BLINDHEIT: Nur literale Klassen. Eine zusammengesetzte Klasse (`'bg-' + farbe`), ein
// Farbwert im style-Attribut, in einer .css-Datei oder in einem SVG-Attribut (`fill="#…"`)
// sieht der Test nicht. Die Farben der Ausweiskarte (designer/kartenFarben.js) sind Werte
// des Entwurfs und gehören nicht hierher.
const FAMILIE =
	'bg|text|border(?:-[trblxyse])?|ring|from|to|via|fill|stroke|divide|outline|decoration|accent|caret|shadow';
const PALETTE = new RegExp(
	`\\b(?:${FAMILIE})-(?:slate|gray|zinc|neutral|stone|blue|indigo|violet|purple|rose|red|green|emerald|amber|yellow|orange|teal|cyan|sky|pink|fuchsia|lime)-\\d{2,3}\\b`,
	'g'
);
const WEISS_SCHWARZ = new RegExp(`(?<![\\w-])(?:${FAMILIE})-(?:white|black)(?![\\w-])`, 'g');

// Eine Klasse, die wie eine Rolle heißt. Was dahinter steht, muss rollen.css kennen.
const ROLLENKLASSE = new RegExp(
	`(?<![\\w-])(?:${FAMILIE}|ring-offset|placeholder)-((?:on-)?(?:primary|secondary|tertiary|error|success|warning|surface|outline|inverse|scrim|background)[a-z0-9-]*)`,
	'g'
);

function bekannteRollen() {
	const css = readFileSync(join(srcRoot, 'styles', 'rollen.css'), 'utf8');
	return new Set([...css.matchAll(/^\s*--color-([a-z0-9-]+):/gm)].map((m) => m[1]));
}

/**
 * @param {RegExp} muster
 * @param {(quelle: string) => string} [vorbereiten]
 */
function fundeJeDatei(muster, vorbereiten = (q) => q) {
	/** @type {string[]} */
	const funde = [];
	let dateien = 0;
	for (const f of sammleQuelldateien(srcRoot)) {
		dateien++;
		const treffer = vorbereiten(readFileSync(f, 'utf8')).match(muster) ?? [];
		if (treffer.length > 0) funde.push(`${relPfad(f)}: ${[...new Set(treffer)].join(', ')}`);
	}
	return { funde, dateien };
}

const HINWEIS =
	'Farben gehören in die M3-Rollen aus styles/rollen.css:\n' +
	'  Fläche       bg-surface / bg-surface-container-low / bg-surface-container-lowest\n' +
	'  Text         text-on-surface (primär), text-on-surface-variant (sekundär)\n' +
	'  Linie        border-outline-variant\n' +
	'  Aktion       bg-primary / text-on-primary / bg-secondary-container\n' +
	'  Fehler       text-error / bg-error-container\n' +
	'  Schleier     bg-scrim/32';

describe('Farb-Hygiene', () => {
	it('erkennt die Formen, die es finden soll, und lässt Rollen durch', () => {
		// Nicht-leer-Garantie: Der Bestand ist leer. Ein Muster, das nichts mehr fasst, sähe
		// genauso aus.
		const trifft = (/** @type {RegExp} */ muster, /** @type {string} */ text) =>
			(text.match(muster) ?? []).length;
		expect(trifft(PALETTE, 'bg-slate-100 hover:text-blue-600 border-l-amber-500')).toBe(3);
		expect(trifft(PALETTE, 'bg-surface text-on-surface-variant border-outline-variant')).toBe(0);
		expect(trifft(WEISS_SCHWARZ, 'bg-white text-black/80 from-black/20 ring-white')).toBe(4);
		expect(trifft(WEISS_SCHWARZ, 'whitespace-nowrap font-black text-on-primary')).toBe(0);
		expect(bekannteRollen().has('on-surface-variant')).toBe(true);
		expect(bekannteRollen().size).toBeGreaterThan(30);
	});

	it('führt keine Tailwind-Palettenfarbe (Farben kommen aus den M3-Rollen)', () => {
		const { funde, dateien } = fundeJeDatei(PALETTE);
		expect(dateien, 'Der Sammler findet kaum Quelldateien').toBeGreaterThan(200);
		expect(funde, HINWEIS).toEqual([]);
	});

	it('führt kein Weiß und kein Schwarz als Klasse', () => {
		const { funde } = fundeJeDatei(WEISS_SCHWARZ);
		expect(funde, HINWEIS).toEqual([]);
	});

	it('benutzt keine Rolle, die rollen.css nicht definiert', () => {
		const bekannt = bekannteRollen();
		/** @type {string[]} */
		const funde = [];
		for (const f of sammleQuelldateien(srcRoot)) {
			const quelle = ohneKommentare(readFileSync(f, 'utf8'));
			for (const m of quelle.matchAll(ROLLENKLASSE)) {
				if (!bekannt.has(m[1])) funde.push(`${relPfad(f)}: ${m[0]}`);
			}
		}
		expect(
			funde,
			'Diese Farbklasse heißt wie eine Rolle, aber rollen.css kennt sie nicht: Sie färbt nichts.'
		).toEqual([]);
	});
});
