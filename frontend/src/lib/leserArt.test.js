import { describe, it, expect } from 'vitest';
import { LESER_ARTEN, leserArtText, istKollegium, artMitKonto, ausweisTitel } from './leserArt.js';
import pruefung from './leserArt.faelle.json';

// Die Arten eines Lesers, Browser-Seite der geteilten Prüffälle. Der Server liest dieselbe
// Datei in TestLeserArten_WieInDerDatenbankUndImBrowser und hält sie dort zusätzlich gegen
// chk_leser_art (Migration 153). Weicht eine Seite ab, hieße ein Praktikum im Browser
// „Schüler" oder bekäme dort ein Pflichtfeld „Schul-E-Mail", das der Server abweist.
describe('Arten eines Lesers, geteilte Prüffälle', () => {
	it('liest die gemeinsame Datei überhaupt ein', () => {
		// Ohne diesen Boden liefe die Suite still grün, wenn die Datei umbenannt wird.
		expect(pruefung.arten.length).toBeGreaterThanOrEqual(7);
	});

	it('bietet dieselben Arten in derselben Reihenfolge an', () => {
		expect(LESER_ARTEN).toEqual(pruefung.arten.map((a) => a.art));
	});

	for (const fall of pruefung.arten) {
		it(`${fall.art}: Wort, Kollegium, Zugang und Ausweis`, () => {
			expect(leserArtText(fall.art)).toBe(fall.text);
			expect(istKollegium({ art: fall.art })).toBe(fall.kollegium);
			expect(artMitKonto(fall.art)).toBe(fall.mitKonto);
			expect(ausweisTitel(fall.art)).toBe(fall.ausweis);
		});
	}

	it('liest eine Zeile ohne Art als Schüler — die Vorgabe der Spalte', () => {
		expect(leserArtText(undefined)).toBe('Schüler');
		expect(istKollegium({ art: undefined })).toBe(false);
		expect(ausweisTitel(undefined)).toBe('Schülerausweis');
	});
});
