import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { sammleQuelldateien, srcRoot, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Knöpfe und Links zeigen ihre Sprechblase über data-tip (actions/tooltip.js, M3 „plain tooltip“):
// nach 150 ms und auch beim Fokus mit der Tastatur. Der title des Browsers kam rund eine Sekunde
// später, in der Optik des Betriebssystems und bei Bedienung mit der Tastatur gar nicht; standen
// beide in einer Maske, sahen zwei Sprechblasen verschieden aus.
//
// Ausnahme: die Ziele der Seitenleiste. An ihrem title findet menuepunkt() in e2e/helpers.js
// jeden Menüpunkt (ARCHITEKTUR 11.5). Sieht nicht: title an anderen Elementen (an gekürztem Text
// zeigt er den vollen Wortlaut) und Attribute, die über {...rest} kommen.
const TITLE_AN_BEDIENELEMENT =
	/<(?:button|Button|a)\b(?:[^>{]|\{(?:[^{}]|\{[^{}]*\})*\})*?\stitle=/;
const AUSNAHMEN = new Map([
	[
		'src/lib/components/layout/SeitenleisteZiele.svelte',
		'menuepunkt() in e2e/helpers.js sucht die Ziele über ihren title (ARCHITEKTUR 11.5)'
	]
]);

describe('Sprechblasen an Knöpfen und Links über data-tip', () => {
	const dateien = sammleQuelldateien(srcRoot).filter((f) => f.endsWith('.svelte'));
	const mitTitle = dateien
		.filter((f) => TITLE_AN_BEDIENELEMENT.test(ohneKommentare(readFileSync(f, 'utf8'))))
		.map(relPfad);

	it('durchsucht überhaupt Dateien und sieht die Ausnahme', () => {
		expect(dateien.length).toBeGreaterThan(300);
		for (const datei of AUSNAHMEN.keys())
			expect(mitTitle, 'Ausnahme ohne Fundstelle').toContain(datei);
	});

	it('kein Knopf und kein Link trägt die Sprechblase des Browsers', () => {
		expect(
			mitTitle.filter((f) => !AUSNAHMEN.has(f)),
			'title an einem Knopf oder Link — data-tip nehmen (actions/tooltip.js)'
		).toEqual([]);
	});

	it('Selbstprobe: der Detektor fasst die Formen', () => {
		for (const form of [
			'<button class="x" title="Kopieren">',
			'<Button variant="secondary"\n\ttitle={text}\n>',
			'<a href={url} title="PDF">',
			'<button onclick={() => (a = b > 1)} title="x">'
		]) {
			expect(TITLE_AN_BEDIENELEMENT.test(form), form).toBe(true);
		}
		for (const form of [
			'<button data-tip="Kopieren">',
			'<span title={voll}>gekürzt</span>',
			'<abbr title="x">',
			'<button aria-label="x">Text</button> <span title="y">'
		]) {
			expect(TITLE_AN_BEDIENELEMENT.test(form), form).toBe(false);
		}
	});
});
