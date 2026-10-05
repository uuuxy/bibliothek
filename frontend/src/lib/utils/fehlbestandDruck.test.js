import { describe, it, expect } from 'vitest';
import { baueFehlbestandDruckHtml } from './fehlbestandDruck.js';

// Die Nutzlast steht in jedem Feld, das aus Daten stammt. Ein Test, der nur den Titel
// vergiftet, bliebe grün, wenn eine spätere Spalte unmaskiert einginge.
const NUTZLAST = `<img src="https://fremder-host/?daten=1"><script>alert(1)</script>`;
const TAG = new Date('2026-10-05T10:00:00');
const LANG =
	'Elemente der Mathematik 7 – Gymnasium G9 Hessen: Schülerband mit Lösungen zu den Kontrollaufgaben';

/**
 * Die Zeilen des Tabellenrumpfs, je Zeile das HTML ihrer Zellen.
 * @param {string} html
 */
function rumpf(html) {
	const tbody = html.slice(html.indexOf('<tbody>'));
	return [...tbody.matchAll(/<tr>(.*?)<\/tr>/gs)].map((zeile) => zeile[1]);
}

describe('baueFehlbestandDruckHtml', () => {
	it('lässt aus keinem Feld ein Element ins Dokument', () => {
		const html = baueFehlbestandDruckHtml(
			[{ signatur: NUTZLAST, titel: NUTZLAST, autor: NUTZLAST, barcode_id: NUTZLAST }],
			{ label: NUTZLAST, gebucht: 1 },
			TAG
		);

		expect(html).not.toContain('<img');
		expect(html).not.toContain('<script');
		expect(html).not.toContain('fremder-host/?daten=1"');
		// Nicht-leer-Garantie: Fenstertitel, Überschrift und die vier Zellen tragen die Nutzlast.
		expect(html.match(/&lt;img/g) ?? []).toHaveLength(6);
	});

	it('schreibt kein Skript ins Dokument — die CSP des Openers würde es blockieren', () => {
		const html = baueFehlbestandDruckHtml([], { gebucht: 0 }, TAG);
		expect(html.toLowerCase()).not.toContain('<script');
		expect(html.toLowerCase()).not.toContain('onload=');
	});

	it('druckt je Exemplar eine Zeile mit ganzem Titel und einem Kästchen zum Abhaken', () => {
		const html = baueFehlbestandDruckHtml(
			[{ signatur: 'Ma 7.1', titel: LANG, autor: 'Griesel, Heinz', barcode_id: '10004200' }],
			{ label: 'Lehrbuchsammlung', gebucht: 1 },
			TAG
		);

		expect(rumpf(html)).toEqual([
			`<td class="schmal">Ma 7.1</td><td>${LANG}</td><td>Griesel, Heinz</td>` +
				'<td class="mono schmal">10004200</td><td class="kaestchen schmal"></td>'
		]);
		expect(html).toContain('<h1>Fehlbestand — Lehrbuchsammlung</h1>');
		// Das Kästchen zeichnet das Gerüst; ohne die Regel bliebe die Spalte leer.
		expect(html).toContain('.kaestchen::before');
	});

	it('behält die Reihenfolge des Berichts: So geht man am Regal entlang', () => {
		const html = baueFehlbestandDruckHtml(
			[
				{ signatur: 'Bel SCH', titel: 'Der Vorleser', barcode_id: '2' },
				{ signatur: 'Ma 7.1', titel: 'Elemente der Mathematik 7', barcode_id: '1' }
			],
			{ gebucht: 2 },
			TAG
		);
		const zeilen = rumpf(html);
		expect(zeilen).toHaveLength(2);
		expect(zeilen[0]).toContain('Der Vorleser');
		expect(zeilen[1]).toContain('Elemente der Mathematik 7');
	});

	it('nennt unter der Überschrift, was gebucht, geklärt und noch offen ist', () => {
		const zwei = [
			{ signatur: 'Ma 7.1', titel: 'A', barcode_id: '1' },
			{ signatur: 'Ma 7.1', titel: 'B', barcode_id: '2' }
		];
		expect(baueFehlbestandDruckHtml(zwei, { gebucht: 5 }, TAG)).toContain(
			'Erstellt am: 5.10.2026 | Als Verlust gebucht: 5 | bereits geklärt: 3 | noch offen: 2 | nach Signatur sortiert'
		);
		// Ist nichts geklärt, entfällt der Teil: „bereits geklärt: 0" wäre eine Zeile ohne Aussage.
		expect(baueFehlbestandDruckHtml(zwei, { gebucht: 2 }, TAG)).toContain(
			'Als Verlust gebucht: 2 | noch offen: 2 | nach Signatur sortiert'
		);
	});

	it('kommt ohne Namen der Inventur, Signatur und Autor aus', () => {
		const html = baueFehlbestandDruckHtml(
			[{ titel: 'Diercke Weltatlas', barcode_id: '3' }],
			{ gebucht: 1 },
			TAG
		);
		expect(html).toContain('<h1>Fehlbestand</h1>');
		expect(rumpf(html)[0]).toContain(
			'<td class="schmal">—</td><td>Diercke Weltatlas</td><td></td>'
		);
	});
});
