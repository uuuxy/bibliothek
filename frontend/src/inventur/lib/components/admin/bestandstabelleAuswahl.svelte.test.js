import { describe, it, expect } from 'vitest';
import { erzeugeAuswahl } from './bestandstabelleAuswahl.svelte.js';

/** Kennungen wie in der Datenbank: UUIDs, 36 Zeichen. @param {number} n */
const zeilen = (n) => Array.from({ length: n }, () => ({ id: crypto.randomUUID() }));

// Die Markierung bleibt auf die sichtbaren Zeilen beschränkt. Der Abgleich suchte jede markierte
// Kennung mit includes in der Liste der sichtbaren: Bei 11.000 markierten und sichtbaren Titeln
// kostete jeder Tastendruck in der Suche 0,6 s, bei 20.000 fast 2 s.
describe('Markierung der Bestandstabelle', () => {
	it('behält beim Abgleich nur die Kennungen sichtbarer Zeilen', () => {
		const fuenf = zeilen(5);
		const auswahl = erzeugeAuswahl();
		auswahl.alleUmschalten(fuenf);
		auswahl.angleichen(fuenf.slice(0, 3));
		expect([...auswahl.ids].sort()).toEqual(
			fuenf
				.slice(0, 3)
				.map((z) => z.id)
				.sort()
		);
	});

	it('gleicht 20.000 markierte und sichtbare Titel ohne spürbare Wartezeit ab', () => {
		/** Der schnellste von drei Abgleichen: Ein Stocken des Rechners trifft nicht alle. @param {number} n */
		const dauer = (n) => {
			const laeufe = [0, 0, 0].map(() => {
				const alle = zeilen(n);
				const auswahl = erzeugeAuswahl();
				auswahl.alleUmschalten(alle);
				const start = performance.now();
				auswahl.angleichen(alle);
				expect(auswahl.anzahl).toBe(n);
				return performance.now() - start;
			});
			return Math.min(...laeufe);
		};
		// Gemessen wird das Wachstum, nicht die Uhr: Viermal so viele Titel kosten mit der
		// Nachschlagetabelle rund das Vierfache, mit includes das Sechzehnfache.
		expect(dauer(20000) / dauer(5000)).toBeLessThan(9);
	});
});
