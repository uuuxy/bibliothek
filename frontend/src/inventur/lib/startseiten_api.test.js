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

describe('buecherSuchen', () => {
	const katalog = [
		buch({ title: 'Mathematik 1', subject: 'Mathematik', gradeLevel: 5 }),
		buch({ title: 'Politik und Wirtschaft', subject: 'PoWi', gradeLevel: 10 }),
		buch({ title: 'Englisch G21', author: 'Schwarz', isbn: '978-3-06-031306-8' }),
		buch({ title: 'Biologie heute', istLernmittel: true }),
		buch({ title: 'Chemie', jahrgangVon: 8, jahrgangBis: 10 })
	];
	const finde = (/** @type {string} */ q) => buecherSuchen(katalog, q).map((b) => b.title);

	it('gibt bei leerer Suche alle Bücher zurück', () => {
		expect(buecherSuchen(katalog, '')).toHaveLength(katalog.length);
		// @ts-expect-error null statt einer Liste: Die Suche liefert dann eine leere Liste.
		expect(buecherSuchen(null, '')).toEqual([]);
	});

	it('findet Bücher über Autor, Titel, Fach', () => {
		expect(finde('schwarz')).toEqual(['Englisch G21']);
		expect(finde('mathe')).toEqual(['Mathematik 1']);
	});

	it('nutzt Synonyme für die Suche', () => {
		// 'powi' sucht nach 'politik' -> findet "Politik und Wirtschaft"
		expect(finde('powi')).toEqual(['Politik und Wirtschaft']);
	});

	it('findet über Jahrgangsstufe (gradeLevel) oder Spanne (jahrgangVon-Bis)', () => {
		expect(finde('10')).toEqual(['Politik und Wirtschaft', 'Chemie']);
		expect(finde('5')).toEqual(['Mathematik 1']);
	});

	it('ignoriert Füllwörter wie Klasse oder Jg bei Zahlensuche', () => {
		expect(finde('10. klasse')).toEqual(['Politik und Wirtschaft', 'Chemie']);
		expect(finde('jg. 8')).toEqual(['Chemie']);
	});

	// Die ISBN im Katalog enthält 7, 8, 9 und „06": Ohne die Regel träfe jede dieser Suchen
	// „Englisch G21", ein Buch ohne Jahrgang.
	it.each([
		['klasse 7', []],
		['9', ['Chemie']],
		['06', []]
	])('liest „%s" als Jahrgang, nicht als Stück einer ISBN', (suche, erwartet) => {
		expect(finde(suche)).toEqual(erwartet);
	});

	it('findet eine Ziffer weiter im Titel', () => {
		expect(finde('1')).toEqual(['Mathematik 1', 'Englisch G21']);
	});

	it('findet über ISBN (mit und ohne Striche, alte 10-stellige)', () => {
		expect(finde('9783060313068')).toEqual(['Englisch G21']);
		expect(finde('978-3-06-031306-8')).toEqual(['Englisch G21']);
		expect(finde('306031306X')).toEqual(['Englisch G21']);
	});

	it('findet über ein Stück der ISBN ab drei Ziffern', () => {
		expect(finde('978')).toEqual(['Englisch G21']);
		expect(finde('0313')).toEqual(['Englisch G21']);
	});

	it('findet über Eigenschaft Lernmittel', () => {
		expect(finde('lernmittel')).toEqual(['Biologie heute']);
	});
});

// Die Spalten-Vorgabe 5 bis 10 steht an jedem Titel, dessen Jahrgang niemand eingetragen hat.
describe('buecherSuchen: Jahrgang ohne Angabe', () => {
	const katalog = [
		buch({ title: 'Atlas', jahrgangVon: 5, jahrgangBis: 10 }),
		buch({ title: 'Erdkunde', jahrgangVon: 5, jahrgangBis: 10, gradeLevel: 7 }),
		buch({ title: 'Geschichte', jahrgangVon: 5, jahrgangBis: 9 })
	];
	const titel = (/** @type {string} */ q) => buecherSuchen(katalog, q).map((b) => b.title);

	it('die Vorgabe 5 bis 10 trifft keinen Jahrgang, die Klasse am Titel schon', () => {
		expect(titel('klasse 7')).toEqual(['Erdkunde', 'Geschichte']);
		expect(titel('10')).toEqual([]);
	});
});

describe('buecherSuchen: Signatur', () => {
	it('findet ein Buch über seine Signatur (Regaladresse)', () => {
		expect(buecherSuchen(bestand, 'goe').map((b) => b.title)).toEqual(['Faust']);
	});
});

// Je Feld ein Buch, das nur dieses Feld trägt: Ein Treffer über ein Nachbarfeld belegte das
// Feld nicht, und ein fehlendes Feld darf die Suche nicht abbrechen.
describe('buecherSuchen: jedes Feld für sich', () => {
	const katalog = [
		buch({ title: 'Faust' }),
		buch({ author: 'Goethe' }),
		buch({ subject: 'Deutsch' }),
		buch({ signatur: 'Kla 12' }),
		buch({ isbn: '978-3-06-031306-8' })
	];

	it.each([
		['faust', 0],
		['goethe', 1],
		['deutsch', 2],
		['kla', 3],
		['0313', 4]
	])('„%s“ trifft genau das eine Buch', (suche, nr) => {
		expect(buecherSuchen(katalog, suche)).toEqual([katalog[nr]]);
	});
});

// Was ein Begriff ist (Jahrgang, ISBN), entscheidet sich je Begriff, nicht am ersten.
describe('buecherSuchen: mehrere Begriffe', () => {
	const katalog = [
		buch({ title: 'Mathematik A', isbn: '978-3-06-031306-8', gradeLevel: 5 }),
		buch({ title: 'Mathematik B', gradeLevel: 7 })
	];
	const titel = (/** @type {string} */ q) => buecherSuchen(katalog, q).map((b) => b.title);

	it('ein Jahrgang hinter einem Wort bleibt ein Jahrgang', () => {
		// Als Stück einer ISBN träfe die 7 auch „Mathematik A“ (978…).
		expect(titel('mathe 7')).toEqual(['Mathematik B']);
	});

	it('eine ISBN hinter einem Wort findet auch die andere Schreibweise', () => {
		expect(titel('mathe 306031306X')).toEqual(['Mathematik A']);
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
