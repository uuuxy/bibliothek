import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Ratsche: Jedes Schiebefeld ist Googles Slider (styles/komponenten.css, .m3-schieber: Spur
// 16 dp, Griff 4 × 44 dp, M3 Sliders, Specs) und trägt einen Namen. Ein <input type="range">
// ohne die Klasse zeichnete der Browser in seiner eigenen Form.
//
// Sieht nicht: Schiebefelder aus anderen Bauteilen und Klassen aus Ausdrücken.

/** Die Köpfe aller <input> einer Quelle, bis zum „>" außerhalb von Klammern und Anführungszeichen. */
function eingaben(/** @type {string} */ quelle) {
	/** @type {string[]} */
	const koepfe = [];
	for (
		let start = quelle.indexOf('<input');
		start !== -1;
		start = quelle.indexOf('<input', start + 6)
	) {
		let tiefe = 0;
		let zitat = '';
		let i = start + 6;
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

describe('Schiebefelder nach M3', () => {
	it('nimmt für jedes Schiebefeld .m3-schieber und einen Namen', () => {
		/** @type {string[]} */
		const funde = [];
		let gesehen = 0;
		for (const f of sammleQuelldateien(srcRoot).filter((f) => f.endsWith('.svelte'))) {
			const quelle = ohneKommentare(readFileSync(f, 'utf8'));
			for (const kopf of eingaben(quelle).filter((k) => /\stype="range"/.test(k))) {
				gesehen++;
				if (!/\bm3-schieber\b/.test(kopf)) funde.push(`${relPfad(f)}: ohne m3-schieber`);
				if (!/\saria-label(ledby)?=/.test(kopf)) funde.push(`${relPfad(f)}: ohne Namen`);
			}
		}
		expect(gesehen, 'kein Schiebefeld gefunden').toBeGreaterThan(0);
		expect(funde, 'Schiebefeld abseits von M3 (Sliders)').toEqual([]);
	});
});
