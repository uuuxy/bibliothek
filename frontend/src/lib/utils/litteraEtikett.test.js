import { describe, it, expect } from 'vitest';
import { dekodiereLitteraEtikett } from './litteraEtikett.js';
import pruefung from './litteraEtikett.faelle.json';

// Die Theke rechnet ein Littera-Etikett ohne Netz genauso zurück wie der Server
// (internal/service/littera_etikett.go). Beide Seiten lesen dieselben Fälle; der Go-Test
// littera_etikett_zwilling_test.go prüft zusätzlich, dass diese Datei sie einliest.
describe('dekodiereLitteraEtikett', () => {
	it.each(pruefung.faelle)('$fall: „$scan"', ({ scan, nummer }) => {
		expect(dekodiereLitteraEtikett(scan)).toBe(nummer);
	});

	it('hat gültige und ungültige Fälle', () => {
		expect(pruefung.faelle.filter((f) => f.nummer !== null).length).toBeGreaterThan(4);
		expect(pruefung.faelle.filter((f) => f.nummer === null).length).toBeGreaterThan(4);
	});
});
