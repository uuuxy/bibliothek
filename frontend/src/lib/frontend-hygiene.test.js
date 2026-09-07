import { describe, it, expect } from 'vitest';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, basename } from 'node:path';
import {
	srcRoot,
	repoFrontend,
	sammleQuelldateien,
	relPfad,
	vergleicheMitBestand
} from './hygiene-quellen.js';

// Drei Struktur-Invarianten der Oberfläche, die sich objektiv entscheiden lassen —
// im Gegensatz zu Radien, Schatten und Schriftstärken, die Ermessensfragen sind
// (und wo eine Bewertung nach KLASSENNAMEN in diesem Projekt sowieso danebengreift:
// die @theme-Skala hat rounded-2xl auf 16px und font-bold auf 500 umdefiniert).
// Die dritte — handgesetzte Schriftgrößen — ist hier trotzdem entscheidbar: Eine
// literale Größe UMGEHT die Skala, statt sie zu wählen.
//
// Frei, deterministisch, Millisekunden — wie routing-consistency.test.js. Und im
// Gegensatz zum Git-Hook läuft es auf JEDEM Arbeitsplatz, weil es im Repo liegt.

// Ein Emoji ist ein Zeichen, das von sich aus bunt dargestellt wird
// (Emoji_Presentation), oder ein Piktogramm, das per Variationsselektor U+FE0F
// ausdrücklich als Emoji angefordert wird.
//
// Bewusst NICHT über die rohen Unicode-Blöcke: Dieselben Blöcke enthalten
// Typografie, die legitim ist — ✓, ✕, ✎, ⌘ sind Schriftzeichen, keine Emojis. Ein
// erster Versuch mit Blockbereichen meldete 30 Dateien mehr, alle davon falsch.
const EMOJI = /\p{Emoji_Presentation}|\p{Extended_Pictographic}️/u;

// Handgesetzte Schriftgrößen (`text-[10px]`) umgehen die Typo-Skala: Sie sind die
// einzige Größenangabe, die keine Zeilenhöhe und keine Laufweite mitbringt. Am
// 04.08.2026 standen 96 davon in sechs verschiedenen Werten (5 bis 13 px) für
// dieselbe Aufgabe — Badge, Hilfszeile, Zähler. 78 sind auf `text-label-small`
// gewandert (Material 3, 11 px), der Rest steht unten mit Grund.
// Auch rem/em/pt: Sonst wandert dieselbe Umgehung nur in eine andere Einheit.
const PIXELGROESSE = /text-\[[0-9.]+(px|rem|em|pt)\]/;

// Wo eine literale Größe RICHTIG ist: Diese Stellen zeichnen etwas Physisches oder
// eine Miniatur davon — die Schrift gehört dort zum Bild, nicht zur Bedienoberfläche,
// und muss mit ihm skalieren. Eine Rolle aus der Skala wäre hier schlicht zu groß.
const ZEICHNUNGEN = [
	'src/lib/WebcamCapture.svelte', // Aufnahme-Overlay über dem Kamerabild
	'src/lib/components/labels/LabelPreview.svelte', // 42,3-mm-Etikett in Originalgröße
	'src/lib/designer/CanvasElement.svelte', // Ausweiskarte in mm auf dem Reißbrett
	'src/lib/designer/CardFace.svelte' // dieselbe Karte im Druck, Größen in echten Punkten
];

// ── Ratsche ─────────────────────────────────────────────────────────────────
// Diese Dateien enthielten am 29.07.2026 bereits Emojis. Sie sind bewusst NICHT
// in einem Rutsch bereinigt worden: 29 Dateien kosmetisch anzufassen ist Risiko
// ohne Ertrag, und ein Purge über drei Dateien hatte am selben Tag schon eine
// Regression erzeugt. Stattdessen ist der Bestand eingefroren — Neues kommt nicht
// dazu, Bestehendes wird beim nächsten fachlichen Anfassen der Datei mit erledigt.
//
// Wer eine Datei bereinigt, nimmt sie hier heraus. Der Test meldet beides:
// neu hinzugekommene Dateien UND Einträge, die inzwischen sauber sind.
const EMOJI_BESTAND = [
	'src/lib/StatsDashboard.svelte',
	'src/lib/StudentPrintReceipt.svelte',
	'src/lib/UserManagement.svelte',
	'src/lib/UserManagementTable.svelte',
	'src/lib/WebcamCapture.svelte',
	'src/lib/components/OmniboxBlockAlert.svelte',
	'src/lib/components/OmniboxVormerkungAlert.svelte',
	'src/lib/components/bestellungen/OrderCart.svelte',
	'src/lib/components/layout/RouteFallback.svelte',
	'src/lib/components/stats/StatistikDetailPage.svelte',
	'src/lib/components/stats/StatsTrendChart.svelte',
	'src/lib/components/students/LusdImportView.svelte',
	'src/lib/components/students/PromoteStudentsView.svelte',
	'src/lib/designer/Toolbar.svelte'
];

