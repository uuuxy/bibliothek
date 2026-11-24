import { describe, it, expect } from 'vitest';
import { gehoertZu, offenerBedarf, bedarfZu } from './bedarfFilter.js';

// Ein Buch mit zwei Auflagen: Die Zeile zeigt die neueste, die Aufschlüsselung nennt beide.
const buch = {
	id: 'neu',
	titel: 'Elemente Chemie 1',
	isbn: '9783120000019',
	fehlen: 4,
	auflagen: [
		{ id: 'neu', isbn: '9783120000019' },
		{ id: 'alt', isbn: '978-3-16-148410-0' }
	]
};
const einzeln = { id: 'physik', titel: 'Physik 8', isbn: '', fehlen: 2 };

describe('gehoertZu', () => {
	it('erkennt die Position an der Kennung der neuesten Auflage', () => {
		expect(gehoertZu(buch, { id: 'neu' })).toBe(true);
	});

	// Wer die alte Auflage über die Suche bestellt, bestellt dasselbe Buch.
	it('erkennt die Position an der Kennung einer älteren Auflage', () => {
		expect(gehoertZu(buch, { id: 'alt' })).toBe(true);
	});

	it('erkennt die Position an der ISBN, auch ohne Bindestriche', () => {
		expect(gehoertZu(buch, { id: 'anderer-titel', isbn: '9783161484100' })).toBe(true);
	});

	it('nimmt eine leere ISBN nicht als Treffer', () => {
		expect(gehoertZu(einzeln, { id: 'anderer-titel', isbn: '' })).toBe(false);
		expect(gehoertZu({ ...einzeln, auflagen: [] }, { id: 'x' })).toBe(false);
	});
});

describe('offenerBedarf', () => {
	it('lässt weg, was schon in der Bestellung steht', () => {
		expect(offenerBedarf([buch, einzeln], [{ id: 'alt', isbn: '' }])).toEqual([einzeln]);
	});

	it('gibt mit leerer Bestellung alle Zeilen zurück', () => {
		expect(offenerBedarf([buch, einzeln], [])).toEqual([buch, einzeln]);
	});
});

describe('bedarfZu', () => {
	it('findet die Zeile zu einer Position', () => {
		expect(bedarfZu([einzeln, buch], { id: 'alt' })).toBe(buch);
	});

	it('gibt ohne passende Zeile null zurück', () => {
		expect(bedarfZu([einzeln, buch], { id: 'manga', isbn: '9783551716996' })).toBeNull();
	});
});
