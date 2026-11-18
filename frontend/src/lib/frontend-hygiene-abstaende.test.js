import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Ratsche: Abstände aus Googles Skala, M3 (Spacing, Tokens): „The base unit of measurement
// md.sys.measurement.space100 is 8dp" mit den Stufen 0, 2, 4, 6, 8, 10, 12, 14, 16, 20, 24, 32,
// 36, 40, 48, 56, 64 und 72 dp. Darüber erweitert sie in Schritten von 8 dp („The system can be
// extended to add more spacing units … as needed", „Spacing units follow an 8dp scale").
// Gilt für Innenabstand, Außenabstand, Lücke und space-* in .svelte und .js.
//
// Sieht nicht: Klassen aus Ausdrücken, Stile in <style> und CSS-Dateien, Abstände aus Breite
// und Höhe. Den Rand der Arbeitsfläche je Fensterbreite misst e2e/abstaende.spec.js.

// Tailwind-Stufe × 4 = px. Erlaubt: die Stufen der Skala und Vielfache von 8 dp darüber.
const SKALA = new Set([0, 0.5, 1, 1.5, 2, 2.5, 3, 3.5, 4, 5, 6, 8, 9, 10, 12, 14, 16, 18]);
const aufSkala = (/** @type {number} */ n) => SKALA.has(n) || (n > 18 && Number.isInteger(n / 2));

/** Ausnahmen mit Grund. */
const AUSNAHMEN = {
	// Das Kästchen (18 dp) steht in seiner Schicht von 40 dp; der Rand gleicht die 11 dp aus, damit
	// es in der Zeile auf der Linie des Texts steht (material-web: container 18, state-layer 40).
	'src/lib/components/ui/Kaestchen.svelte': ['-m-2.75'],
	// Der Hinweis unter einem Kästchen beginnt unter seiner Beschriftung: Kästchen 18 dp und die
	// Lücke von 12 dp in ui/Kaestchen.
	'src/inventur/lib/components/admin/BuchEingabefelderKategorisierung.svelte': ['pl-7.5']
};

const ABSTAND =
	/(?<![\w-])(-?)((?:p|px|py|pt|pb|pl|pr|ps|pe|m|mx|my|mt|mb|ml|mr|ms|me|gap|gap-x|gap-y|space-x|space-y)-([0-9.]+|px|\[[^\]]+\]))(?![\w-])/g;

describe('Abstände aus der Skala von M3', () => {
	it('nimmt jeden Abstand aus der Skala', () => {
		/** @type {string[]} */
		const abseits = [];
		for (const f of sammleQuelldateien(srcRoot)) {
			if (!/\.(svelte|js)$/.test(f) || /\.test\.js$/.test(f)) continue;
			const pfad = relPfad(f);
			if (pfad.startsWith('src/lib/designer/')) continue;
			const quelle = ohneKommentare(readFileSync(f, 'utf8'));
			for (const [ganz, minus, klasse, wert] of quelle.matchAll(ABSTAND)) {
				if (AUSNAHMEN[pfad]?.includes(ganz)) continue;
				const n = Number(wert);
				if (wert === 'px' || wert.startsWith('[') || !aufSkala(n))
					abseits.push(`${pfad}: ${minus}${klasse}`);
			}
		}
		expect(abseits, 'Abstand abseits der Skala von M3 (Spacing, Tokens)').toEqual([]);
	});
});
