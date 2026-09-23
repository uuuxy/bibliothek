import { describe, it, expect } from 'vitest';
import { menueEintraege, loeschFolgen } from './schlagwortPflege.js';

const wort = { id: 'a', wort: 'Fantasy', titel: 12, verweise: ['Tierfantasy'], ist_filter: false };
const verweis = {
	id: 'b',
	wort: 'Tierfantasy',
	titel: 0,
	verweis_auf_id: 'a',
	verweis_auf: 'Fantasy',
	verweise: [],
	ist_filter: false
};

describe('Schlagwort-Pflege: Anzeige-Regeln', () => {
	it('bietet am Verweis nur Umbenennen und Löschen an', () => {
		expect(menueEintraege(verweis).map((e) => e.id)).toEqual(['umbenennen', 'loeschen']);
		expect(menueEintraege(wort).map((e) => e.id)).toEqual([
			'umbenennen',
			'zusammenfuehren',
			'verweis',
			'loeschen'
		]);
	});

	it('nennt in der Rückfrage die Zahl der Titel und der Verweise', () => {
		expect(loeschFolgen(wort)).toBe(
			'12 Titel verlieren das Schlagwort, der Verweis darauf fällt mit. Das lässt sich nicht rückgängig machen.'
		);
		expect(loeschFolgen({ ...wort, verweise: ['X', 'Y'] })).toMatch(
			/, 2 Verweise darauf fallen mit\. /
		);
		expect(loeschFolgen({ ...wort, verweise: [] })).toMatch(
			/^12 Titel verlieren das Schlagwort\. /
		);
		expect(loeschFolgen(verweis)).toMatch(/nicht mehr bei „Fantasy“/);
	});
});
