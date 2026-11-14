import { describe, it, expect } from 'vitest';
import {
	klassenHinweisText,
	doppeltHinweisText,
	ausgabeHinweiseAus,
	ohneExemplar
} from './ausgabeHinweise.js';

// Die Sätze und die Liste der Hinweise zu einem ausgegebenen Schulbuch (OmniboxThekeHinweise).
describe('Hinweise zu einem ausgegebenen Schulbuch', () => {
	it('nennt beim fremden Buch Titel, Klasse und woran es gemessen ist', () => {
		expect(klassenHinweisText({ klasse: '08G1', mit_buechern: 19, kinder: 28 }, 'Mathe 7')).toBe(
			'„Mathe 7" hat sonst niemand in der 08G1. 19 von 28 Kindern haben ihre Schulbücher schon.'
		);
	});

	it('nennt beim doppelten Buch das Kind, ohne Vornamen „Das Kind"', () => {
		expect(doppeltHinweisText('Ida', 'Mathe 8')).toBe('Ida hat „Mathe 8" jetzt zweimal');
		expect(doppeltHinweisText('', 'Mathe 8')).toBe('Das Kind hat „Mathe 8" jetzt zweimal');
	});

	it('liest die Hinweise einer Antwort in der Reihenfolge der Theke', () => {
		const daten = {
			book: { id: 'ex-7', titel: 'Mathe 7' },
			auflagen_hinweis: {
				klasse: '08G1',
				auflage: '4. Aufl.',
				andere: [{ auflage: '3. Aufl.', kinder: 2 }]
			},
			klassen_hinweis: { klasse: '08G1', mit_buechern: 6, kinder: 8 },
			doppelt_hinweis: true
		};
		expect(ausgabeHinweiseAus(daten, 'Ida').map((h) => [h.exemplar, h.art])).toEqual([
			['ex-7', 'doppelt'],
			['ex-7', 'klasse'],
			['ex-7', 'auflagen']
		]);
		expect(ausgabeHinweiseAus({ book: { id: 'ex-7', titel: 'Mathe 7' } }, 'Ida')).toEqual([]);
	});

	it('nimmt nur die Zeilen des Exemplars weg, ohne Exemplar keine', () => {
		const liste = [
			{ exemplar: 'ex-7', art: /** @type {const} */ ('klasse'), text: 'a' },
			{ exemplar: 'ex-8', art: /** @type {const} */ ('doppelt'), text: 'b' }
		];
		expect(ohneExemplar(liste, 'ex-7')).toEqual([liste[1]]);
		expect(ohneExemplar(liste, undefined)).toEqual(liste);
		expect(ohneExemplar(liste, '')).toEqual(liste);
	});
});
