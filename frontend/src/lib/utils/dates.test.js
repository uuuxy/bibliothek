import { describe, it, expect } from 'vitest';
import { localISO, lastOfMonth } from './dates.js';

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
