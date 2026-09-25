import { describe, it, expect } from 'vitest';
import { buecherSuchen, buecherJeBuch } from './startseiten_api.js';

const buch = (/** @type {any} */ felder) => ({ id: Math.random().toString(), ...felder });

// Filter, Sortierung und Signatur-Suche der Buch-Suche (02.09.2026): Der Reiter hieß
// „Suche & Filter", hatte aber keinen Filter; die Signatur (Regaladresse) fand die
// Suche nicht, obwohl der Payload sie trug; Titel ohne Exemplare waren von komplett
// verliehenen nicht zu unterscheiden.
const bestand = [
	buch({
		title: 'Faust',
		author: 'Goethe',
		subject: 'Deutsch',
		signatur: 'Deu Goe',
		verfuegbar: 2,
		gesamt: 3,
		jahrgangVon: 9,
		jahrgangBis: 10,
		track: 'Gymnasium'
	}),
	buch({
		title: 'Analysis',
		subject: 'Mathematik',
		signatur: 'Mat Ana',
		verfuegbar: 0,
		gesamt: 5,
		gradeLevel: 11
	}),
	buch({ title: 'Ohne Bestand', subject: 'Deutsch', signatur: '', verfuegbar: 0, gesamt: 0 }),
	buch({ title: 'Hörbuch', subject: 'Englisch', medientyp: 'CD', verfuegbar: 1, gesamt: 1 })
];

describe('buecherSuchen: Signatur', () => {
	it('findet ein Buch über seine Signatur (Regaladresse)', () => {
		expect(buecherSuchen(bestand, 'goe').map((b) => b.title)).toEqual(['Faust']);
	});
});

// Schlagworte (docs/OFFEN.md 4.20): Welche Wörter einen Titel finden, sagt der Server je
// Titel — seine Schlagworte und die Verweise darauf (`suchwoerter`, siehe
// repository.SuchwoerterDerTitel). Die Suche prüft sie wie Titel und Autor.
describe('buecherSuchen: Schlagworte und Verweise', () => {
	const katalog = [
		buch({
			title: 'Drachenreiter',
			author: 'Cornelia Funke',
			suchwoerter: ['Drachen', 'Fantasy', 'Tierfantasy']
		}),
		buch({ title: 'Krabat', author: 'Otfried Preußler' })
	];
	const titel = (/** @type {string} */ q) => buecherSuchen(katalog, q).map((b) => b.title);

	it('findet über ein Schlagwort, auch angefangen', () => {
		expect(titel('fanta')).toEqual(['Drachenreiter']);
	});
	it('findet über einen Verweis auf ein Schlagwort', () => {
		expect(titel('Tierfantasy')).toEqual(['Drachenreiter']);
	});
	it('jeder Begriff darf ein anderes Feld treffen', () => {
		expect(titel('funke fantasy')).toEqual(['Drachenreiter']);
	});
	it('ein Titel ohne Schlagworte wird weiter über seine Felder gefunden', () => {
		expect(titel('krabat')).toEqual(['Krabat']);
	});
});

// Ein Buch in mehreren Auflagen ist eine Kachel (docs/OFFEN.md 4.18, Stufe 6). werkId und
// werkRang kommen vom Server (1 = die neueste Auflage); gezählt wird über alle Auflagen der
// ganzen Liste, oben steht die getroffene.
describe('buecherJeBuch', () => {
	const neu = {
		id: 'neu',
		title: 'Mathe 7',
		isbn: '978-3-12-000002',
		auflage: '4. Aufl.',
		erscheinungsjahr: 2023,
		werkId: 'w',
		werkRang: 1,
		gesamt: 4,
		verfuegbar: 1,
		imZulauf: 30
	};
	const alt = {
		id: 'alt',
		title: 'Mathe 7',
		isbn: '978-3-12-000001',
		auflage: '3. Aufl.',
		erscheinungsjahr: 2019,
		werkId: 'w',
		werkRang: 2,
		gesamt: 42,
		verfuegbar: 40,
		imZulauf: 0
	};
	const faust = { id: 'faust', title: 'Faust', gesamt: 3, verfuegbar: 2 };
	const katalog = [faust, alt, neu];
	const summe = {
		gesamt: 46,
		verfuegbar: 41,
		imZulauf: 30,
		auflagen: [
			{ id: 'neu', auflage: '4. Aufl.', erscheinungsjahr: 2023, gesamt_bestand: 4 },
			{ id: 'alt', auflage: '3. Aufl.', erscheinungsjahr: 2019, gesamt_bestand: 42 }
		]
	};

	it('zeigt das Buch einmal: die neueste Auflage oben, die Summe aller, die neueste zuerst', () => {
		expect(buecherJeBuch(katalog, katalog)).toEqual([faust, { ...neu, buch: summe }]);
	});

	it('trifft die Suche nur die alte Auflage, steht sie oben — gezählt wird trotzdem das ganze Buch', () => {
		const karten = buecherJeBuch(katalog, buecherSuchen(katalog, '978-3-12-000001'));
		expect(karten).toEqual([{ ...alt, buch: summe }]);
	});

	it('die Kachel steht dort, wo die erste getroffene Auflage stand', () => {
		expect(buecherJeBuch(katalog, [alt, faust, neu]).map((b) => b.id)).toEqual(['neu', 'faust']);
	});

	it('eine Auflage allein im Katalog ist eine gewöhnliche Kachel', () => {
		expect(buecherJeBuch([faust, neu], [faust, neu])).toEqual([faust, neu]);
	});
});
