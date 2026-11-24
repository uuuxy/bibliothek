import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import {
	srcRoot,
	sammleQuelldateien,
	relPfad,
	ohneKommentare,
	elemente
} from './hygiene-quellen.js';

// Ratsche: Der Weg zurück ist überall derselbe Symbolknopf mit Googles Pfeil.
//
// Material 3, App bars, Guidelines: „A back arrow, which returns to the previous screen“. Fünf
// Zurück-Stellen hatten fünf Formen: Textknopf mit Chevron, umrandeter Knopf mit Pfeil, Link,
// Symbolknopf mit Chevron. Ein Knopf zurück ist <Button variant="symbol"> mit
// <Icon name="arrow_back" />, ohne Text und mit Namen; erkannt wird er am Pfeil, an einem Namen
// oder Text, der mit „Zurück“ beginnt, oder an einem Pfeil nach links neben Text („‹ Statistik“).
//
// Sieht nicht: einen Weg zurück ohne diese Merkmale (etwa ein Symbolknopf mit Chevron und einem
// Namen ohne „Zurück“) und Knöpfe aus einem anderen Bauteil.

/** Knöpfe, die so aussehen dürfen, je Datei, Tag, Symbol und Text, mit Grund. */
const AUSNAHMEN = {
	// Die Aktion des Leerzustands „Buch nicht gefunden“ steht unter der Meldung, nicht oben.
	'src/lib/components/buchakte/Buchakte.svelte: button: Zurück zum Katalog': 'Leerzustand'
};

/** @param {string} inhalt */
const sichtbarerText = (inhalt) =>
	inhalt
		.replace(/\{[#:/@][^}]*\}/g, ' ')
		.replace(/<(?:=>|[^>])*>/g, ' ')
		.replace(/\s+/g, ' ')
		.trim();

/** @param {string} kopf */
const name = (kopf) => (kopf.match(/\saria-label=(?:"([^"]*)"|\{[^}]*\})/) ?? [])[1] ?? '';

/** @param {string} kopf @param {string} inhalt */
function istZurueck(kopf, inhalt) {
	return (
		/<Icon name="arrow_back"/.test(inhalt) ||
		/^Zurück/.test(name(kopf)) ||
		/^Zurück/.test(sichtbarerText(inhalt)) ||
		(/<Icon name="(?:chevron_left|arrow_back_ios)"/.test(inhalt) && sichtbarerText(inhalt) !== '')
	);
}

/** @param {string} kopf @param {string} inhalt */
function wieGoogle(kopf, inhalt) {
	return (
		/^<Button\b/.test(kopf) &&
		/\svariant="symbol"/.test(kopf) &&
		/\saria-label=/.test(kopf) &&
		/<Icon name="arrow_back"/.test(inhalt) &&
		sichtbarerText(inhalt) === ''
	);
}

function zurueckKnoepfe() {
	/** @type {{ ort: string, schluessel: string, richtig: boolean }[]} */
	const treffer = [];
	for (const datei of sammleQuelldateien(srcRoot)) {
		if (!datei.endsWith('.svelte')) continue;
		const quelle = ohneKommentare(readFileSync(datei, 'utf8'));
		for (const tag of /** @type {const} */ (['button', 'Button'])) {
			for (const k of elemente(quelle, tag)) {
				if (!istZurueck(k.kopf, k.inhalt)) continue;
				const zeile = quelle.slice(0, k.start).split('\n').length;
				const symbol = /<Icon\b/.test(k.inhalt) ? ' mit Symbol' : '';
				treffer.push({
					ort: `${relPfad(datei)}:${zeile}`,
					schluessel: `${relPfad(datei)}: ${tag}${symbol}: ${sichtbarerText(k.inhalt)}`,
					richtig: wieGoogle(k.kopf, k.inhalt)
				});
			}
		}
	}
	return treffer;
}

describe('Zurück ist der Pfeil als Symbolknopf (M3, App bars)', () => {
	it('der Detektor erkennt die alten Formen und lässt den Pfeil gelten', () => {
		const pfeil = '<Icon name="arrow_back" />';
		expect(wieGoogle('<Button variant="symbol" aria-label="Zurück zum Katalog"', pfeil)).toBe(true);
		expect(
			istZurueck('<Button variant="ghost" onclick={x}', '<Icon name="chevron_left" /> Statistik')
		).toBe(true);
		expect(
			wieGoogle('<Button variant="ghost" onclick={x}', '<Icon name="chevron_left" /> Statistik')
		).toBe(false);
		expect(
			istZurueck('<Button variant="ghost"', '<Icon name="chevron_left" /> Zurück zum Katalog')
		).toBe(true);
		expect(wieGoogle('<Button variant="secondary"', `${pfeil} Zurück zur Bestellhistorie`)).toBe(
			false
		);
		expect(wieGoogle('<button class="icon-btn h-10 w-10" aria-label="Zurück"', pfeil)).toBe(false);
		expect(
			wieGoogle('<Button variant="symbol" aria-label="Zurück"', '<Icon name="chevron_left" />')
		).toBe(false);
		expect(
			istZurueck(
				'<Button variant="symbol" aria-label="Vorheriger Monat"',
				'<Icon name="chevron_left" />'
			)
		).toBe(false);
	});

	it('jeder Weg zurück ist der Pfeil als Symbolknopf', () => {
		const knoepfe = zurueckKnoepfe();
		// Fände der Detektor keinen, bliebe die Ratsche still grün.
		expect(knoepfe.length).toBeGreaterThanOrEqual(5);
		expect(
			knoepfe.filter((k) => !k.richtig && !(k.schluessel in AUSNAHMEN)).map((k) => k.ort),
			'Zurück ist <Button variant="symbol" aria-label="…" data-tip="…"><Icon name="arrow_back" />' +
				'</Button> (M3, App bars: „A back arrow, which returns to the previous screen“).'
		).toEqual([]);
	});

	it('jede Ausnahme trifft noch einen Knopf', () => {
		const schluessel = new Set(zurueckKnoepfe().map((k) => k.schluessel));
		expect(Object.keys(AUSNAHMEN).filter((s) => !schluessel.has(s))).toEqual([]);
	});
});
