import { describe, it, expect } from 'vitest';
import { bestandHinweis, leeresBuchFormular, mehrjahresbandHinweis } from './buch_form_optionen.js';

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

	it('an: akzeptiert auch Strings aus HTML-Eingabefeldern', () => {
		const hinweis = mehrjahresbandHinweis(true, '5', '10');
		expect(hinweis).toContain('bis zum Ende von Jahrgang 10');
		expect(hinweis).toContain('Kind der 5');
		expect(hinweis).toContain('nach 6 Schuljahren');
	});

	it('an: erlaubt die maximale Spanne von Jahrgang 1 bis 13', () => {
		const hinweis = mehrjahresbandHinweis(true, 1, 13);
		expect(hinweis).toContain('bis zum Ende von Jahrgang 13');
		expect(hinweis).toContain('Kind der 1');
		expect(hinweis).toContain('nach 13 Schuljahren');
	});
});

// Ein Titel ohne Exemplar steht in keinem Katalog (docs/OFFEN.md 9.4). Ein neues Buch beginnt
// deshalb mit einem Exemplar, und wer 0 einträgt, liest vor dem Speichern, wo der Titel landet.
describe('buch_form_optionen: Bestand eines neuen Buchs', () => {
	it('die Vorlage beginnt mit einem Exemplar', () => {
		expect(leeresBuchFormular().stock).toBe(1);
	});

	it('neuer Titel ohne Exemplar: nennt die Sicht, in der er steht', () => {
		for (const bestand of [0, '0', '', null, undefined]) {
			expect(bestandHinweis(null, bestand)).toContain('„Ohne Exemplare“');
		}
	});

	it('kein Hinweis mit Bestand und keiner an einem vorhandenen Titel', () => {
		expect(bestandHinweis(null, 1)).toBe('');
		expect(bestandHinweis(null, '3')).toBe('');
		expect(bestandHinweis('titel-1', 0)).toBe('');
	});

	it('vorhandener Titel mit geleertem Feld: Der Bestand bleibt, und der Hinweis sagt es', () => {
		for (const leer of ['', null, undefined]) {
			expect(bestandHinweis('titel-1', leer)).toBe('Ohne Zahl bleibt der Bestand, wie er ist.');
		}
	});
});
