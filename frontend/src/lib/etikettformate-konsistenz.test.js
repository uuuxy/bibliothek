import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import { ETIKETT_FORMATE, felderProBogen } from './etikettformate.js';
import { relPfad, sammleQuelldateien, srcRoot } from './hygiene-quellen.js';

// Die Etikettenraster stehen an mehreren Stellen — und das ist genau die Bauform, die in
// diesem Projekt schon zweimal auseinandergelaufen ist (zuletzt die dreifach kopierte
// Cover-Host-Allowlist, die eine Kopie kannte books.google nicht).
//
// Maßgeblich ist api/label_formats.go. Das Frontend führt seit dem 24.08.2026 GENAU EINE
// eigene Kopie davon: src/lib/etikettformate.js. Vorher waren es zwei (Liste in
// LabelLayoutOptionen.svelte, Stückzahlen in stores/labels.svelte.js), und mit den
// Schüler-Etiketten wäre die Ausweis-Werkzeugleiste die dritte geworden.
//
// Bewusst KEIN Umbau des Druck-Centers auf die Server-Liste: Die Bestätigungsseite des
// Lieferanten holt die Formate seit dem 06.08.2026 vom Server (etiketten_formate), im
// Druck-Center wäre derselbe Umbau eine Verhaltensänderung an einem täglich benutzten
// Bildschirm. Bis dahin hält dieser Test die Kopien deckungsgleich — er ist frei,
// deterministisch und läuft in Millisekunden, wie frontend-hygiene.test.js.

const libDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = join(libDir, '..', '..', '..');

const goQuelle = readFileSync(join(repoRoot, 'api', 'label_formats.go'), 'utf8');

/**
 * Liest die Formate aus api/label_formats.go: ID, Cols, Rows.
 * @returns {{id: string, cols: number, rows: number}[]}
 */
function formateAusGo() {
	const out = [];
	// Jeder Eintrag beginnt mit  "id": {  und enthält FormatID/Cols/Rows.
	const block = /"([a-z0-9_]+)":\s*\{([\s\S]*?)\n\t\},/g;
	let m;
	while ((m = block.exec(goQuelle)) !== null) {
		const [, id, rumpf] = m;
		const cols = rumpf.match(/Cols:\s*(\d+)/);
		const rows = rumpf.match(/Rows:\s*(\d+)/);
		if (cols && rows) out.push({ id, cols: Number(cols[1]), rows: Number(rows[1]) });
	}
	return out;
}

describe('Etikettenformate: Go-Liste und Oberfläche', () => {
	const goFormate = formateAusGo();

	it('die Go-Datei ist überhaupt lesbar', () => {
		// Ohne diese Prüfung wäre der Test bei einem geänderten Dateiaufbau still grün:
		// keine Formate gefunden = nichts zu vergleichen = alles in Ordnung.
		expect(goFormate.length).toBeGreaterThanOrEqual(3);
	});

	it('jedes Format aus dem Backend steht in der Frontend-Liste', () => {
		for (const f of goFormate) {
			expect(
				ETIKETT_FORMATE.some((e) => e.value === f.id),
				`Format ${f.id} fehlt in etikettformate.js`
			).toBe(true);
		}
	});

	it('die Frontend-Liste bietet kein Format an, das es im Backend nicht gibt', () => {
		const ids = goFormate.map((f) => f.id);
		for (const e of ETIKETT_FORMATE) {
			expect(ids, `etikettformate.js bietet ${e.value} an, das Backend kennt es nicht`).toContain(
				e.value
			);
		}
	});

	it('Spalten und Zeilen stimmen Feld für Feld mit dem Backend überein', () => {
		// Nicht nur die Gesamtzahl: 3×8 und 4×6 sind beide 24 und trotzdem zwei
		// verschiedene Bögen. Wer nur das Produkt prüft, lässt den Bogen durch, auf dem
		// jedes Etikett um eine Spalte verrutscht klebt.
		for (const f of goFormate) {
			const e = ETIKETT_FORMATE.find((x) => x.value === f.id);
			if (!e) {
				throw new Error(`Format ${f.id} fehlt in etikettformate.js`);
			}
			expect([e.spalten, e.zeilen], `Raster von ${f.id}`).toEqual([f.cols, f.rows]);
		}
	});

	it('felderProBogen liefert Spalten × Zeilen — auch für Unbekanntes eine bedienbare Zahl', () => {
		// Die Zahl ist das max des Startpositionsfeldes. Eine 0 machte es unbedienbar.
		for (const f of goFormate) {
			expect(felderProBogen(f.id), `felderProBogen('${f.id}')`).toBe(f.cols * f.rows);
		}
		expect(felderProBogen('gibt-es-nicht')).toBeGreaterThan(0);
	});

	it('keine zweite Formatliste im Frontend', () => {
		// Der eigentliche Zweck dieser Datei: Die Kopie soll EINE bleiben. Wer die IDs
		// woanders erneut hinschreibt, baut die Dopplung wieder auf, die es hier schon
		// zweimal gab.
		const treffer = sammleQuelldateien(srcRoot)
			.filter((pfad) => !pfad.endsWith('etikettformate.js') && !pfad.endsWith('.test.js'))
			.filter((pfad) => {
				const inhalt = readFileSync(pfad, 'utf8');
				return goFormate.filter((f) => inhalt.includes(`'${f.id}'`)).length >= 2;
			})
			.map((pfad) => relPfad(pfad));

		expect(treffer, 'diese Dateien führen eine eigene Formatliste').toEqual([]);
	});
});
