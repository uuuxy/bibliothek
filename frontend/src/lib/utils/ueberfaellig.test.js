import { describe, it, expect } from 'vitest';
import { istUeberfaellig } from './ueberfaellig.js';

// Eine Regel für jede Stelle, die eine überfällige Ausleihe kennzeichnet: Leserakte, Buchakte,
// Druckliste und Quittung.
describe('istUeberfaellig', () => {
	const jetzt = new Date('2026-10-07T12:00:00Z');

	it('ist nach der Frist überfällig und davor nicht', () => {
		expect(istUeberfaellig({ rueckgabe_frist: '2026-10-07T11:59:00Z' }, jetzt)).toBe(true);
		expect(istUeberfaellig({ rueckgabe_frist: '2026-10-07T12:01:00Z' }, jetzt)).toBe(false);
	});

	it('lässt eine Dauerleihe nie überfällig werden', () => {
		const dauerleihe = { rueckgabe_frist: '2025-01-01T00:00:00Z', ist_dauerleihe: true };
		expect(istUeberfaellig(dauerleihe, jetzt)).toBe(false);
	});

	it('vergleicht ohne Angabe mit dem Augenblick', () => {
		expect(istUeberfaellig({ rueckgabe_frist: '2000-01-01T00:00:00Z' })).toBe(true);
		expect(istUeberfaellig({ rueckgabe_frist: '2999-01-01T00:00:00Z' })).toBe(false);
	});
});
