import { describe, expect, it } from 'vitest';
import pruefung from './aussonderungsgruende.faelle.json';
import { AUSSONDERUNGSGRUENDE, GRUND_VORGABE } from './aussonderungsgruende.js';

// Die Auswahl im Löschdialog nennt dieselben vier Gründe mit denselben Wörtern wie das
// Abgangsbuch des Servers; beide lesen aussonderungsgruende.faelle.json.
describe('AUSSONDERUNGSGRUENDE', () => {
	it('liest die geteilten Fälle', () => {
		expect(pruefung.gruende).toHaveLength(4);
	});

	it('nennt Wert, Wort und Reihenfolge der geteilten Fälle', () => {
		expect(AUSSONDERUNGSGRUENDE).toEqual(pruefung.gruende);
	});

	it('wählt den Grund der geteilten Fälle vor', () => {
		expect(GRUND_VORGABE).toBe(pruefung.vorgabe);
		expect(AUSSONDERUNGSGRUENDE.map((g) => g.wert)).toContain(GRUND_VORGABE);
	});
});
