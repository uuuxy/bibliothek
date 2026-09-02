import { describe, it, expect } from 'vitest';
import { formatProzent, formatDatum, formatZeitpunkt, formatZahl } from './format.js';

describe('format.js — eine deutsche Schreibweise', () => {
	it('Prozent: Komma, eine Nachkommastelle, geschütztes Leerzeichen; Rohstring vom Backend geht auch', () => {
		expect(formatProzent(23.94)).toBe('23,9 %');
		expect(formatProzent('1.18')).toBe('1,2 %');
		expect(formatProzent(0)).toBe('0 %');
		expect(formatProzent(undefined)).toBe('0 %');
		expect(formatProzent('kaputt')).toBe('0 %');
	});

	it('Datum und Zeitpunkt: immer zweistellig, Zeitpunkt ohne Sekunden', () => {
		const t = new Date(2026, 8, 1, 21, 5, 3);
		expect(formatDatum(t)).toBe('01.09.2026');
		expect(formatZeitpunkt(t)).toBe('01.09.2026, 21:05');
		expect(formatDatum(null)).toBe('');
		expect(formatZeitpunkt('nicht-datum')).toBe('');
	});

	it('Zahl: Tausenderpunkt', () => {
		expect(formatZahl(12345)).toBe('12.345');
	});
});
