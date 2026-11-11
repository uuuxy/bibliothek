import { describe, it, expect } from 'vitest';
import { abgangsjahrAm } from './abgangsjahr.js';
import pruefung from './abgangsjahr.faelle.json';

// Browser-Seite der geteilten Prüffälle; der Server liest dieselbe Datei in
// TestAbgaengerJahrAm_WieImBrowser (repository).
describe('Abgangsjahr aus der Klasse, geteilte Prüffälle', () => {
	it('liest die gemeinsame Datei überhaupt ein', () => {
		// Ohne diesen Boden liefe die Suite still grün, wenn die Datei umbenannt wird.
		expect(pruefung.faelle.length).toBeGreaterThanOrEqual(15);
	});

	for (const fall of pruefung.faelle) {
		it(`${fall.fall}: „${fall.klasse}“ am ${fall.datum}`, () => {
			expect(abgangsjahrAm(fall.klasse, new Date(`${fall.datum}T12:00:00`))).toBe(fall.soll);
		});
	}
});
