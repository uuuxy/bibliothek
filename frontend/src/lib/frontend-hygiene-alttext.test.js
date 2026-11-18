import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Ratsche: Alt-Texte nach Material 3 (Content design, Alt text).
// - Jedes Bild trägt alt; ein Bild, das nichts sagt, ein leeres: „An image can be marked
//   decorative using a null alt attribute, such as alt=”” in HTML".
// - Höchstens 140 Zeichen: „Don’t write more than 140 characters of alt text."
// - Nicht mit „Bild von" beginnen: „Screen readers announce 'image' when they come across an
//   image", davor stünde es doppelt.
// - Kein Dateiname: „Avoid leaving the automatically generated file number as alt text."
//
// Sieht nicht: Alt-Texte aus Ausdrücken (alt={…}), Bilder aus CSS (background-image) und ob
// der Text zum Bild und zur Bildunterschrift passt; das liest ein Mensch.

const BILD_VON = /^(bild|foto|abbildung|grafik|image|picture|photo)\b/i;
const DATEI = /\.(png|jpe?g|gif|webp|svg|avif)\b|^(img|dsc|bild)[_-]?\d+/i;

/** Die Köpfe aller <img> einer Quelle, bis zum schließenden „>" außerhalb von Klammern. */
function bilder(/** @type {string} */ quelle) {
	/** @type {string[]} */
	const koepfe = [];
	for (
		let start = quelle.indexOf('<img');
		start !== -1;
		start = quelle.indexOf('<img', start + 4)
	) {
		if (/\w/.test(quelle[start + 4] ?? '')) continue;
		let tiefe = 0;
		let zitat = '';
		let i = start + 4;
		for (; i < quelle.length; i++) {
			const z = quelle[i];
			if (zitat) zitat = z === zitat ? '' : zitat;
			else if (z === '"' || z === "'" || z === '`') zitat = z;
			else if (z === '{') tiefe++;
			else if (z === '}') tiefe--;
			else if (z === '>' && tiefe === 0) break;
		}
		koepfe.push(quelle.slice(start, i + 1));
	}
	return koepfe;
}

describe('Alt-Texte nach M3', () => {
	const dateien = sammleQuelldateien(srcRoot).filter((f) => f.endsWith('.svelte'));
	/** @type {{ pfad: string, kopf: string }[]} */
	const alle = dateien.flatMap((f) =>
		bilder(ohneKommentare(readFileSync(f, 'utf8'))).map((kopf) => ({ pfad: relPfad(f), kopf }))
	);

	it('gibt jedem Bild ein alt, leer nur für Schmuck', () => {
		const ohne = alle.filter(
			(b) => !/\salt(=|\s|>|\/)/.test(b.kopf) && !/\{\s*\.\.\./.test(b.kopf)
		);
		expect(
			ohne.map((b) => `${b.pfad}: ${b.kopf.slice(0, 80)}`),
			'Bild ohne alt'
		).toEqual([]);
	});

	it('schreibt feste Alt-Texte kurz, ohne „Bild von" und ohne Dateinamen', () => {
		/** @type {string[]} */
		const funde = [];
		for (const b of alle) {
			const fest = /\salt="([^"]*)"/.exec(b.kopf)?.[1];
			if (!fest) continue;
			if (fest.length > 140) funde.push(`${b.pfad}: länger als 140 Zeichen`);
			if (BILD_VON.test(fest.trim())) funde.push(`${b.pfad}: „${fest}" beginnt mit „Bild"`);
			if (DATEI.test(fest)) funde.push(`${b.pfad}: „${fest}" ist ein Dateiname`);
		}
		expect(funde, 'Alt-Text abseits von M3 (Content design, Alt text)').toEqual([]);
	});

	it('findet die Bilder', () => {
		expect(alle.length).toBeGreaterThan(8);
	});
});
