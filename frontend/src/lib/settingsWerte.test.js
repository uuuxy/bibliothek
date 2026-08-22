import { describe, it, expect } from 'vitest';
import { zahlOderUnveraendert } from './settingsWerte.js';

/**
 * Leer ist nicht null-Komma-nichts: Ein geleertes Zahlenfeld darf eine Befristung nicht
 * still abschalten. Nur die getippte 0 heißt „aus"; alles Leere/Unlesbare bleibt, wie es
 * gespeichert ist.
 */
describe('zahlOderUnveraendert', () => {
	it('getippte 0 ist ein Wert (aus), nicht "unverändert"', () => {
		expect(zahlOderUnveraendert(0)).toBe(0);
		expect(zahlOderUnveraendert('0')).toBe(0);
	});
	it('leeres Feld (null/""/undefined) heißt unverändert', () => {
		expect(zahlOderUnveraendert(null)).toBeNull();
		expect(zahlOderUnveraendert('')).toBeNull();
		expect(zahlOderUnveraendert(undefined)).toBeNull();
	});
	it('Unlesbares und Negatives wird nicht zur 0', () => {
		expect(zahlOderUnveraendert('abc')).toBeNull();
		expect(zahlOderUnveraendert(-5)).toBeNull();
		expect(zahlOderUnveraendert(NaN)).toBeNull();
	});
	it('normale Zahlen kommen ganzzahlig durch', () => {
		expect(zahlOderUnveraendert(90)).toBe(90);
		expect(zahlOderUnveraendert('730')).toBe(730);
		expect(zahlOderUnveraendert(15.7)).toBe(15);
	});
});
