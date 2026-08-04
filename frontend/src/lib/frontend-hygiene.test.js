import { describe, it, expect } from 'vitest';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join, relative, basename } from 'node:path';

// Drei Struktur-Invarianten der Oberfläche, die sich objektiv entscheiden lassen —
// im Gegensatz zu Radien, Schatten und Schriftstärken, die Ermessensfragen sind
// (und wo eine Bewertung nach KLASSENNAMEN in diesem Projekt sowieso danebengreift:
// die @theme-Skala hat rounded-2xl auf 16px und font-bold auf 500 umdefiniert).
// Die dritte — handgesetzte Schriftgrößen — ist hier trotzdem entscheidbar: Eine
// literale Größe UMGEHT die Skala, statt sie zu wählen.
//
// Frei, deterministisch, Millisekunden — wie routing-consistency.test.js. Und im
// Gegensatz zum Git-Hook läuft es auf JEDEM Arbeitsplatz, weil es im Repo liegt.

const libDir = dirname(fileURLToPath(import.meta.url));
const srcRoot = join(libDir, '..');
const repoFrontend = join(srcRoot, '..');

/** @param {string} p @returns {string[]} */
function sammleQuelldateien(p) {
	/** @type {string[]} */
	const out = [];
	for (const entry of readdirSync(p)) {
		if (entry === 'node_modules') continue;
		const full = join(p, entry);
		if (statSync(full).isDirectory()) out.push(...sammleQuelldateien(full));
		else if (/\.(svelte|js)$/.test(entry) && !entry.endsWith('.test.js')) out.push(full);
	}
	return out;
}

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
	'src/inventur/lib/components/BuchKarteCover.svelte', // Buchcover-Attrappe
	'src/inventur/lib/components/KlassenBuchKachelStartseite.svelte', // dito, in der Kachel
	'src/inventur/lib/components/admin/KlassenBuchKachel.svelte', // dito
	'src/lib/WebcamCapture.svelte', // Aufnahme-Overlay über dem Kamerabild
	'src/lib/components/labels/LabelPreview.svelte', // 42,3-mm-Etikett in Originalgröße
	'src/lib/components/mahnwesen/MahnwesenTable.svelte', // Buchrücken-Symbol, 8×10 px
	'src/lib/designer/CanvasArea.svelte', // Ausweiskarte in mm auf dem Reißbrett
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
	'src/lib/AdminAuditLog.svelte',
	'src/lib/AuditLog.svelte',
	'src/lib/BookAkteMeta.svelte',
	'src/lib/Monitor.svelte',
	'src/lib/OpacSearch.svelte',
	'src/lib/PermissionManager.svelte',
	'src/lib/StatsDashboard.svelte',
	'src/lib/StudentPrintReceipt.svelte',
	'src/lib/UserManagement.svelte',
	'src/lib/UserManagementTable.svelte',
	'src/lib/WebcamCapture.svelte',
	'src/lib/components/OmniboxBlockAlert.svelte',
	'src/lib/components/OmniboxVormerkungAlert.svelte',
	'src/lib/components/bestellungen/OrderCart.svelte',
	'src/lib/components/layout/RouteFallback.svelte',
	'src/lib/components/layout/Sidebar.svelte',
	'src/lib/components/stats/StatistikDetailPage.svelte',
	'src/lib/components/stats/StatsTrendChart.svelte',
	'src/lib/components/students/LusdImportView.svelte',
	'src/lib/components/students/PromoteStudentsView.svelte',
	'src/lib/designer/Toolbar.svelte',
	'src/lib/permissionMetadata.js'
];

/** @param {string} f */
const relPfad = (f) => relative(repoFrontend, f).split('\\').join('/');

describe('Oberflächen-Hygiene', () => {
	it('führt keine neuen Emojis ein (Icons kommen aus @lucide/svelte)', () => {
		const betroffen = sammleQuelldateien(srcRoot)
			.filter((f) => EMOJI.test(readFileSync(f, 'utf8')))
			.map(relPfad)
			.sort();

		const bestand = [...EMOJI_BESTAND].sort();
		const neu = betroffen.filter((f) => !bestand.includes(f));
		const inzwischenSauber = bestand.filter((f) => !betroffen.includes(f));

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

		const bestand = [...ZEICHNUNGEN].sort();
		const neu = betroffen.filter((f) => !bestand.includes(f));
		const inzwischenSauber = bestand.filter((f) => !betroffen.includes(f));

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
});
