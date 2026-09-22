import { describe, it, expect } from 'vitest';
import { leeresBuchFormular, mehrjahresbandHinweis } from './buch_form_optionen.js';

// Mehrjahresband (docs/OFFEN.md 9.6, 22.09.2026): ein Schalter am Werk, die Zahl kommt aus
// der Spanne „bis". Ein neues Buch beginnt mit „aus"; fehlte der Wert in der Vorlage,
// schickte der Anlegen-Weg das Feld nie mit (siehe den Kommentar an leeresBuchFormular).
// Der Hinweis rechnet dieselbe Regel wie der Server (inventur/mehrjahresband.go).
describe('buch_form_optionen: Mehrjahresband', () => {
	it('die Vorlage eines neuen Buchs beginnt mit „aus"', () => {
		expect(leeresBuchFormular().mehrjahresband).toBe(false);
	});

	it('aus: erklärt beide Zustände, ohne zu rechnen', () => {
		expect(mehrjahresbandHinweis(false, 7, 9)).toMatch(/^Aus: /);
	});

	it('an: nennt das Ende und die Zahl der Schuljahre aus der Spanne', () => {
		const hinweis = mehrjahresbandHinweis(true, 7, 9);
		expect(hinweis).toContain('bis zum Ende von Jahrgang 9');
		expect(hinweis).toContain('Kind der 7');
		expect(hinweis).toContain('nach 3 Schuljahren');
	});

	it('an mit unbrauchbarer Spanne: warnt statt zu rechnen (7 bis 7, bis unter von, leer)', () => {
		for (const [von, bis] of [
			[7, 7],
			[9, 7],
			['', 9],
			[0, 9],
			[7, 14]
		]) {
			expect(mehrjahresbandHinweis(true, von, bis)).toMatch(/^Braucht eine Spanne/);
		}
	});
});
