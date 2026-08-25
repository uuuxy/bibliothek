import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad } from './hygiene-quellen.js';

// Fünfte Struktur-Invariante, dieselbe Bauart wie die Symbol-Ratsche: Farben kommen
// aus den M3-Rollen (styles/rollen.css), nicht aus der Tailwind-Palette.
//
// Warum das zählt: styles/rollen.css trägt das Versprechen „dann ist ein Farbwechsel
// künftig EINE Zeile". Das gilt nur für Fundstellen, die die Rollen auch benutzen.
// Gemessen am 09.08.2026: 100 Rollen-Klassen gegen 2.738 Paletten-Klassen — 3 %.
// Ohne Ratsche wächst der Rückstand mit jeder neuen Komponente weiter, und Dark Mode
// (M3 ist im Kern ein Zwei-Schema-System) wird Jahr für Jahr unerreichbarer.
//
// Warum hier eine ZAHL steht und keine Dateiliste wie bei den Symbolen: Paletten-
// Klassen stehen in fast jeder Datei. Eine Liste wäre der Dateibaum und würde nichts
// aussagen. Die Ratsche greift deshalb an der Summe.
const PALETTE =
	/\b(?:bg|text|border|ring|from|to|via|fill|stroke|divide|outline|decoration|accent|caret|shadow)-(?:slate|gray|zinc|neutral|stone|blue|indigo|violet|purple|rose|red|green|emerald|amber|yellow|orange|teal|cyan|sky|pink|fuchsia|lime)-\d{2,3}\b/g;

// ── Ratsche ─────────────────────────────────────────────────────────────────
// Diese Zahl darf NUR sinken. Wer Fundstellen auf die Rollen umstellt, trägt den
// neuen Stand hier ein — der Test sagt einem die Zahl.
//
// Sie ist ein Bestand, KEINE Erlaubnis: Neues gehört auf bg-surface,
// text-on-surface-variant, border-outline-variant.
const PALETTE_BESTAND = 2605;

// Warum das nicht in einem Durchgang umgeschrieben wird (gemessen am 09.08.2026, damit
// es niemand ein zweites Mal untersuchen muss):
//
//  1. Die Werte sind auseinandergedriftet. Nur zwei der sieben häufigsten Klassen
//     treffen ihre Rolle exakt (blue-600 = primary = #0061a4, slate-50 = surface =
//     #faf9fd). slate-500 (#4c5158) gegen on-surface-variant (#42474e), slate-200
//     (#e3e2e6) gegen outline-variant (#c2c7cf): Jede Umschreibung verschiebt die Farbe.
//  2. Die Palette führt SECHS Textgraustufen (slate-400…900), M3 kennt dafür zwei
//     Rollen. slate-400 und slate-500 landen beide auf on-surface-variant — ein
//     Massen-Rename ebnet eine bestehende Hierarchie ein.
//
// Ein Refactoring ist also möglich, aber es ist eine Umgestaltung und keine Umbenennung:
// in Portionen, bei denen man sieht, was sich ändert. Die Ratsche hält den Stand, bis
// jemand dazu kommt.

/** Zählt die Paletten-Fundstellen je Datei. */
function zaehleProDatei() {
	/** @type {{ datei: string, treffer: number }[]} */
	const out = [];
	for (const f of sammleQuelldateien(srcRoot)) {
		const treffer = (readFileSync(f, 'utf8').match(PALETTE) ?? []).length;
		if (treffer > 0) out.push({ datei: relPfad(f), treffer });
	}
	return out.sort((a, b) => b.treffer - a.treffer);
}

describe('Farb-Hygiene', () => {
	it('führt keine neuen Tailwind-Paletten-Farben ein (Farben kommen aus den M3-Rollen)', () => {
		const proDatei = zaehleProDatei();
		const summe = proDatei.reduce((n, e) => n + e.treffer, 0);

		const spitzenreiter = proDatei
			.slice(0, 8)
			.map((e) => `  ${String(e.treffer).padStart(4)}  ${e.datei}`)
			.join('\n');

		expect(
			summe,
			`Neue Paletten-Farben: ${summe} statt ${PALETTE_BESTAND} (+${summe - PALETTE_BESTAND}).\n` +
				`Farben gehören in die M3-Rollen aus styles/rollen.css:\n` +
				`  Fläche       bg-surface / bg-surface-container-low / bg-surface-container\n` +
				`  Text         text-on-surface (primär), text-on-surface-variant (sekundär)\n` +
				`  Linie        border-outline-variant\n` +
				`  Aktion       bg-primary / text-on-primary / bg-secondary-container\n` +
				`  Fehler       text-error / bg-error-container\n` +
				`Dateien mit den meisten Fundstellen:\n${spitzenreiter}`
		).toBeLessThanOrEqual(PALETTE_BESTAND);

		expect(
			summe,
			`${PALETTE_BESTAND - summe} Fundstelle(n) sind auf die M3-Rollen umgestellt — danke.\n` +
				`Bitte PALETTE_BESTAND in dieser Datei auf ${summe} setzen, damit die Ratsche greift.`
		).toBe(PALETTE_BESTAND);
	});
});
