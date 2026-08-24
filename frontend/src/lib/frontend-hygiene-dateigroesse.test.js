import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, relPfad, sammleQuelldateien } from './hygiene-quellen.js';

// Ratsche auf die Komponenten-Regel „≤ 200 Zeilen pro .svelte-Datei"
// (docs/ARCHITECTURE.md, Abschnitt „Komponenten-Regeln").
//
// Die Regel steht dort ohne Einschränkung — geprüft hat sie nie jemand. Am 23.08.2026
// brachen sie 43 von 206 Dateien, die größte mit 412 Zeilen. Eine Regel, die ein
// Fünftel des Baums verletzt und nichts davon merkt, ist keine Regel, sondern eine
// Absichtserklärung. Aufgefallen ist es beim Nachprüfen des Befund-Registers: Dort
// stand sie nicht, weil sie nie jemand gemessen hat.
//
// Die Ratsche macht daraus einen Zustand, der nur besser werden kann:
//
//   - Eine NEUE Datei über 200 Zeilen ist rot. Das ist der eigentliche Zweck.
//   - Eine Datei aus dem Bestand, die WEITER wächst, ist ebenfalls rot — sonst wäre
//     die Ausnahme ein Freibrief.
//   - Wer eine Datei unter 200 bringt, muss sie austragen. Der Bestand ist damit eine
//     Arbeitsliste, keine Duldung.
//
// Bewusst NICHT: ein Limit für .js/.go. Die Regel in ARCHITECTURE.md gilt den
// Svelte-Komponenten, und nur dafür steht hier ein Gate.
const BESTAND = {
	'src/App.svelte': 208,
	'src/inventur/lib/components/BuchKarte.svelte': 240,
	'src/inventur/lib/components/KlassenBuchKachelStartseite.svelte': 225,
	'src/inventur/lib/components/admin/BookTable.svelte': 241,
	'src/inventur/lib/components/admin/BookTableZeile.svelte': 203,
	'src/inventur/lib/components/admin/ClassAssignmentDialog.svelte': 221,
	'src/inventur/lib/components/admin/KlassenBuchKachel.svelte': 228,
	'src/inventur/lib/components/admin/KlassenUebersicht.svelte': 215,
	'src/inventur/routes/admin/+page.svelte': 211,
	'src/lib/BestellBestaetigung.svelte': 254,
	'src/lib/BestellWorkspace.svelte': 306,
	'src/lib/BorrowedBooksList.svelte': 291,
	'src/lib/KollegiumPortal.svelte': 288,
	'src/lib/MailTemplates.svelte': 205,
	'src/lib/Monitor.svelte': 222,
	'src/lib/Omnibox.svelte': 298,
	'src/lib/Router.svelte': 248,
	'src/lib/StatsDashboard.svelte': 391,
	'src/lib/StudentDirectory.svelte': 222,
	'src/lib/StudentEditSheet.svelte': 226,
	'src/lib/StudentProfile.svelte': 258,
	'src/lib/StudentProfileActions.svelte': 244,
	'src/lib/StudentProfileCard.svelte': 222,
	'src/lib/UnifiedInventory.svelte': 325,
	'src/lib/UserManagement.svelte': 258,
	'src/lib/components/BookExemplarCard.svelte': 229,
	'src/lib/components/GeraeteVerwaltung.svelte': 227,
	'src/lib/components/admin/DataManagement.svelte': 301,
	'src/lib/components/bestellungen/OrderRecommendations.svelte': 223,
	'src/lib/components/bestellungen/OrderSearch.svelte': 287,
	'src/lib/components/bestellungen/SupplierManager.svelte': 225,
	'src/lib/components/labels/EtikettenNachdruck.svelte': 412,
	'src/lib/components/layout/Sidebar.svelte': 206,
	'src/lib/components/mahnwesen/MahnwesenTable.svelte': 249,
	'src/lib/components/signaturen/SystematikVerwaltung.svelte': 211,
	'src/lib/components/stats/StatsTrendChart.svelte': 243,
	'src/lib/components/students/LusdImportView.svelte': 394,
	'src/lib/components/students/PromoteStudentsView.svelte': 212,
	'src/lib/components/ui/KlassenVersandDialog.svelte': 206,
	'src/lib/components/ui/Select.svelte': 207,
	'src/lib/designer/CanvasArea.svelte': 309
};

describe('Komponenten-Regel: hoechstens 200 Zeilen je .svelte-Datei', () => {
	const dateien = sammleQuelldateien(srcRoot).filter((p) => p.endsWith('.svelte'));
	// Wie `wc -l` zaehlen, also Zeilenumbrueche: Ein abschliessendes \n erzeugt beim
	// Split ein leeres letztes Element, und das ist keine Zeile. Ohne diese Zeile meldete
	// das Gate JEDE Datei um eins zu gross und damit den ganzen Bestand als gewachsen.
	const zeilenVon = (absolut) => {
		const teile = readFileSync(absolut, 'utf8').split('\n');
		if (teile.at(-1) === '') teile.pop();
		return teile.length;
	};
	const aktuell = new Map(dateien.map((a) => [relPfad(a), zeilenVon(a)]));

	it('durchsucht ueberhaupt Dateien — sonst waere das Gate wertlos gruen', () => {
		expect(dateien.length).toBeGreaterThan(150);
	});

	it('kennt keine NEUE Datei ueber 200 Zeilen', () => {
		const neu = [];
		for (const [pfad, zeilen] of aktuell) {
			if (zeilen > 200 && !(pfad in BESTAND)) neu.push(`${pfad} (${zeilen})`);
		}
		expect(
			neu,
			'Neue Komponente ueber 200 Zeilen. docs/ARCHITECTURE.md verlangt hoechstens 200 — ' +
				'aufteilen (Teilkomponente, {#snippet}, Daten nach .js) oder die Regel dort aendern.'
		).toEqual([]);
	});

	it('laesst keine Datei aus dem Bestand weiter wachsen', () => {
		const gewachsen = [];
		for (const [pfad, erlaubt] of Object.entries(BESTAND)) {
			const zeilen = aktuell.get(pfad);
			if (zeilen !== undefined && zeilen > erlaubt)
				gewachsen.push(`${pfad}: ${erlaubt} → ${zeilen}`);
		}
		expect(
			gewachsen,
			'Eine geduldete Datei ist weiter gewachsen. Die Ausnahme ist kein Freibrief.'
		).toEqual([]);
	});

	it('fuehrt nichts, was es nicht mehr gibt oder inzwischen sauber ist', () => {
		const veraltet = [];
		for (const [pfad, erlaubt] of Object.entries(BESTAND)) {
			const zeilen = aktuell.get(pfad);
			if (zeilen === undefined) veraltet.push(`${pfad} (gibt es nicht mehr)`);
			else if (zeilen <= 200) veraltet.push(`${pfad} (nur noch ${zeilen} Zeilen)`);
			else if (zeilen < erlaubt)
				veraltet.push(`${pfad} (geschrumpft auf ${zeilen}, Eintrag sagt ${erlaubt})`);
		}
		expect(
			veraltet,
			'Der Bestand ist ueberholt — austragen bzw. die Zahl nachziehen. Eine Liste, die ' +
				'Erledigtes weiterfuehrt, verliert ihre Aussage.'
		).toEqual([]);
	});
});
