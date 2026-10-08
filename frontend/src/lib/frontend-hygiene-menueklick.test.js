import { describe, it, expect } from 'vitest';
import { readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import { srcRoot, repoFrontend, ohneKommentare } from './hygiene-quellen.js';

// Ein Browser-Test findet einen Menüpunkt über `menuepunkt` (e2e/helpers.js), nicht über die
// ganze Seite.
//
// `page.getByTitle('Abgänger')` trifft jedes Element, dessen title den Namen enthält: auch die
// Kachel eines Buchs, das so heißt. Solange kein solches Buch im Katalog steht, klickt der Test
// richtig; legt ein anderer Test eines an, bricht Playwright ab oder klickt die Kachel.
//
// Die Namen kommen aus menu.js, nicht aus einer Liste hier. Blind für: einen Namen aus einer
// Variablen und eine Suche über einen regulären Ausdruck.
const UEBER_DIE_SEITE = /getByTitle\(\s*(['"`])([^'"`]+)\1/g;

describe('Menüpunkte in Browser-Tests', () => {
	const menue = readFileSync(join(srcRoot, 'lib', 'menu.js'), 'utf8');
	const namen = [...menue.matchAll(/label:\s*'([^']+)'/g)].map((m) => m[1]);

	it('der Detektor kennt die Menüpunkte und erkennt die Form', () => {
		expect(namen).toContain('Leserdatei');
		expect(namen).toContain('Mein Portal');
		expect(namen.length).toBeGreaterThan(10);
		const treffer = [..."await page.getByTitle('Leserdatei').click();".matchAll(UEBER_DIE_SEITE)];
		expect(treffer.map((m) => m[2])).toEqual(['Leserdatei']);
		expect([..."menuepunkt(page, 'Leserdatei')".matchAll(UEBER_DIE_SEITE)]).toEqual([]);
	});

	it('kein Spec sucht einen Menüpunkt über die ganze Seite', () => {
		const e2e = join(repoFrontend, 'e2e');
		/** @type {string[]} */
		const betroffen = [];
		let mitHelfer = 0;
		for (const datei of readdirSync(e2e).filter((d) => d.endsWith('.spec.js'))) {
			const quelle = ohneKommentare(readFileSync(join(e2e, datei), 'utf8'));
			if (/\bmenuepunkt\(/.test(quelle)) mitHelfer++;
			for (const m of quelle.matchAll(UEBER_DIE_SEITE)) {
				if (namen.includes(m[2])) betroffen.push(`${datei}: getByTitle('${m[2]}')`);
			}
		}

		// Ohne diese Zusicherung wäre ein leerer Suchlauf ein still grüner Test.
		expect(
			mitHelfer,
			'kaum Specs mit menuepunkt gefunden — der Detektor misst nichts'
		).toBeGreaterThan(30);
		expect(
			betroffen,
			'Menüpunkt über die ganze Seite gesucht. Stattdessen: menuepunkt(page, name) aus ./helpers.js.'
		).toEqual([]);
	});
});
