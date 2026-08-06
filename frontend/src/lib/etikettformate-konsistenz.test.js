import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

// Die Etikettenraster stehen an mehreren Stellen — und das ist genau die Bauform, die in
// diesem Projekt schon zweimal auseinandergelaufen ist (zuletzt die dreifach kopierte
// Cover-Host-Allowlist, die eine Kopie kannte books.google nicht).
//
// Maßgeblich ist api/label_formats.go. Die Oberfläche des Druck-Centers führt ihre eigene
// Liste (LabelLayoutOptionen.svelte) und ihre eigenen Stückzahlen (stores/labels.svelte.js).
// Beide werden hier gegen die Go-Datei gehalten.
//
// Bewusst KEIN Umbau des Druck-Centers auf die Server-Liste: Die Bestätigungsseite des
// Lieferanten holt die Formate seit dem 06.08.2026 vom Server (etiketten_formate), im
// Druck-Center wäre derselbe Umbau eine Verhaltensänderung an einem täglich benutzten
// Bildschirm. Bis dahin hält dieser Test die Kopien deckungsgleich — er ist frei,
// deterministisch und läuft in Millisekunden, wie frontend-hygiene.test.js.

const libDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = join(libDir, '..', '..', '..');

const goQuelle = readFileSync(join(repoRoot, 'api', 'label_formats.go'), 'utf8');
const optionenQuelle = readFileSync(join(libDir, 'components', 'labels', 'LabelLayoutOptionen.svelte'), 'utf8');
const storeQuelle = readFileSync(join(libDir, 'stores', 'labels.svelte.js'), 'utf8');

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

	it('jedes Format aus dem Backend steht in der Auswahl des Druck-Centers', () => {
		for (const f of goFormate) {
			expect(
				optionenQuelle.includes(`'${f.id}'`),
				`Format ${f.id} fehlt in LabelLayoutOptionen.svelte`
			).toBe(true);
		}
	});

	it('die Auswahl bietet kein Format an, das es im Backend nicht gibt', () => {
		const ids = goFormate.map((f) => f.id);
		const angeboten = [...optionenQuelle.matchAll(/value:\s*'([a-z0-9_]+)'/g)].map((m) => m[1]);
		for (const id of angeboten) {
			// BARCODE_AUSGABE steht in derselben Datei; deren Werte sind keine Raster.
			if (id === 'code39' || id === 'qr') continue;
			expect(ids, `LabelLayoutOptionen.svelte bietet ${id} an, das Backend kennt es nicht`).toContain(
				id
			);
		}
	});

	it('die Stückzahlen im Store stimmen mit Spalten × Zeilen überein', () => {
		// labels.svelte.js rechnet die Etiketten je Bogen selbst aus (für die Vorschau).
		// Eine Zahl, die hier abweicht, zeigt einen Bogen mit der falschen Kachelzahl.
		for (const f of goFormate) {
			const treffer = storeQuelle.match(
				new RegExp(`formatId === '${f.id}'\\)\\s*return\\s+(\\d+)`)
			);
			if (!treffer) continue; // nicht jedes Format muss im Store gesondert stehen
			expect(
				Number(treffer[1]),
				`labels.svelte.js sagt ${treffer[1]} Etiketten für ${f.id}, das Raster ist ${f.cols}×${f.rows}`
			).toBe(f.cols * f.rows);
		}
	});
});
