import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import {
	srcRoot,
	relPfad,
	ohneKommentare,
	sammleQuelldateien,
	sammleTestdateien
} from './hygiene-quellen.js';

// Der Omnibox-Store ist in Teile geschnitten (stores/omnibox<Teil>.svelte.js, docs/OFFEN.md 5.72),
// und nur der Store selbst lädt sie. Bauteile und Tests gehen über seine Fassade: Lüde ein Test
// einen Teil direkt, startete er dessen Zeitgeber an frontend-hygiene-thekenzeitgeber.test.js
// vorbei, das nur Importe aus omnibox.svelte.js kennt; lüde ein Bauteil ihn, gäbe es zwei Wege
// zum selben Zustand.
//
// Sieht nicht: einen Pfad aus einer Variablen oder einer Verkettung, `vi.mock` eines Teils und
// einen Teil, dessen Dateiname nicht mit `omnibox` und einem Großbuchstaben beginnt.
const FASSADE = 'src/lib/stores/omnibox.svelte.js';
const TEIL_IMPORT = /(?:from\s*|import\s*\(\s*)['"`][^'"`]*\/omnibox[A-Z]\w*\.svelte\.js['"`]/;

describe('Teile des Omnibox-Stores', () => {
	const dateien = [...sammleQuelldateien(srcRoot), ...sammleTestdateien(srcRoot)];

	it('lädt nur die Fassade', () => {
		const fremd = dateien
			.filter((d) => TEIL_IMPORT.test(ohneKommentare(readFileSync(d, 'utf8'))))
			.map(relPfad)
			.filter((p) => p !== FASSADE);
		expect(
			fremd,
			'Diese Dateien laden einen Teil des Omnibox-Stores direkt. Über omniboxStore gehen ' +
				'(stores/omnibox.svelte.js); was fehlt, bekommt die Fassade.'
		).toEqual([]);
	});

	it('misst etwas: die Fassade lädt mindestens einen Teil', () => {
		const fassade = dateien.find((d) => relPfad(d) === FASSADE);
		expect(fassade, `${FASSADE} fehlt — der Detektor misst nichts`).toBeDefined();
		expect(TEIL_IMPORT.test(ohneKommentare(readFileSync(String(fassade), 'utf8')))).toBe(true);
	});

	// Die Proben entstehen aus Teilen: Stünde ein Import wörtlich hier, fänden ihn dieses Gate und
	// frontend-hygiene-thekenzeitgeber.test.js in dieser Datei.
	it('Selbstprobe: der Detektor fasst die Importformen und lässt die Fassade aus', () => {
		/** @param {string} pfad */
		const statisch = (pfad) => `import { x } from '${pfad}';`;
		for (const form of [
			statisch('./omniboxRueckmeldung.svelte.js'),
			statisch('../stores/omniboxSuche.svelte.js').replaceAll("'", '"'),
			`const teil = await import('${'./stores/omniboxReihe.svelte.js'}');`
		]) {
			expect(TEIL_IMPORT.test(form), form).toBe(true);
		}
		expect(TEIL_IMPORT.test(statisch('./stores/omnibox.svelte.js'))).toBe(false);
	});
});
