import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { sammleQuelldateien, srcRoot, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Jede Sprechblase kommt über data-tip (actions/tooltip.js, M3 „plain tooltip“): nach 150 ms und
// auch beim Fokus mit der Tastatur. Der title des Browsers kam rund eine Sekunde später, in der
// Optik des Betriebssystems und bei Bedienung mit der Tastatur gar nicht; standen beide in einer
// Maske, sahen zwei Sprechblasen verschieden aus. Das gilt auch für gekürzten Text, wie im Titel
// der Leserakte. Sagt die Sprechblase etwas, das sonst nicht dasteht, braucht der Bildschirmleser
// es daneben als sr-only oder aria-label: data-tip liest er nicht vor.
//
// Ausnahme: die Ziele der Seitenleiste. An ihrem title findet menuepunkt() in e2e/helpers.js
// jeden Menüpunkt (ARCHITEKTUR 11.5). Ausgenommen ist auch <abbr>, dessen title die Abkürzung
// auflöst. Sieht nicht: Attribute, die über {...rest} kommen.
const TITLE_AN_BEDIENELEMENT =
	/<(?!title\b|abbr\b)[A-Za-z][\w.:-]*\b(?:[^>{]|\{(?:[^{}]|\{[^{}]*\})*\})*?\s(?:title=|\{title\})/;
const AUSNAHMEN = new Map([
	[
		'src/lib/components/layout/SeitenleisteZiele.svelte',
		'menuepunkt() in e2e/helpers.js sucht die Ziele über ihren title (ARCHITEKTUR 11.5)'
	]
]);

describe('Sprechblasen über data-tip', () => {
	const dateien = sammleQuelldateien(srcRoot).filter((f) => f.endsWith('.svelte'));
	const mitTitle = dateien
		.filter((f) => TITLE_AN_BEDIENELEMENT.test(ohneKommentare(readFileSync(f, 'utf8'))))
		.map(relPfad);

	it('durchsucht überhaupt Dateien und sieht die Ausnahme', () => {
		expect(dateien.length).toBeGreaterThan(300);
		for (const datei of AUSNAHMEN.keys())
			expect(mitTitle, 'Ausnahme ohne Fundstelle').toContain(datei);
	});

	it('kein Element trägt die Sprechblase des Browsers', () => {
		expect(
			mitTitle.filter((f) => !AUSNAHMEN.has(f)),
			'title an einem Element — data-tip nehmen (actions/tooltip.js), und was nur dort steht, für den Bildschirmleser daneben'
		).toEqual([]);
	});

	it('Selbstprobe: der Detektor fasst die Formen', () => {
		for (const form of [
			'<button class="x" title="Kopieren">',
			'<Button variant="secondary"\n\ttitle={text}\n>',
			'<a href={url} title="PDF">',
			'<button onclick={() => (a = b > 1)} title="x">',
			'<span title={voll}>gekürzt</span>',
			'<div\n\tclass="x"\n\ttitle="Hinweis"\n>',
			'<Button variant="ghost" {title}>'
		]) {
			expect(TITLE_AN_BEDIENELEMENT.test(form), form).toBe(true);
		}
		for (const form of [
			'<button data-tip="Kopieren">',
			'<span data-tip={voll}>gekürzt</span>',
			'<abbr title="x">',
			'<title>Bibliothek</title>',
			'<SuchZustand titel="Nichts gefunden" />'
		]) {
			expect(TITLE_AN_BEDIENELEMENT.test(form), form).toBe(false);
		}
	});
});
