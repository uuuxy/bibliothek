import { describe, it, expect } from 'vitest';
import pruefung from './titelReihenfolge.faelle.json';

// Die Titelliste ordnet der Server (GET /api/books, sortiereBuecherNachTitel in
// inventur/endpunkte_buecher_lesen.go). Die Listen der Oberfläche ordnen mit dieser Regel.
// Beide Seiten lesen dieselben Fälle; der Go-Test steht in endpunkte_buecher_lesen_test.go.
const ORDNUNG = new Intl.Collator('de', { numeric: true });

describe('Reihenfolge nach Titel: dieselbe wie am Server', () => {
	it('liest die geteilten Fälle', () => {
		expect(pruefung.faelle.length).toBeGreaterThanOrEqual(10);
	});

	for (const { fall, erwartet } of pruefung.faelle) {
		it(fall, () => {
			expect(erwartet.toReversed().sort(ORDNUNG.compare)).toEqual(erwartet);
		});
	}
});
