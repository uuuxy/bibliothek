import { describe, it, expect } from 'vitest';
import { filtereBedarf } from './bedarfFilter.js';

// Die Nummern sind die Beispielpaare der ISBN-Norm: 0-306-40615-2 ↔ 978-0-306-40615-7 und
// 3-16-148410-X ↔ 978-3-16-148410-0.
const zeilen = [
	{
		titel: 'Advanced Organic Chemistry',
		isbn: '9780306406157',
		verlag: 'Springer',
		signatur: 'Ch'
	},
	{
		titel: 'Elemente Chemie 1',
		isbn: '9783120000019',
		verlag: 'Klett',
		auflagen: [{ isbn: '9783120000019' }, { isbn: '9783161484100' }]
	},
	{ titel: 'Ohne ISBN', verlag: 'Eigenverlag' }
];
const titel = (/** @type {string} */ filter) => filtereBedarf(zeilen, filter).map((r) => r.titel);

describe('filtereBedarf', () => {
	it('ohne Filter: alle Zeilen', () => {
		expect(filtereBedarf(zeilen, '  ')).toBe(zeilen);
	});

	it('trifft Titel, Verlag und Signatur als Teilstring', () => {
		expect(titel('organic')).toEqual(['Advanced Organic Chemistry']);
		expect(titel('klett')).toEqual(['Elemente Chemie 1']);
		expect(titel('ch')).toEqual(['Advanced Organic Chemistry', 'Elemente Chemie 1']);
	});

	it('trifft die ISBN als Teilstring, auch die einer älteren Auflage', () => {
		expect(titel('97803064')).toEqual(['Advanced Organic Chemistry']);
		expect(titel('9783161484100')).toEqual(['Elemente Chemie 1']);
	});

	// Der Katalog führt die ISBN dreizehnstellig; auf dem Titelblatt steht die zehnstellige.
	it('trifft die ISBN in der anderen Länge und mit Bindestrichen', () => {
		expect(titel('0306406152')).toEqual(['Advanced Organic Chemistry']);
		expect(titel('0-306-40615-2')).toEqual(['Advanced Organic Chemistry']);
		expect(titel('978-0-306-40615-7')).toEqual(['Advanced Organic Chemistry']);
		expect(titel('3-16-148410-x')).toEqual(['Elemente Chemie 1']);
	});

	it('eine andere Nummer trifft nichts', () => {
		expect(titel('0306406160')).toEqual([]);
	});
});
