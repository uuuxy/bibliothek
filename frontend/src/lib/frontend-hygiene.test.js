import { describe, it, expect } from 'vitest';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join, relative, basename } from 'node:path';

// Zwei Struktur-Invarianten der Oberfläche, die sich objektiv entscheiden lassen —
// im Gegensatz zu Radien, Schatten und Schriftstärken, die Ermessensfragen sind
// (und wo eine Bewertung nach KLASSENNAMEN in diesem Projekt sowieso danebengreift:
// die @theme-Skala in app.css hat rounded-2xl auf 8px und font-bold auf 600
// umdefiniert).
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
	'src/inventur/lib/components/admin/BuchEingabefelder.svelte',
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
