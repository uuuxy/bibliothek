import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Ratsche: Kästchen, Auswahlknöpfe und Schalter kommen aus ui/ — nicht von Hand.
//
// Anlass (08.09.2026): 22 native <input type="checkbox"> und 5 <input type="radio">
// in 19 Dateien, dazu zwei handgebaute Schalter (sr-only + peer-checked, 40×24 und
// 28×16 px) neben dem M3-Switch (52×32). Die Kästchen trugen Klassen des Tailwind-
// Forms-Plugins, das nie installiert war — im Browser blieb Chromes eigenes Blau in
// drei Größen. Gemessen am gebauten Stylesheet, nicht gegreppt.
//
// Ein natives Kästchen lässt sich nicht nach Material 3 einkleiden. Deshalb gibt es
// seither GENAU EIN <input type="checkbox"> (ui/Kaestchen.svelte), GENAU EIN
// <input type="radio"> (ui/Radio.svelte) und keinen peer-checked-Nachbau mehr —
// Zustände schaltet ui/Switch.svelte.
//
// Rot bewiesen am 08.09.2026 gegen den Bestand vor der Umstellung (27 Fundstellen).
const ERLAUBT = new Set([
	'src/lib/components/ui/Kaestchen.svelte',
	'src/lib/components/ui/Radio.svelte'
]);
const NATIV = /<input\b[^>]*\btype="(?:checkbox|radio)"/gs;
const NACHBAU = /peer-checked/;

function fundstellen(regex) {
	const treffer = [];
	for (const datei of sammleQuelldateien(srcRoot)) {
		if (!datei.endsWith('.svelte')) continue;
		const rel = relPfad(datei);
		if (ERLAUBT.has(rel)) continue;
		const quelle = ohneKommentare(readFileSync(datei, 'utf8'));
		const n = (quelle.match(regex) || []).length;
		if (n) treffer.push(`${rel} (${n})`);
	}
	return treffer;
}

describe('Kästchen und Schalter kommen aus ui/', () => {
	it('kennt kein natives <input type="checkbox"|"radio"> außerhalb von ui/Kaestchen und ui/Radio', () => {
		expect(
			fundstellen(NATIV),
			'Natives Kästchen/Radio — bitte ui/Kaestchen.svelte bzw. ui/Radio.svelte nehmen'
		).toEqual([]);
	});

	it('baut keinen Schalter aus sr-only + peer-checked nach (Zustände schaltet ui/Switch)', () => {
		expect(fundstellen(NACHBAU), 'peer-checked-Nachbau — bitte ui/Switch.svelte nehmen').toEqual(
			[]
		);
	});
});
