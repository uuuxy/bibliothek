import { describe, it, expect } from 'vitest';
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
	});
});
