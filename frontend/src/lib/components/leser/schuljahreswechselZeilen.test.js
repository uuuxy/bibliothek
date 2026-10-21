import { describe, it, expect } from 'vitest';
import { zeilen } from './schuljahreswechselZeilen.js';

const antwort = {
	promoted_count: 120,
	archived_count: 30,
	dry_run: true,
	mapping_versetzt: 8,
	mapping_entfernt: 2
};

// Eine Klasse wie „14A" oder eine Buchnummer im Feld Klasse versetzt der Server nicht. Die
// Vorschau nennt sie, damit sie vor dem Lauf berichtigt werden kann.
describe('Schuljahreswechsel: Zeilen der Zusammenfassung', () => {
	it('nennt die Klassen, die nicht versetzt werden, bei den Versetzten', () => {
		const [versetzt] = zeilen({ ...antwort, klassen_ausserhalb: ['14A', '4711000123'] });

		expect(versetzt.value).toBe(120);
		expect(versetzt.hint).toBe(
			'Klasse wird um eine Stufe hochgezählt — nicht versetzt, weil kein Jahrgang von 1 bis 13: 14A, 4711000123'
		);
	});

	it('ohne solche Klassen bleibt der Hinweis kurz', () => {
		expect(zeilen(antwort)[0].hint).toBe('Klasse wird um eine Stufe hochgezählt');
		expect(zeilen({ ...antwort, klassen_ausserhalb: [] })[0].hint).toBe(
			'Klasse wird um eine Stufe hochgezählt'
		);
	});

	it('ohne Antwort gibt es keine Zeilen', () => {
		expect(zeilen(null)).toEqual([]);
	});
});
