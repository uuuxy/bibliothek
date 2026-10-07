import { describe, it, expect } from 'vitest';
import { resthoeheAus } from './resthoehe.js';

// Die Höhe bis zum unteren Rand des Scrollbereichs. Am Bildschirm misst
// e2e/bestellung-erreichbar.spec.js; hier steht die Rechnung.
describe('resthoeheAus', () => {
	it('reicht von der Oberkante bis zur Unterkante des Bereichs', () => {
		expect(resthoeheAus({ bereichUnten: 928, gescrollt: 0, oben: 321 })).toBe(607);
	});

	it('rechnet im Fluss mit der Lage im Bereich, gleich wie weit gescrollt ist', () => {
		const oben = resthoeheAus({ bereichUnten: 928, gescrollt: 0, oben: 321 });
		const gescrollt = resthoeheAus({ bereichUnten: 928, gescrollt: 100, oben: 221 });
		expect(gescrollt, 'das Element wüchse mit dem Scrollen').toBe(oben);
	});

	it('lässt ein haftendes Element wachsen, bis es klebt', () => {
		const wahl = { haftet: true };
		expect(resthoeheAus({ bereichUnten: 928, gescrollt: 0, oben: 169 }, wahl)).toBe(759);
		expect(resthoeheAus({ bereichUnten: 928, gescrollt: 100, oben: 119 }, wahl)).toBe(809);
	});

	it('rundet ab und unterschreitet die Mindesthöhe nicht', () => {
		expect(resthoeheAus({ bereichUnten: 928.6, gescrollt: 0, oben: 321.2 })).toBe(607);
		expect(resthoeheAus({ bereichUnten: 400, gescrollt: 0, oben: 321 })).toBe(240);
		expect(resthoeheAus({ bereichUnten: 400, gescrollt: 0, oben: 321 }, { mindestens: 0 })).toBe(
			79
		);
	});
});
