import { describe, it, expect } from 'vitest';
import { leeresBuchFormular, zielJahrgangOptionen } from './buch_form_optionen.js';

// Mehrjahresband (docs/OFFEN.md 9.6, 22.09.2026): Die Maske bietet dieselben Werte an, die
// der Server annimmt (inventur/ziel_jahrgang.go: 0 oder 5 bis 13), und ein neues Buch
// beginnt mit 0 — ein Schuljahr. Fehlte der Wert in der Vorlage, schickte der Anlegen-Weg
// das Feld nie mit (siehe den Kommentar an leeresBuchFormular).
describe('buch_form_optionen: Zieljahrgang', () => {
	it('die Vorlage eines neuen Buchs beginnt mit einem Schuljahr', () => {
		expect(leeresBuchFormular().zielJahrgang).toBe(0);
	});

	it('bietet 0 und die Jahrgänge 5 bis 13 an, nichts darunter', () => {
		expect(zielJahrgangOptionen.map((o) => o.value)).toEqual([0, 5, 6, 7, 8, 9, 10, 11, 12, 13]);
		expect(zielJahrgangOptionen[0].label).toMatch(/Ein Schuljahr/);
	});
});
