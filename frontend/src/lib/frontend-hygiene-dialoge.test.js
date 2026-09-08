import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Ratsche: Rückfragen und Meldungen kommen aus der Anwendung, nicht aus dem Browser.
//
// Anlass (08.09.2026): 19 `confirm()` und 15 `alert()` in 15 Dateien — Browser-Dialoge
// in Systemschrift und Systemfarbe, auf jedem Rechner anders, per Enter blind
// bestätigbar. Zwei Stellen hatten sich einen eigenen M3-Dialog gebaut, der Rest fiel
// auf confirm() zurück, weil es keinen billigen Weg gab. Seither:
//   Rückfrage  → `await bestaetigen({ titel, text, aktion, gefaehrlich })`
//                (stores/bestaetigung.svelte.js, Dialog hängt einmal in App.svelte)
//   Meldung    → toastStore.addToast(text, 'error' | 'info' | …)
//
// Rot bewiesen am 08.09.2026 gegen den Bestand vor der Umstellung (33 Fundstellen).
const BROWSER_DIALOG = /(?<![\w.$])(?:window\.)?(?:confirm|alert|prompt)\s*\(/g;

describe('Keine Browser-Dialoge', () => {
	it('ruft weder confirm() noch alert() noch prompt() auf', () => {
		const treffer = [];
		for (const datei of sammleQuelldateien(srcRoot)) {
			if (!/\.(svelte|js)$/.test(datei) || datei.endsWith('.test.js')) continue;
			const quelle = ohneKommentare(readFileSync(datei, 'utf8'));
			const n = (quelle.match(BROWSER_DIALOG) || []).length;
			if (n) treffer.push(`${relPfad(datei)} (${n})`);
		}
		expect(
			treffer,
			'Browser-Dialog — bitte bestaetigen() aus stores/bestaetigung.svelte.js bzw. toastStore.addToast()'
		).toEqual([]);
	});
});
