import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { sammleQuelldateien, srcRoot, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Zwei Schreibweisen haben einen Helfer: der Betrag in Euro (formatEuro in utils/format.js) und
// der Text eines gefangenen Fehlers (fehlertext in utils/fehlertext.js). Von Hand geschrieben
// laufen die Stellen auseinander, etwa im Leerzeichen vor dem Zeichen und im Tausenderpunkt. Wer
// eine der beiden außerhalb ihres Helfers schreibt, wird hier rot.
//
// Ebenso der Vergleich einer Maske mit ihrem Stand vom Öffnen (nurGeaendertes in
// utils/geaendert.js): Er stand in vier Masken von Hand. Der Titel vergleicht in
// inventur/lib/buch_felder.js mit eigener Regel und anderer Form.
//
// Sieht nicht: einen Betrag ohne Leerzeichen vor dem Zeichen in einer Vorlage (`{wert}€`), ein
// Zeichen aus einer Variablen und einen Fehlertext mit eigenem Ersatzsatz
// (`e instanceof Error ? e.message : 'Export fehlgeschlagen'`); der ist gewollt.
const FORMAT = 'src/lib/utils/format.js';
const BETRAG = /\+\s*['"`](?:\s|\\u00a0)*€|\}(?:\s|\\u00a0|&nbsp;)+€|currency\s*:\s*['"`]EUR/;
const FEHLERTEXT =
	/instanceof\s+Error\s*\?\s*[\w$.]+\.message\s*:\s*String\(|String\(\s*[\w$.]+\s+instanceof\s+Error\s*\?/;
const GEAENDERT = 'src/lib/utils/geaendert.js';
const VERGLEICH_MIT_GELADEN = /!==\s*(?:[\w$]+\.)*geladen\[/;

describe('Betrag und Fehlertext nur über ihre Helfer', () => {
	const dateien = sammleQuelldateien(srcRoot);
	/** @param {RegExp} muster */
	const treffer = (muster) =>
		dateien.filter((f) => muster.test(ohneKommentare(readFileSync(f, 'utf8')))).map(relPfad);

	it('durchsucht überhaupt Dateien', () => {
		expect(dateien.length).toBeGreaterThan(300);
	});

	it('keine Quelldatei außer utils/format.js schreibt einen Betrag in Euro von Hand', () => {
		const von_hand = treffer(BETRAG);
		// Sähe der Detektor die erlaubte Stelle nicht, mäße er nichts.
		expect(von_hand, 'der Detektor findet formatEuro nicht').toContain(FORMAT);
		expect(
			von_hand.filter((f) => f !== FORMAT),
			'Betrag von Hand geschrieben — formatEuro aus utils/format.js nehmen'
		).toEqual([]);
	});

	it('keine Quelldatei schreibt den Text eines gefangenen Fehlers von Hand', () => {
		expect(
			treffer(FEHLERTEXT),
			'Fehlertext von Hand geschrieben — fehlertext aus utils/fehlertext.js nehmen'
		).toEqual([]);
	});

	it('keine Maske vergleicht von Hand mit ihrem Stand vom Öffnen', () => {
		const von_hand = treffer(VERGLEICH_MIT_GELADEN);
		expect(von_hand, 'der Detektor findet nurGeaendertes nicht').toContain(GEAENDERT);
		expect(
			von_hand.filter((f) => f !== GEAENDERT),
			'Vergleich mit dem geladenen Stand von Hand — nurGeaendertes aus utils/geaendert.js nehmen'
		).toEqual([]);
		for (const form of [
			'.filter(([name, wert]) => wert !== form.geladen[name])',
			'AENDERBAR.filter((name) => form[name] !== form.geladen[name])',
			'.filter((name) => jetzt[name] !== geladen[name])'
		]) {
			expect(VERGLEICH_MIT_GELADEN.test(form), form).toBe(true);
		}
		expect(VERGLEICH_MIT_GELADEN.test('return nurGeaendertes(form.geladen, felder);')).toBe(false);
	});

	it('Selbstprobe: die Detektoren fassen die Formen', () => {
		for (const form of [
			"return n.toLocaleString('de-DE', { minimumFractionDigits: 2 }) + ' €';",
			"const euro = (b) => b.toFixed(2).replace('.', ',') + ' €';",
			'const text = wert.toFixed(2) + "\\u00a0€";',
			'const satz = `Die Forderung über ${betrag} € wurde storniert.`;',
			'<td>{zeile.summe.toFixed(2)} €</td>',
			'<td>{zeile.summe.toFixed(2)}&nbsp;€</td>',
			"wert.toLocaleString('de-DE', { style: 'currency', currency: 'EUR' })"
		]) {
			expect(BETRAG.test(form), form).toBe(true);
		}
		// Das Zeichen allein hinter einem Feld ist kein Betrag.
		expect(BETRAG.test('{#snippet nachlaufend()}€{/snippet}')).toBe(false);
		expect(BETRAG.test('<span class="text-sm">€</span>')).toBe(false);

		for (const form of [
			'error = err instanceof Error ? err.message : String(err);',
			'showToast(fehler instanceof Error ? fehler.message : String(fehler));',
			'fehler = String(e instanceof Error ? e.message : e);'
		]) {
			expect(FEHLERTEXT.test(form), form).toBe(true);
		}
		expect(
			FEHLERTEXT.test("return e instanceof Error ? e.message : 'Zuordnen fehlgeschlagen';")
		).toBe(false);
		expect(FEHLERTEXT.test('if (e instanceof Error) return e.message;')).toBe(false);
	});
});
