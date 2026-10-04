import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import {
	srcRoot,
	repoFrontend,
	sammleQuelldateien,
	relPfad,
	vergleicheMitBestand,
	ohneKommentare
} from './hygiene-quellen.js';

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
	'src/lib/components/stats/StatsTrendChart.svelte', // Verlaufsgraph, viewBox aus den Daten
	// Kartenhintergründe der Design-Vorlagen (Kopfband, Wellen-Motiv) als data:-URIs —
	// Bilder auf dem Ausweis, keine Bauteil-Symbole (dasselbe Muster wie BuchCoverUpload).
	'src/lib/designer/ausweisVorlagen.js'
];

// ── Ratsche ─────────────────────────────────────────────────────────────────
// Wer eine Datei auf Lucide umstellt, nimmt sie hier heraus. Der Test meldet
// beides: neu hinzugekommene Dateien UND Einträge, die inzwischen sauber sind.
//
// Die verbliebene Datei steht hier aus einem Grund der Bauart, nicht aus Rückstand:
// In BuchCoverUpload steckt das <svg> in einer data:-URI als Platzhalterbild, es ist
// also kein Bauteil-Symbol.
const SVG_BESTAND = ['src/inventur/lib/components/admin/BuchCoverUpload.svelte'];

// ── Veraltete Symbolnamen ───────────────────────────────────────────────────
// Lucide benennt Symbole um und führt den alten Namen als Alias weiter, in seinen
// Typdateien mit @deprecated markiert. Beide Namen zeichnen dasselbe Bild. Fällt der Alias
// in einer späteren Fassung weg, scheitert der Bau an jeder Einfuhr. Die Markierung kommt
// aus der installierten Fassung, nicht aus einer Liste: Was Lucide neu als veraltet
// markiert, zählt ab dem Update mit.
const ALIAS_ORDNER = join(repoFrontend, 'node_modules/@lucide/svelte/dist/aliases');
const ALIAS_DATEIEN = ['aliases.d.ts', 'prefixed.d.ts', 'suffixed.d.ts'];
const VERALTET = /@deprecated([^*]*)\*\/\s*default as (\w+)/g;
const EINFUHR = /import\s*\{([^}]*)\}\s*from\s*['"]@lucide\/svelte['"]/g;

/** Veralteter Name → der Name, den die installierte Fassung an seiner Stelle nennt. */
function veralteteSymbolnamen() {
	/** @type {Map<string, string>} */
	const namen = new Map();
	for (const datei of ALIAS_DATEIEN) {
		const vorher = namen.size;
		for (const [, hinweis, name] of readFileSync(join(ALIAS_ORDNER, datei), 'utf8').matchAll(
			VERALTET
		)) {
			namen.set(name, /\{@link (\w+)\}/.exec(hinweis)?.[1] ?? 'siehe Typdatei');
		}
		// Je Datei geprüft: Eine volle Datei darf eine versiegte nicht verdecken.
		if (namen.size - vorher < 100) {
			throw new Error(
				`${datei} nennt kaum veraltete Namen. Hat Lucide die Aliase entfernt, ist diese ` +
					'Prüfung erledigt (dann scheitert der Bau an einem alten Namen); sonst hat sich ' +
					'die Form der Typdatei geändert.'
			);
		}
	}
	return namen;
}

/**
 * Was eine Quelle aus @lucide/svelte einführt (bei `A as B` der Name A), und wie oft sie das
 * Paket auf eine Art nennt, die hier nicht gelesen wird (Unterpfad, `import *`, Weiterreichen).
 * @param {string} quelle
 */
function lucideEinfuhren(quelle) {
	const code = ohneKommentare(quelle);
	const einfuhren = [...code.matchAll(EINFUHR)];
	const namen = einfuhren
		.flatMap(([, liste]) => liste.split(','))
		.map((teil) => teil.trim().split(/\s+as\s+/)[0])
		.filter(Boolean);
	return { namen, ungelesen: code.split('@lucide/svelte').length - 1 - einfuhren.length };
}

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

	it('liest die Einfuhr in ihren Formen', () => {
		const veraltet = veralteteSymbolnamen();
		const formen = `<script>
			import { Trash2 } from '@lucide/svelte';
			import {
				AlertTriangle as Warnung,
				Check
			} from "@lucide/svelte";
			import { Trash2Icon, LucideUnlock } from '@lucide/svelte';
			// import { Frown } from '@lucide/svelte';
			import { BookMarked } from './eigenes.js';
		</script>`;
		const { namen, ungelesen } = lucideEinfuhren(formen);
		expect(namen.filter((n) => veraltet.has(n))).toEqual([
			'Trash2',
			'AlertTriangle',
			'Trash2Icon',
			'LucideUnlock'
		]);
		expect(ungelesen).toBe(0);
		expect(veraltet.get('AlertTriangle')).toBe('TriangleAlert');

		for (const form of [
			"import Trash2 from '@lucide/svelte/icons/trash-2';",
			"import * as symbole from '@lucide/svelte';",
			"export { Trash2 } from '@lucide/svelte';"
		]) {
			expect(lucideEinfuhren(form).ungelesen, form).toBe(1);
		}
	});

	it('führt kein Symbol unter einem veralteten Namen ein', () => {
		const veraltet = veralteteSymbolnamen();
		/** @type {string[]} */
		const funde = [];
		/** @type {string[]} */
		const ungelesen = [];
		let mitLucide = 0;
		for (const datei of sammleQuelldateien(srcRoot)) {
			const einfuhr = lucideEinfuhren(readFileSync(datei, 'utf8'));
			if (einfuhr.namen.length) mitLucide++;
			if (einfuhr.ungelesen) ungelesen.push(relPfad(datei));
			for (const name of einfuhr.namen) {
				if (veraltet.has(name)) funde.push(`${relPfad(datei)}: ${name} → ${veraltet.get(name)}`);
			}
		}
		expect(mitLucide, 'der Sammler greift ins Leere — dieses Gate wäre still grün').toBeGreaterThan(
			100
		);
		expect(
			ungelesen,
			'Diese Dateien nennen @lucide/svelte auf eine Art, die der Detektor nicht liest. ' +
				"Symbole als benannte Einfuhr holen: import { Name } from '@lucide/svelte'."
		).toEqual([]);
		expect(
			funde,
			'Veraltete Symbolnamen (alter Name → neuer Name). Beide zeichnen dasselbe Bild; ' +
				'der alte fällt mit einer späteren Fassung von Lucide weg.'
		).toEqual([]);
	});
});
