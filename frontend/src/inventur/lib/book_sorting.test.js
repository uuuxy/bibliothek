import { describe, it, expect } from 'vitest';
import { sortBooksBySubjectAndTitle } from './book_sorting.js';

describe('sortBooksBySubjectAndTitle', () => {
	it('sorts by predefined subject order', () => {
		const a = { subject: 'deutsch', title: 'Buch A' };
		const b = { subject: 'mathe', title: 'Buch B' };
		expect(sortBooksBySubjectAndTitle(a, b)).toBeGreaterThan(0);
		expect(sortBooksBySubjectAndTitle(b, a)).toBeLessThan(0);
	});

	it('sorts by title when subjects are the same', () => {
		const a = { subject: 'mathe', title: 'Algebra' };
		const b = { subject: 'mathe', title: 'Geometrie' };
		expect(sortBooksBySubjectAndTitle(a, b)).toBeLessThan(0);
		expect(sortBooksBySubjectAndTitle(b, a)).toBeGreaterThan(0);
	});

	it('handles "mathematik" variation as "mathe"', () => {
		const a = { subject: 'mathematik', title: 'Algebra' };
		const b = { subject: 'deutsch', title: 'Grammatik' };
		expect(sortBooksBySubjectAndTitle(a, b)).toBeLessThan(0);
	});

	it('handles case-insensitivity and whitespace', () => {
		const a = { subject: ' Mathe ', title: 'Algebra' };
		const b = { subject: 'DEUTSCH', title: 'Grammatik' };
		expect(sortBooksBySubjectAndTitle(a, b)).toBeLessThan(0);
	});

	it('sorts unknown subjects to the end', () => {
		const a = { subject: 'sport', title: 'Basketball' };
		const b = { subject: 'physik', title: 'Mechanik' };
		expect(sortBooksBySubjectAndTitle(a, b)).toBeGreaterThan(0);
		expect(sortBooksBySubjectAndTitle(b, a)).toBeLessThan(0);
	});

	it('sorts unknown subjects by title', () => {
		const a = { subject: 'sport', title: 'Basketball' };
		const b = { subject: 'kunst', title: 'Malerei' };
		expect(sortBooksBySubjectAndTitle(a, b)).toBeLessThan(0);
	});

	it('handles missing subjects', () => {
		const a = { title: 'Zebra' };
		const b = { subject: 'mathe', title: 'Algebra' };
		expect(sortBooksBySubjectAndTitle(a, b)).toBeGreaterThan(0);

		const c = { subject: '', title: 'Affe' };
		expect(sortBooksBySubjectAndTitle(a, c)).toBeGreaterThan(0);
	});
});
