import { describe, it, expect } from 'vitest';
import pruefung from './suchnorm.faelle.json';
import { suchnorm, passtZurSuche } from './suchnorm.js';

// Die Fälle sind mit der SQL-Funktion suchnorm erzeugt; suchnorm_faelle_test.go prüft Go an
// denselben. Rechnete der Browser anders, fände dieselbe Eingabe hier andere Menschen als an der
// Theke.
describe('suchnorm', () => {
	it('rechnet jedes Zeichen und jeden Namen wie die Datenbank', () => {
		expect(pruefung.zeichen.length).toBeGreaterThan(600);
		const abweichend = [...pruefung.zeichen, ...pruefung.namen].filter(
			([eingabe, erwartet]) => suchnorm(eingabe) !== erwartet
		);
		expect(abweichend).toEqual([]);
	});
});

describe('passtZurSuche', () => {
	it.each([
		['ozturk', ['Mehmet', 'Öztürk']],
		['alkahil', ['Ahmad', 'Al-Kahil']],
		['al kahil', ['Yusuf', 'Alkahil']],
		['al-kahil', ['Samir', 'Al Kahil']],
		['annalena', ['Anna-Lena', 'Brandt']],
		['anna lena', ['Annalena', 'Kurz']],
		['obrien', ['Sean', 'O’Brien']],
		["o'brien", ['Sean', 'OBrien']],
		['müller anna', ['Anna', 'Mueller']],
		['07h', ['Lena', 'Bauer', '07H2']],
		['', ['egal']]
	])('„%s“ trifft %j', (suche, felder) => {
		expect(passtZurSuche(suche, ...felder)).toBe(true);
	});

	it.each([
		['ozturk anna', ['Mehmet', 'Öztürk']],
		['kahil', ['Anna', 'Müller']]
	])('„%s“ trifft %j nicht', (suche, felder) => {
		expect(passtZurSuche(suche, ...felder)).toBe(false);
	});
});
