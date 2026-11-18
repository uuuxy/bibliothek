import { describe, it, expect } from 'vitest';
import {
	kalenderBlatt,
	langesDatum,
	lastOfMonth,
	leseDatum,
	localISO,
	verschiebe
} from './datum.js';

// Regressionstests für den Zeitzonen-Bug: die alte Implementierung ging über
// toISOString() (UTC) und ließ Monatsberichte in UTC+x den letzten Tag verlieren.
describe('dates', () => {
	it('localISO formatiert lokal, nicht über UTC', () => {
		// Lokale Mitternacht — via toISOString() wäre das in UTC+x der Vortag
		expect(localISO(new Date(2026, 0, 31, 0, 0, 0))).toBe('2026-01-31');
		expect(localISO(new Date(2026, 11, 31, 23, 59, 59))).toBe('2026-12-31');
		expect(localISO(new Date(2026, 6, 7))).toBe('2026-07-07');
	});

	it('lastOfMonth liefert den Monatsletzten inkl. 31., 30. und Februar', () => {
		expect(lastOfMonth('2026-01')).toBe('2026-01-31');
		expect(lastOfMonth('2026-04')).toBe('2026-04-30');
		expect(lastOfMonth('2026-02')).toBe('2026-02-28');
		expect(lastOfMonth('2026-12')).toBe('2026-12-31');
	});

	it('lastOfMonth behandelt Schaltjahre korrekt', () => {
		expect(lastOfMonth('2024-02')).toBe('2024-02-29');
		expect(lastOfMonth('2028-02')).toBe('2028-02-29');
		expect(lastOfMonth('2100-02')).toBe('2100-02-28'); // Säkularjahr, kein Schaltjahr
	});
});

// Das Datumsfeld nimmt, was M3 (Date pickers, Accessibility) verlangt: „accept a range of formats
// including dashes, spaces, slashes, dots, and 0 to the left of a single digit month/day“.
describe('leseDatum', () => {
	it('liest Punkte, Striche, Schrägstriche, Leerzeichen und die Form des Browsers', () => {
		const formen = ['1.8.2026', '01.08.2026', '1-8-2026', '1/8/2026', '1 8 2026', '01082026'];
		for (const eingabe of [...formen, '2026-08-01', ' 1.8.2026 ']) {
			expect(leseDatum(eingabe), eingabe).toBe('2026-08-01');
		}
	});

	it('gibt für leer leer und für einen Tag, den es nicht gibt, nichts', () => {
		expect(leseDatum('')).toBe('');
		expect(leseDatum('   ')).toBe('');
		for (const eingabe of ['31.02.2026', '1.8.26', '32.1.2026', '1.13.2026', 'morgen', '2026-8']) {
			expect(leseDatum(eingabe), eingabe).toBeNull();
		}
	});
});

describe('kalenderBlatt', () => {
	it('beginnt am Montag vor dem Ersten und zeigt sechs Wochen', () => {
		// Der 1. August 2026 ist ein Samstag.
		const blatt = kalenderBlatt(2026, 8);
		expect(blatt).toHaveLength(42);
		expect(blatt[0]).toBe('2026-07-27');
		expect(blatt[5]).toBe('2026-08-01');
		expect(blatt[41]).toBe('2026-09-06');
	});

	it('beginnt am Ersten, wenn er ein Montag ist', () => {
		expect(kalenderBlatt(2026, 6)[0]).toBe('2026-06-01');
	});
});

describe('verschiebe', () => {
	it('hält ein Monatsende im Zielmonat und rechnet Tage über Monats- und Jahresgrenzen', () => {
		expect(verschiebe('2026-01-31', { monate: 1 })).toBe('2026-02-28');
		expect(verschiebe('2028-02-29', { jahre: 1 })).toBe('2029-02-28');
		expect(verschiebe('2026-01-15', { monate: -1 })).toBe('2025-12-15');
		expect(verschiebe('2026-12-31', { tage: 1 })).toBe('2027-01-01');
		expect(verschiebe('2026-03-01', { tage: -1 })).toBe('2026-02-28');
	});
});

describe('langesDatum', () => {
	it('nennt Wochentag, Tag, Monat und Jahr', () => {
		expect(langesDatum('2026-08-17')).toBe('Montag, 17. August 2026');
	});
});
