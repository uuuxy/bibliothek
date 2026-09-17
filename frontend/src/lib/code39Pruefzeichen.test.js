import { describe, it, expect } from 'vitest';
import { ohnePruefzeichen, pruefzeichen } from './code39Pruefzeichen.js';
import pruefung from './code39.faelle.json';

// Die JavaScript-Seite der geteilten Prüffälle (pkg/code39/zwilling_test.go prüft die
// Go-Seite). Rechnen beide verschieden, gilt ein altes Etikett offline als unklar, das
// online gebucht wird — oder umgekehrt.
describe('Code-39-Prüfzeichen, geteilte Prüffälle', () => {
	const faelle = pruefung.faelle;

	it('liest die gemeinsame Datei überhaupt ein', () => {
		// Ohne diesen Boden liefe die Suite still grün, wenn die Datei umbenannt wird
		// oder ihr Format sich ändert.
		expect(faelle.length).toBeGreaterThanOrEqual(9);
		expect(faelle.filter((f) => f.kern !== null).length).toBeGreaterThanOrEqual(5);
		expect(faelle.filter((f) => f.kern === null).length).toBeGreaterThanOrEqual(4);
	});

	for (const f of faelle) {
		it(`${f.fall}: ${JSON.stringify(f.scan)}`, () => {
			expect(ohnePruefzeichen(f.scan)).toBe(f.kern);
		});
	}

	it('rechnet die beiden gemessenen Werte nach', () => {
		// Vorwärts, nicht nur rückwärts: „B-10001" muss die '6' ergeben, die am echten
		// Etikett steht.
		expect(pruefzeichen('B-10001')).toBe('6');
		expect(pruefzeichen('A-10003')).toBe('7');
	});

	it('meldet ein Zeichen, das Code 39 nicht kennt', () => {
		expect(pruefzeichen('müller')).toBeNull();
	});
});
