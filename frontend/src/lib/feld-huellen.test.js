import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad } from './hygiene-quellen.js';

// Ein SettingField darf nicht in einer <div>-Hülle stecken, die die Spaltenbreite setzt.
//
// Warum das eine Regel ist und keine Geschmacksfrage: Das Feld verteilt seine drei
// Zeilen (Beschriftung, Feld, Hinweis) als `grid-rows-subgrid` im Raster des Aufrufers —
// so stehen alle Felder einer Reihe auf gleicher Höhe, egal wie lang eine Beschriftung
// ist. Eine Hülle ist selbst das Rasterelement und spannt EINE Zeile, während ihre
// Nachbarn drei spannen; das Feld darin rutscht aus der Reihe. Richtig ist
// `class="sm:col-span-2"` AM FELD.
//
// Der Fehler ist am 23.08.2026 zweimal aufgetreten: erst durch eine umbrechende
// Beschriftung (behoben mit subgrid), dann eine Stunde später durch genau so eine Hülle
// im Anliegen-Formular. Die Messung im Browser (e2e/helpers.js, pruefeFeldreihen) sieht
// die WIRKUNG; diese Regel schließt die URSACHE aus — auch dort, wo gerade kein
// e2e-Test hinsieht.
const HUELLE = /<div[^>]*class="[^"]*col-span[^"]*"[^>]*>\s*<SettingField/;

describe('Feld-Hüllen', () => {
	it('packt kein SettingField in eine <div>-Hülle mit col-span', () => {
		const betroffen = sammleQuelldateien(srcRoot)
			.filter((f) => HUELLE.test(readFileSync(f, 'utf8')))
			.map(relPfad)
			.sort();

		expect(
			betroffen,
			`Ein SettingField steckt in einer <div class="… col-span-… ">-Hülle:\n  ` +
				`${betroffen.join('\n  ')}\n` +
				`Die Hülle spannt eine Rasterzeile, das Feld braucht drei — es rutscht aus der Reihe.\n` +
				`Richtig ist die Spaltenangabe AM FELD: <SettingField … class="sm:col-span-2" />`
		).toEqual([]);
	});

	it('erkennt so eine Hülle überhaupt', () => {
		// Gegenprobe am DETEKTOR: Ein Muster, das nichts findet, meldet ewig „alles gut".
		expect(HUELLE.test('<div class="sm:col-span-2">\n\t<SettingField bind:value={x} />')).toBe(
			true
		);
		expect(HUELLE.test('<div class="md:col-span-2">\n<SettingField />')).toBe(true);
		// Erlaubt: Spaltenangabe am Feld selbst, und Hüllen ohne col-span.
		expect(HUELLE.test('<SettingField class="sm:col-span-2" />')).toBe(false);
		expect(HUELLE.test('<div class="max-w-xs">\n\t<SettingField />')).toBe(false);
	});
});
