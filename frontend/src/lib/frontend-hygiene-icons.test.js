import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad, vergleicheMitBestand } from './hygiene-quellen.js';

// Vierte Struktur-Invariante, dieselbe Bauart wie die drei in frontend-hygiene.test.js:
// Symbole kommen aus @lucide/svelte, nicht aus handgeschriebenem <svg>.
//
// Warum das zählt: Derselbe Pfad steht mehrfach im Baum — das Kiosk-Symbol allein
// viermal (menu.js, Sidebar 2×, BookTableToolbar). Jede Kopie hat ihre eigene
// Strichstärke, ihre eigene Kantenform und altert für sich. Material 3 verlangt an
// erster Stelle Konsistenz; ein Symbolsatz, der an 67 Stellen nachgezeichnet wird,
// ist genau das Gegenteil. Lucide liefert die Rolle EINMAL.
//
// Gemessen am 07.08.2026: 140 <svg> in 68 Dateien, 46 von 180 Komponenten nutzten
// Lucide. Nach dem Tausch am 08.08.2026: 7 <svg> in 4 Dateien, 105 von 182
// Komponenten nutzen Lucide. Die Liste ist ein Bestand, KEINE Erlaubnis.
const SVG = /<svg[\s>]/;

// Echte Zeichnungen: Hier entsteht ein Bild, kein Symbol. Erkennbar am viewBox —
// alles andere im Baum ist 24×24 (bzw. 20×20), also eine Symbolfläche.
// Diese Liste schrumpft NICHT; sie ist die Ausnahme, nicht der Rückstand.
const ZEICHNUNGEN = [
	'src/lib/components/stats/StatsTrendChart.svelte' // Verlaufsgraph, viewBox aus den Daten
];

// ── Ratsche ─────────────────────────────────────────────────────────────────
// Wer eine Datei auf Lucide umstellt, nimmt sie hier heraus. Der Test meldet
// beides: neu hinzugekommene Dateien UND Einträge, die inzwischen sauber sind.
//
// Die drei Verbliebenen stehen hier aus STRUKTURELLEN Gründen, nicht aus Rückstand —
// damit niemand sie ein zweites Mal untersucht:
//   BuchCoverUpload       das <svg> steckt in einer data:-URI als Platzhalterbild,
//                         ist also gar kein Bauteil-Symbol.
//   DataManagement        rendert `d={iconPath}`: ein generischer Schnipsel, der
//                         Pfad-STRINGS als Parameter nimmt.
//   MahnwesenDruckMenue   dasselbe Muster mit `d={BRIEF}`.
// Die beiden letzten umzustellen heißt, ihre Snippet-Signatur von „Pfad" auf
// „Komponente" zu ändern — eine eigene Entscheidung, kein Nachziehen.
const SVG_BESTAND = [
	'src/inventur/lib/components/admin/BuchCoverUpload.svelte',
	'src/lib/components/admin/DataManagement.svelte',
	'src/lib/components/mahnwesen/MahnwesenDruckMenue.svelte'
];

describe('Symbol-Hygiene', () => {
	it('zeichnet keine neuen Symbole von Hand (Icons kommen aus @lucide/svelte)', () => {
		const betroffen = sammleQuelldateien(srcRoot)
			.filter((f) => SVG.test(readFileSync(f, 'utf8')))
			.map(relPfad)
			.filter((f) => !ZEICHNUNGEN.includes(f))
			.sort();

		const { neu, inzwischenSauber } = vergleicheMitBestand(betroffen, [...SVG_BESTAND].sort());

		expect(
			neu,
			`Handgezeichnete Symbole. @lucide/svelte hat die Rolle bereits — importieren\n` +
				`statt nachzeichnen, sonst driften Strichstärke und Kantenform auseinander:\n  ${neu.join('\n  ')}`
		).toEqual([]);

		expect(
			inzwischenSauber,
			`Diese Dateien zeichnen nicht mehr selbst — bitte aus SVG_BESTAND entfernen,\ndamit die Ratsche greift:\n  ${inzwischenSauber.join('\n  ')}`
		).toEqual([]);
	});
});
