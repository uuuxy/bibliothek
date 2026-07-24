import { describe, it, expect } from 'vitest';
import { sortBooksBySubjectAndTitle } from './book_sorting.js';

describe('sortBooksBySubjectAndTitle', () => {
	it('sorts by predefined subject order', () => {
		const books = [
			{ subject: 'Deutsch', title: 'Buch B' },
			{ subject: 'Mathe', title: 'Buch A' }
		];

		const sorted = [...books].sort(sortBooksBySubjectAndTitle);

		expect(sorted[0].subject).toBe('Mathe');
		expect(sorted[1].subject).toBe('Deutsch');
	});

	it('sorts by title if subjects are identical', () => {
		const books = [
			{ subject: 'Mathe', title: 'Buch Z' },
			{ subject: 'Mathe', title: 'Buch A' }
		];

		const sorted = [...books].sort(sortBooksBySubjectAndTitle);

		expect(sorted[0].title).toBe('Buch A');
		expect(sorted[1].title).toBe('Buch Z');
	});

	it('handles different casing and trailing spaces in subjects', () => {
		const books = [
			{ subject: ' DEUTSCH ', title: 'Buch B' },
			{ subject: 'mAtHe', title: 'Buch A' }
		];

		const sorted = [...books].sort(sortBooksBySubjectAndTitle);

		expect(sorted[0].subject).toBe('mAtHe');
		expect(sorted[1].subject).toBe(' DEUTSCH ');
	});

	it('normalizes "mathematik" to "mathe"', () => {
		const books = [
			{ subject: 'Englisch', title: 'Buch B' },
			{ subject: 'Mathematik', title: 'Buch A' }
		];

		const sorted = [...books].sort(sortBooksBySubjectAndTitle);

		expect(sorted[0].subject).toBe('Mathematik');
		expect(sorted[1].subject).toBe('Englisch');
	});

	it('puts unknown subjects at the end, sorted by title', () => {
		const books = [
			{ subject: 'Sport', title: 'Z' },
			{ subject: 'Informatik', title: 'A' },
			{ subject: 'Mathe', title: 'M' }
		];

		const sorted = [...books].sort(sortBooksBySubjectAndTitle);

		expect(sorted[0].title).toBe('M');
		expect(sorted[1].title).toBe('A'); // Informatik
		expect(sorted[2].title).toBe('Z'); // Sport
	});

	it('handles missing or undefined subjects gracefully', () => {
		const books = [
			{ title: 'Z Ohne Fach' },
			{ subject: 'Mathe', title: 'A Mathe' }
		];

		const sorted = [...books].sort(sortBooksBySubjectAndTitle);

		expect(sorted[0].title).toBe('A Mathe');
		expect(sorted[1].title).toBe('Z Ohne Fach');
	});

	it('sorts correctly when both subjects are unknown or missing', () => {
		const a = { subject: undefined, title: 'B' };
		const b = { subject: '', title: 'A' };

		expect(sortBooksBySubjectAndTitle(a, b)).toBe(1); // 'B' > 'A'
		expect(sortBooksBySubjectAndTitle(b, a)).toBe(-1); // 'A' < 'B'
	});
});
