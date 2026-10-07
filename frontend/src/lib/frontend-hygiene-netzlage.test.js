import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { sammleQuelldateien, srcRoot, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Ob der Server erreichbar ist, misst stores/netzLage an /health. `navigator.onLine` ist eine
// Auskunft des Betriebssystems: Ein Browser kann „offline" melden, obwohl der Server
// antwortet, und was an dem Wert hängt, läuft an so einem Rechner nie (Warteschlange der
// Theke, Sperre, Abgleich der Buchnummern). Wer ihn außerhalb von netzLage liest, wird hier rot.
//
// Sieht nicht: den Wert über eine Variable (`const n = navigator; n.onLine`) und
// Entscheidungen, die allein an den Ereignissen `online` und `offline` hängen.
const ERLAUBT = ['src/lib/stores/netzLage.svelte.js'];
const MUSTER = /\bnavigator\s*(\?\.|\.)\s*onLine\b|\[\s*['"`]onLine['"`]\s*\]/;

describe('Netz-Zustand nur über netzLage', () => {
	it('keine Quelldatei außer netzLage liest navigator.onLine', () => {
		const lesend = sammleQuelldateien(srcRoot)
			.filter((f) => MUSTER.test(ohneKommentare(readFileSync(f, 'utf8'))))
			.map(relPfad);
		// Sähe der Detektor die erlaubte Stelle nicht, mäße er nichts.
		expect(lesend, 'der Detektor findet netzLage nicht').toEqual(expect.arrayContaining(ERLAUBT));
		expect(
			lesend.filter((f) => !ERLAUBT.includes(f)),
			'navigator.onLine außerhalb von netzLage — netzLage.offline nehmen'
		).toEqual([]);
	});

	it('Selbstprobe: der Detektor fasst die Formen', () => {
		for (const form of [
			'if (navigator.onLine) starte();',
			'if (!navigator.onLine) return;',
			'const an = window.navigator.onLine;',
			'const an = navigator?.onLine ?? true;',
			"const an = navigator['onLine'];"
		]) {
			expect(MUSTER.test(form), form).toBe(true);
		}
		expect(MUSTER.test("window.addEventListener('online', melde)")).toBe(false);
		expect(MUSTER.test(ohneKommentare('// navigator.onLine ist eine Behauptung\n'))).toBe(false);
	});
});