describe('Oberflächen-Hygiene', () => {
	it('führt keine neuen Emojis ein (Icons kommen aus @lucide/svelte)', () => {
		const betroffen = sammleQuelldateien(srcRoot)
			.filter((f) => EMOJI.test(readFileSync(f, 'utf8')))
			.map(relPfad)
			.sort();

		const { neu, inzwischenSauber } = vergleicheMitBestand(betroffen, [...EMOJI_BESTAND].sort());

		expect(
			neu,
			`Neue Emojis im Quellcode. Bitte ein Lucide-Icon verwenden:\n  ${neu.join('\n  ')}`
		).toEqual([]);

		expect(
			inzwischenSauber,
			`Diese Dateien sind emojifrei — bitte aus EMOJI_BESTAND entfernen, damit die Ratsche greift:\n  ${inzwischenSauber.join('\n  ')}`
		).toEqual([]);
	});

	it('setzt keine Schriftgrößen von Hand (die Skala hat text-label-small)', () => {
		const betroffen = sammleQuelldateien(srcRoot)
			.filter((f) => PIXELGROESSE.test(readFileSync(f, 'utf8')))
			.map(relPfad)
			.sort();

		const { neu, inzwischenSauber } = vergleicheMitBestand(betroffen, [...ZEICHNUNGEN].sort());

		expect(
			neu,
			`Handgesetzte Pixelgrößen. Die Typo-Skala deckt das ab — für alles unter 12 px\n` +
				`ist text-label-small (11 px) die Rolle, darüber text-xs/text-sm:\n  ${neu.join('\n  ')}`
		).toEqual([]);

		expect(
			inzwischenSauber,
			`Diese Dateien setzen keine Pixelgrößen mehr — bitte aus ZEICHNUNGEN entfernen:\n  ${inzwischenSauber.join('\n  ')}`
		).toEqual([]);
	});

	it('enthält keine Komponente, die nirgends importiert wird', () => {
		const dateien = sammleQuelldateien(srcRoot);
		const komponenten = dateien.filter((f) => f.endsWith('.svelte'));

		// App.svelte ist der Einstiegspunkt und wird aus main.js geladen; e2e-Specs
		// zählen als Referenz, damit reine Testkomponenten nicht fälschlich anschlagen.
		const e2eDir = join(repoFrontend, 'e2e');
		const suchraum = [...dateien, ...sammleQuelldateien(e2eDir)];
		const inhalte = new Map(suchraum.map((f) => [f, readFileSync(f, 'utf8')]));

		const verwaist = komponenten
			.filter((k) => basename(k) !== 'App.svelte')
			.filter((k) => {
				const name = basename(k);
				for (const [f, inhalt] of inhalte) {
					if (f !== k && inhalt.includes(name)) return false;
				}
				return true;
			})
			.map(relPfad)
			.sort();

		expect(
			verwaist,
			`Nie importierte Komponenten — bitte löschen. Toter Code kostet beim Suchen und\nverleitet dazu, die falsche Datei zu bearbeiten:\n  ${verwaist.join('\n  ')}`
		).toEqual([]);
	});

	// Die blinde Hälfte des Tests darüber (gefunden am 07.09.2026): Er prüft nur
	// `.svelte`-KOMPONENTEN. Eine exportierte Funktion in einem `.js`-Modul, die niemand
	// importiert, fiel durch — so überlebte `importiereListe` in `admin_api.js` samt der
	// Route `POST /api/books/import` dahinter unbemerkt als Weg ohne Oberfläche
	// (Befund-Register). Backend-seitig gibt es dafür `deadcode`; im Frontend nichts.
	//
	// Der Bestand darunter ist eine ARBEITSLISTE, keine Erlaubnis: Neues fällt sofort
	// auf, Bestehendes wird beim nächsten Anfassen der Datei geklärt — löschen, oder das
	// `export` entfernen, wenn die Funktion nur im eigenen Modul gebraucht wird (das ist
	// bei den Fabriken der Fall, die ihren Store gleich daneben erzeugen).
	const NIE_IMPORTIERT = [];

	it('exportiert keine Funktion, die kein anderes Modul importiert', () => {
		const dateien = sammleQuelldateien(srcRoot);
		const e2eDir = join(repoFrontend, 'e2e');
		const suchraum = [...dateien, ...sammleQuelldateien(e2eDir)];
		const inhalte = new Map(suchraum.map((f) => [f, readFileSync(f, 'utf8')]));

		// Gesucht wird in ALLEN Dateien, Tests eingeschlossen — `sammleQuelldateien` lässt
		// `.test.js` bewusst aus, deshalb kommen sie hier eigens dazu. Die strengere Regel
		// („nur Quellmodule zählen als Leser") war der erste Versuch und ging daneben: Sie
		// meldete die Testhelfer selbst — `hygiene-quellen.js`, `_zuruecksetzenFuerTests`
		// in liveEvents — als verwaist, und die existieren genau für Tests. Ein Gate, das
		// seine eigenen Werkzeuge anklagt, wird abgeschaltet statt befolgt.
		const module = dateien.filter((f) => f.endsWith('.js'));
		const testDateien = sammleTestdateien(srcRoot);
		// DIESE Datei zählt nicht als Leser: Ihre Bestandsliste unten nennt jeden Namen,
		// und der Test fände ihn dort wieder — jede eingetragene Ausfuhr sähe benutzt aus,
		// die Ratsche meldete für immer „alles sauber". Genau die Bugklasse „lügende
		// Ratsche" (docs/sweeps.md), hier beim Bau in die eigene Falle gelaufen und
		// bemerkt, weil die erwarteten zwölf Funde plötzlich null waren.
		const leser = [...inhalte, ...testDateien.map((f) => [f, readFileSync(f, 'utf8')])].filter(
			([f]) => !f.endsWith('frontend-hygiene.test.js')
		);

		/** @type {string[]} */
		const verwaist = [];
		for (const datei of module) {
			const quelle = /** @type {string} */ (inhalte.get(datei) ?? '');
			for (const [, name] of quelle.matchAll(/^export (?:async )?function ([A-Za-z0-9_]+)/gm)) {
				const wort = new RegExp(`\\b${name}\\b`);
				const anderswo = leser.some(([f, inhalt]) => f !== datei && wort.test(inhalt));
				if (!anderswo) verwaist.push(`${relPfad(datei)} :: ${name}`);
			}
		}
		verwaist.sort();

		const { neu, inzwischenSauber } = vergleicheMitBestand(verwaist, [...NIE_IMPORTIERT].sort());

		expect(
			neu,
			`Exportiert, aber von keinem anderen Modul importiert. Entweder wird die Funktion\n` +
				`gebraucht — dann fehlt der Aufrufer — oder sie ist tot:\n  ${neu.join('\n  ')}`
		).toEqual([]);

		expect(
			inzwischenSauber,
			`Diese Ausfuhren haben inzwischen einen Aufrufer (oder sind weg) — bitte aus\nNIE_IMPORTIERT entfernen:\n  ${inzwischenSauber.join('\n  ')}`
		).toEqual([]);
	});
});

/** Alle `.test.js` unter p — das Gegenstück zu sammleQuelldateien, das sie auslässt.
 * @param {string} p @returns {string[]} */
function sammleTestdateien(p) {
	/** @type {string[]} */
	const out = [];
	for (const entry of readdirSync(p)) {
		if (entry === 'node_modules') continue;
		const full = join(p, entry);
		if (statSync(full).isDirectory()) out.push(...sammleTestdateien(full));
		else if (entry.endsWith('.test.js')) out.push(full);
	}
	return out;
}
