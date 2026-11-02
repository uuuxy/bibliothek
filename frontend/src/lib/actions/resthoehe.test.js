import { describe, it, expect, vi, afterEach } from 'vitest';
import { resthoehe, resthoeheAus } from './resthoehe.js';

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

// In einem Scrollbereich beobachtet die Aktion die Vorfahren bis zu ihm, nicht weiter.
describe('resthoehe in einem Scrollbereich', () => {
	afterEach(() => {
		vi.unstubAllGlobals();
		document.body.innerHTML = '';
	});

	it('hängt sich ein und setzt die Höhe', () => {
		const beobachtet = [];
		vi.stubGlobal(
			'ResizeObserver',
			class {
				/** @param {Element} el */
				observe(el) {
					beobachtet.push(el);
				}
				disconnect() {}
			}
		);
		const bereich = document.createElement('div');
		bereich.style.overflowY = 'auto';
		const huelle = document.createElement('div');
		const liste = document.createElement('div');
		huelle.append(liste);
		bereich.append(huelle);
		document.body.append(bereich);

		const aktion = resthoehe(liste);

		expect(liste.style.getPropertyValue('--resthoehe')).toMatch(/px$/);
		expect(beobachtet).toContain(huelle);
		aktion.destroy();
	});
});
