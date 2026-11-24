import { describe, it, expect } from 'vitest';
import pruefung from './suchnorm.faelle.json';
import { suchnorm, kompakt, passtZurSuche } from './suchnorm.js';

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

// Dieselben Fälle prüft leser_suche_pg_test.go an der Datenbank: Welche Zeichen ein Name verliert,
// entscheidet an der Theke die SQL-Form, im Mahnwesen diese.
describe('kompakt', () => {
	it('lässt dieselben Trenner weg wie die Datenbank und keine anderen', () => {
		expect(pruefung.kompakt.length).toBeGreaterThan(40);
		const abweichend = pruefung.kompakt.filter(
			([eingabe, erwartet]) => kompakt(eingabe) !== erwartet
		);
		expect(abweichend).toEqual([]);
	});

	// Vollprobe über den Grundbereich: Jedes Zeichen, das suchnorm stehen lässt und der Browser aus
	// einem Namen weglässt, steht in den Fällen. Sonst wüchse die Liste hier, ohne dass die
	// Datenbank es merkte.
	it('lässt kein Zeichen weg, das die Fälle nicht als Trenner führen', () => {
		const trenner = new Set(
			pruefung.kompakt
				.filter(([e, w]) => e.length === 8 && /^Al.Kahil$/su.test(e) && w === 'alkahil')
				.map(([e]) => e[2])
		);
		/** @type {string[]} */
		const weg = [];
		for (let cp = 1; cp <= 0xfffd; cp++) {
			if (cp >= 0xd800 && cp <= 0xdfff) continue;
			const zeichen = String.fromCharCode(cp);
			const probe = `x${zeichen}z`;
			if (suchnorm(probe) === `x${zeichen.toLowerCase()}z` && kompakt(probe) === 'xz')
				weg.push(zeichen);
		}
		expect(weg.filter((z) => !trenner.has(z))).toEqual([]);
		expect(weg.length).toBe(trenner.size);
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
