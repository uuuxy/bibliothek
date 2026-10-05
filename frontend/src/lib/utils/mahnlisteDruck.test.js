import { describe, it, expect } from 'vitest';
import { baueMahnlisteDruckHtml } from './mahnlisteDruck.js';

// Die Nutzlast steht in jedem Feld, das aus Daten oder aus der Eingabe stammt. Ein Test,
// der nur den Namen vergiftet, bliebe grün, wenn eine spätere Spalte unmaskiert einginge.
const NUTZLAST = `<img src="https://fremder-host/?daten=1"><script>alert(1)</script>`;
const ALLES = { ansicht: 'Alle', klasse: '', suche: '' };
const TAG = new Date('2026-10-05T10:00:00');

/**
 * Die Zellen des Tabellenrumpfs, Zeile für Zeile.
 * @param {string} html
 */
function rumpf(html) {
	const tbody = html.slice(html.indexOf('<tbody>'));
	return [...tbody.matchAll(/<tr>(.*?)<\/tr>/gs)].map((zeile) =>
		[...zeile[1].matchAll(/<td[^>]*>(.*?)<\/td>/gs)].map((zelle) => zelle[1])
	);
}

describe('baueMahnlisteDruckHtml', () => {
	it('lässt aus keinem Feld ein Element ins Dokument', () => {
		const html = baueMahnlisteDruckHtml(
			[
				{
					name: NUTZLAST,
					klasse: NUTZLAST,
					medien: [
						{
							titel: NUTZLAST,
							faellig_am: NUTZLAST,
							mahnstufe: NUTZLAST,
							letztes_mahndatum: NUTZLAST
						}
					]
				}
			],
			{ ansicht: NUTZLAST, klasse: NUTZLAST, suche: NUTZLAST },
			TAG
		);

		expect(html).not.toContain('<img');
		expect(html).not.toContain('<script');
		expect(html).not.toContain('fremder-host/?daten=1"');
		// Nicht-leer-Garantie: Die Nutzlast ist angekommen, nur eben maskiert.
		expect(html).toContain('&lt;img');
	});

	it('schreibt kein Skript ins Dokument — die CSP des Openers würde es blockieren', () => {
		const html = baueMahnlisteDruckHtml([], ALLES, TAG);
		expect(html.toLowerCase()).not.toContain('<script');
		expect(html.toLowerCase()).not.toContain('onload=');
	});

	it('druckt je Buch eine Zeile: Klasse, Kind, Buch, Frist und wie oft gemahnt', () => {
		const html = baueMahnlisteDruckHtml(
			[
				{
					name: 'Lena Groß',
					klasse: '08H3',
					medien: [
						{ titel: 'Die Räuber', faellig_am: '24.09.2026', mahnstufe: 0 },
						{
							titel: 'Emil und die Detektive',
							faellig_am: '01.09.2026',
							mahnstufe: 2,
							letztes_mahndatum: '2026-09-26'
						}
					]
				}
			],
			ALLES,
			TAG
		);

		expect(rumpf(html)).toEqual([
			['08H3', 'Lena Groß', 'Die Räuber', '24.09.2026', 'noch nicht gemahnt'],
			[
				'08H3',
				'Lena Groß',
				'Emil und die Detektive',
				'01.09.2026',
				'2× gemahnt, zuletzt 26.09.2026'
			]
		]);
		expect(html).toContain(
			'<th class="schmal">Klasse</th><th>Schüler/in</th><th>Buch</th>' +
				'<th class="schmal">Fällig seit</th><th class="schmal">Gemahnt</th>'
		);
	});

	// Die Liste am Bildschirm ordnet nach Dringlichkeit; auf dem Blatt sucht man ein Kind
	// über seine Klasse.
	it('ordnet nach Klasse und Name, Zahlen in der Klasse als Zahl', () => {
		const kind = (/** @type {string} */ name, /** @type {string} */ klasse) => ({
			name,
			klasse,
			medien: [{ titel: 'Buch', faellig_am: '01.09.2026' }]
		});
		const html = baueMahnlisteDruckHtml(
			[kind('Zoe', '10R'), kind('Ben', '9A'), kind('Anna', '10R'), kind('Erik', 'Ehemalige')],
			ALLES,
			TAG
		);
		expect(rumpf(html).map((z) => `${z[0]} ${z[1]}`)).toEqual([
			'9A Ben',
			'10R Anna',
			'10R Zoe',
			'Ehemalige Erik'
		]);
	});

	// Ein Blatt aus einem Reiter oder einer Klasse sähe ohne diese Zeile aus wie die ganze Liste.
	it('nennt den Ausschnitt und zählt Kinder und Bücher', () => {
		const kinder = [
			{ name: 'A', klasse: '5a', medien: [{ titel: 'X' }, { titel: 'Y' }] },
			{ name: 'B', klasse: '5a', medien: [{ titel: 'Z' }] }
		];
		const meta = (/** @type {string} */ html) => html.match(/<p class="meta">(.*?)<\/p>/)?.[1];

		expect(meta(baueMahnlisteDruckHtml(kinder, ALLES, TAG))).toBe(
			'Erstellt am: 5.10.2026 | Ansicht: Alle | alle Klassen | 2 Kinder, 3 Bücher'
		);
		expect(
			meta(
				baueMahnlisteDruckHtml([kinder[1]], { ansicht: 'Mahnung', klasse: '5a', suche: ' b ' }, TAG)
			)
		).toBe('Erstellt am: 5.10.2026 | Ansicht: Eskaliert | Klasse 5a | Suche: b | 1 Kind, 1 Buch');
	});

	it('nennt im Fenstertitel dasselbe wie in der Überschrift des Blatts', () => {
		const html = baueMahnlisteDruckHtml([], ALLES, TAG);
		expect(html.match(/<title>(.*?)<\/title>/)?.[1]).toBe('Mahnliste');
		expect(html.match(/<h1>(.*?)<\/h1>/)?.[1]).toBe('Mahnliste');
	});
});
