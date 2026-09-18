import { describe, it, expect } from 'vitest';
import { isbnFormen, normalisiereIsbn } from './isbnFormen.js';

// Die Zahlen stammen aus echten Schulbüchern, nicht aus einem Generator: Der Vergleich
// ISBN-10 ↔ ISBN-13 ist eine Rechnung mit Prüfziffer, und eine selbst ausgedachte ISBN
// würde den Test grün halten, ohne dass die Rechnung stimmt.
describe('normalisiereIsbn', () => {
	it('lässt nur bedeutungstragende Zeichen stehen', () => {
		expect(normalisiereIsbn('978-3-06-013076-4')).toBe('9783060130764');
		expect(normalisiereIsbn('978 3 464 68012 9')).toBe('9783464680129');
		expect(normalisiereIsbn('3-16-148410-x')).toBe('316148410X');
		expect(normalisiereIsbn(null)).toBe('');
	});
});

describe('isbnFormen', () => {
	it('rechnet ISBN-13 (978) auf die zehnstellige Form zurück', () => {
		// 978-0-306-40615-7 und 0-306-40615-2 sind dasselbe Buch — das veröffentlichte
		// Beispielpaar der ISBN-Norm.
		expect(isbnFormen('9780306406157')).toEqual(['9780306406157', '0306406152']);
	});

	it('rechnet ISBN-10 auf die dreizehnstellige Form vor', () => {
		expect(isbnFormen('0306406152')).toEqual(['0306406152', '9780306406157']);
	});

	it('findet dieselbe ISBN über die Schreibweise hinweg', () => {
		// Gescannt wird die EAN vom Buchrücken, im Bestand steht die Alt-ISBN.
		expect(isbnFormen('978-0-306-40615-7')).toContain('0306406152');
		expect(isbnFormen('0-306-40615-2')).toContain('9780306406157');
	});

	it('trägt das Prüfzeichen X, wo es hingehört', () => {
		// 3-16-148410-X ↔ 978-3-16-148410-0, das zweite Beispielpaar der Norm.
		expect(isbnFormen('9783161484100')).toEqual(['9783161484100', '316148410X']);
		expect(isbnFormen('3-16-148410-x')).toEqual(['316148410X', '9783161484100']);
	});

	it('lässt 979er stehen — dazu gibt es keine ISBN-10', () => {
		expect(isbnFormen('9791234567896')).toEqual(['9791234567896']);
	});

	it('meldet leere Liste, wo keine ISBN ist — dann bleibt es bei der Textsuche', () => {
		expect(isbnFormen('mathematik')).toEqual([]);
		expect(isbnFormen('7')).toEqual([]);
		expect(isbnFormen('B97601826457')).toEqual([]); // Ausweis-Fremdnummer, keine ISBN
		expect(isbnFormen('')).toEqual([]);
	});
});
