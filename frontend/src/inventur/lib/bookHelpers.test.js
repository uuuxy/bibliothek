import { describe, it, expect, vi } from 'vitest';
import { getSubjectGradient, getSpineGradient, formatDate } from './bookHelpers.js';

describe('bookHelpers', () => {
	describe('getSubjectGradient', () => {
		it('returns correct gradient for Math variations', () => {
			const expected =
				'bg-linear-to-br from-blue-600 via-indigo-600 to-blue-700 border-blue-500/30';
			expect(getSubjectGradient('Math')).toBe(expected);
			expect(getSubjectGradient('Mathematik')).toBe(expected);
			expect(getSubjectGradient(' MATH ')).toBe(expected); // tests trim and case-insensitivity
		});

		it('returns correct gradient for German variations', () => {
			const expected = 'bg-linear-to-br from-red-600 via-rose-600 to-red-700 border-red-500/30';
			expect(getSubjectGradient('Deu')).toBe(expected);
			expect(getSubjectGradient('Deutsch')).toBe(expected);
		});

		it('returns correct gradient for foreign languages', () => {
			const expected =
				'bg-linear-to-br from-violet-600 via-purple-600 to-violet-700 border-purple-500/30';
			expect(getSubjectGradient('Englisch')).toBe(expected);
			expect(getSubjectGradient('Französisch')).toBe(expected);
		});

		it('returns correct gradient for Biology/Nature variations', () => {
			const expected =
				'bg-linear-to-br from-teal-600 via-emerald-600 to-teal-700 border-teal-500/30';
			expect(getSubjectGradient('Bio')).toBe(expected);
			expect(getSubjectGradient('Biologie')).toBe(expected);
		});

		it('returns correct gradient for History/Society variations', () => {
			const expected =
				'bg-linear-to-br from-amber-600 via-orange-600 to-amber-700 border-amber-500/30';
			expect(getSubjectGradient('Geschichte')).toBe(expected);
			expect(getSubjectGradient('Ges')).toBe(expected);
		});

		it('returns correct gradient for Music/Art variations', () => {
			const expected =
				'bg-linear-to-br from-pink-600 via-fuchsia-600 to-pink-700 border-pink-500/30';
			expect(getSubjectGradient('Musik')).toBe(expected);
			expect(getSubjectGradient('Kunst')).toBe(expected);
		});

		it('returns correct gradient for Computer Science variations', () => {
			const expected =
				'bg-linear-to-br from-slate-600 via-slate-700 to-slate-800 border-emerald-500/30';
			expect(getSubjectGradient('Informatik')).toBe(expected);
			expect(getSubjectGradient('Inf')).toBe(expected);
		});

		it('returns default gradient for unknown subjects', () => {
			expect(getSubjectGradient('Sport')).toBe(
				'bg-linear-to-br from-slate-500 via-slate-600 to-slate-700 border-slate-400/30'
			);
		});

		it('returns default gradient for empty or null inputs', () => {
			const expected =
				'bg-linear-to-br from-slate-500 via-slate-600 to-slate-700 border-slate-400/30';
			expect(getSubjectGradient(null)).toBe(expected);
			expect(getSubjectGradient(undefined)).toBe(expected);
			expect(getSubjectGradient('')).toBe(expected);
		});
	});

	describe('getSpineGradient', () => {
		it('returns correct spine gradient for Math variations', () => {
			expect(getSpineGradient('Mathematik')).toBe('from-blue-300 to-indigo-400');
		});

		it('returns correct spine gradient for German variations', () => {
			expect(getSpineGradient('Deutsch')).toBe('from-red-300 to-rose-400');
		});

		it('returns correct spine gradient for foreign languages', () => {
			const expected = 'from-violet-300 to-fuchsia-400';
			expect(getSpineGradient('Englisch')).toBe(expected);
			expect(getSpineGradient('Französisch')).toBe(expected);
		});

		it('returns correct spine gradient for Biology/Nature variations', () => {
			expect(getSpineGradient('Biologie')).toBe('from-teal-300 to-emerald-400');
		});

		it('returns correct spine gradient for History/Society variations', () => {
			expect(getSpineGradient('Geschichte')).toBe('from-amber-300 to-orange-400');
		});

		it('returns correct spine gradient for Music/Art variations', () => {
			expect(getSpineGradient('Musik')).toBe('from-pink-300 to-fuchsia-400');
		});

		it('returns correct spine gradient for Computer Science variations', () => {
			expect(getSpineGradient('Informatik')).toBe('from-emerald-300 to-teal-400');
		});

		it('returns default spine gradient for unknown subjects', () => {
			expect(getSpineGradient('Sport')).toBe('from-slate-400 to-slate-500');
		});

		it('returns default spine gradient for empty or null inputs', () => {
			const expected = 'from-slate-400 to-slate-500';
			expect(getSpineGradient(null)).toBe(expected);
			expect(getSpineGradient(undefined)).toBe(expected);
			expect(getSpineGradient('')).toBe(expected);
		});
	});

	describe('formatDate', () => {
		it('formats valid date string correctly', () => {
			// use standard ISO format
			expect(formatDate('2023-10-15')).toBe('15.10.2023');
			expect(formatDate('2023-10-15T12:00:00Z')).toBe('15.10.2023');
		});

		it('returns null for empty string or null input', () => {
			expect(formatDate(null)).toBeNull();
			expect(formatDate('')).toBeNull();
			expect(formatDate(undefined)).toBeNull();
		});

		it('returns null for invalid date string', () => {
			expect(formatDate('not-a-date')).toBeNull();
		});

		it('returns null if Intl.DateTimeFormat throws an exception', () => {
			const spy = vi.spyOn(global.Intl, 'DateTimeFormat').mockImplementation(function () {
				throw new Error('Test Exception');
			});
			expect(formatDate('2023-10-15')).toBeNull();
			spy.mockRestore();
		});
	});
});
