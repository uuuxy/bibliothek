import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad, vergleicheMitBestand } from './hygiene-quellen.js';

// Reiterleisten kommen aus components/ui/Reiter.svelte — dieselbe Invariante wie bei
// Suchfeldern und Symbolen. Von Hand gebaute Leisten laufen in Höhe, Gewicht und Indikator
// auseinander.
const HANDGEBAUT = /role=(["'])tab\1/;

// Kein Bestand: Jede Leiste der Anwendung kommt aus dem Bauteil.
/** @type {string[]} */
const BESTAND = [];

// Die Komponente selbst trägt das role="tab" — sie ist die Quelle, nicht ein Verstoß.
const QUELLE = 'src/lib/components/ui/Reiter.svelte';

describe('Reiter-Hygiene', () => {
	it('baut keine neuen Reiterleisten von Hand (sie kommen aus Reiter.svelte)', () => {
		const betroffen = sammleQuelldateien(srcRoot)
			.filter((f) => HANDGEBAUT.test(readFileSync(f, 'utf8')))
			.map(relPfad)
			.filter((f) => f !== QUELLE)
			.sort();

		const { neu, inzwischenSauber } = vergleicheMitBestand(betroffen, BESTAND);

		expect(
			neu,
			`Neue handgebaute Reiterleiste(n):\n  ${neu.join('\n  ')}\n` +
				`Reiter kommen aus components/ui/Reiter.svelte — sonst laufen sie auseinander ` +
				`wie die Suchfelder (zehn Kopien, sieben verschiedene Maße).`
		).toEqual([]);

		expect(
			inzwischenSauber,
			`Diese Dateien sind auf Reiter.svelte umgestellt — danke.\n  ${inzwischenSauber.join('\n  ')}\n` +
				`Bitte aus BESTAND in dieser Datei entfernen, damit die Ratsche greift.`
		).toEqual([]);
	});

	it('erkennt eine handgebaute Reiterleiste überhaupt', () => {
		// Gegenprobe am DETEKTOR: Ein Muster, das nichts findet, meldet ewig „alles gut".
		expect(HANDGEBAUT.test('<button role="tab" aria-selected={x}>')).toBe(true);
		expect(HANDGEBAUT.test("<button role='tab'>")).toBe(true);
		expect(HANDGEBAUT.test('<button role="tablist">')).toBe(false);
		expect(HANDGEBAUT.test('<button onclick={x}>Reiter</button>')).toBe(false);
	});
});
