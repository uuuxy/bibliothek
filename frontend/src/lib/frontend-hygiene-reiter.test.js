import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad, vergleicheMitBestand } from './hygiene-quellen.js';

// Reiterleisten kommen aus components/ui/Reiter.svelte — dieselbe Invariante wie bei
// Suchfeldern und Symbolen.
//
// Anlass: Am 23.08.2026 brauchte das Kollegiums-Portal Reiter, und es gab VIER
// handgebaute Leisten im Haus (Medienkatalog, Bestellwesen, Buch-Akte,
// Inventur-Startseite). Eine fünfte Kopie hätte die Reihe fortgesetzt, die bei den
// Suchfeldern in zehn Fassungen mit sieben verschiedenen Maßen endete. Also erst das
// gemeinsame Bauteil, dann der Reiter.
//
// Die vier Bestandsfälle sind bewusst NICHT in einem Rutsch umgestellt: Es sind drei
// täglich benutzte Bildschirme, und ein Massen-Refactoring an einem Tag hat in diesem
// Projekt schon zweimal eine Regression erzeugt. Sie stehen hier eingefroren und werden
// bei ihrem nächsten fachlichen Anfassen nachgezogen.
const HANDGEBAUT = /role=(["'])tab\1/;

// BestellWorkspace ist am 04.09.2026 nachgezogen — genau der Fall, den der Kommentar
// oben vorsieht („bei ihrem nächsten fachlichen Anfassen"). Anlass war die gemessene
// Höhe: 34 px gegen 32 px überall sonst, verursacht vom `border-b-2` im Textfluss.
// Die Ratsche rückt damit von drei auf zwei Bestandsfälle.
const BESTAND = ['src/lib/BookAkte.svelte', 'src/lib/MediaCatalog.svelte'];

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
