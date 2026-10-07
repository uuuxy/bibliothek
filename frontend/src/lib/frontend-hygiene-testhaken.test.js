import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { basename } from 'node:path';
import { srcRoot, relPfad, ohneKommentare, sammleTestdateien } from './hygiene-quellen.js';

// Ein Haken gibt keine Attrappe zurück.
//
// `beforeEach(() => attrappe.mockReset())` gibt zurück, was mockReset liefert: die Attrappe
// selbst. Gibt ein Haken eine Funktion zurück, ruft Vitest sie nach dem Test als Aufräumer
// auf. Lehnt die Attrappe dann ab (mockRejectedValue), wird der Test rot, obwohl seine
// Erwartungen stimmen, und die Meldung zeigt auf die Zeile des Fehlers, nicht auf den Haken.
// Mit geschweiften Klammern gibt der Haken nichts zurück.
//
// Der Detektor sucht das Merkmal: ein Haken mit Pfeil ohne Klammern, dessen Ausdruck eine
// Methode der Attrappe ruft (mockReset, mockClear, mockResolvedValue …; jede gibt die
// Attrappe zurück). `vi.clearAllMocks()` gibt `vi` zurück, kein Aufräumer.
const HAKEN_GIBT_ATTRAPPE_ZURUECK =
	/\b(?:beforeEach|afterEach|beforeAll|afterAll)\(\s*(?:async\s*)?\(\)\s*=>\s*(?!\{)[^;{]*\.mock[A-Z]\w*\(/;

describe('Haken in Tests', () => {
	it('der Detektor erkennt jede Form und lässt die erlaubten stehen', () => {
		const verboten = [
			'beforeEach(() => vi.mocked(apiFetch).mockReset());',
			'beforeEach(() => spion.mockClear());',
			'afterEach(() => vi.mocked(lade).mockRestore());',
			'beforeAll(async () => vi.mocked(lade).mockResolvedValue([]));',
			'beforeEach(() =>\n\tvi.mocked(einSehrLangerName).mockReset()\n);'
		];
		for (const form of verboten) {
			expect(HAKEN_GIBT_ATTRAPPE_ZURUECK.test(form), form).toBe(true);
		}
		const erlaubt = [
			'beforeEach(() => {\n\tvi.mocked(apiFetch).mockReset();\n});',
			'beforeEach(() => vi.clearAllMocks());',
			'afterEach(() => vi.useRealTimers());',
			"beforeEach(() => vi.clearAllMocks());\nit('x', () => {\n\tspion.mockReset();\n});"
		];
		for (const form of erlaubt) {
			expect(HAKEN_GIBT_ATTRAPPE_ZURUECK.test(form), form).toBe(false);
		}
	});

	it('kein Haken gibt eine Attrappe zurück', () => {
		/** @type {string[]} */
		const betroffen = [];
		let mitHaken = 0;
		for (const datei of sammleTestdateien(srcRoot)) {
			// Diese Datei nennt die verbotenen Formen als Beispiele.
			if (basename(datei) === 'frontend-hygiene-testhaken.test.js') continue;
			const quelle = ohneKommentare(readFileSync(datei, 'utf8'));
			if (/\b(?:beforeEach|afterEach)\(/.test(quelle)) mitHaken++;
			if (HAKEN_GIBT_ATTRAPPE_ZURUECK.test(quelle)) betroffen.push(relPfad(datei));
		}

		// Ohne diese Zusicherung wäre ein leerer Suchlauf ein still grüner Test.
		expect(
			mitHaken,
			'kaum Testdateien mit Haken gefunden — der Detektor misst nichts'
		).toBeGreaterThan(50);
		expect(
			betroffen,
			`Diese Testdateien geben aus einem Haken eine Attrappe zurück:\n` +
				betroffen.map((f) => `  ${f}`).join('\n') +
				`\nVitest ruft sie nach dem Test als Aufräumer auf. Den Rumpf in geschweifte Klammern\n` +
				`setzen: beforeEach(() => { attrappe.mockReset(); });`
		).toEqual([]);
	});
});
