import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

// Vokabular-Drift-Gate für vormerkungen.status (Vokabular-Sweep 19.08.2026).
//
// Die Statuswerte einer Vormerkung leben in der DB (chk_vormerkung_status in schema.sql)
// UND in der Oberfläche, die je Status etwas anderes zeigen muss. Bis zum 19.08.2026
// kannte die Oberfläche 'abholbereit' nicht: StudentVormerkungenCard.svelte zeigte JEDE
// Reservierung als „Wartet seit …" — auch eine, die zur Abholung bereitlag und deren
// Frist bald verfiel. Der Zustand war in DB und Backend real, nur nirgends sichtbar.
//
// Dieses Gate hält die beiden Ebenen deckungsgleich: Jeder Statuswert, den die Datenbank
// erlaubt (und den der Verfall-/Zuteilungs-Cron real setzt), muss in der Oberfläche
// vorkommen. Bauform wie etikettformate-konsistenz.test.js — frei, deterministisch,
// Millisekunden. Kommt ein Status hinzu (z. B. 'verlaengert'), schlägt es rot aus, bis
// die Oberfläche ihn behandelt.

const libDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = join(libDir, '..', '..', '..');

const schema = readFileSync(join(repoRoot, 'schema.sql'), 'utf8');
// Die UI-Dateien, die den Vormerkungs-Status anzeigen dürfen.
const uiQuellen = [readFileSync(join(libDir, 'StudentVormerkungenCard.svelte'), 'utf8')];

/** Liest die erlaubten Statuswerte aus chk_vormerkung_status. */
function statusWerteAusSchema() {
	const m = schema.match(/chk_vormerkung_status[\s\S]*?CHECK\s*\(status IN \(([^)]*)\)\)/);
	if (!m) throw new Error('chk_vormerkung_status nicht in schema.sql gefunden');
	return [...m[1].matchAll(/'([^']+)'/g)].map((x) => x[1]);
}

// 'wartend' ist der DOKUMENTIERTE Default: Die Oberfläche rendert ihn (und alles
// Unbekannte) als "Wartet seit …", ohne den Wert als Literal zu führen. Jeder Status
// DARÜBER HINAUS muss explizit behandelt werden — sonst zeigt die Oberfläche einen real
// gesetzten Zustand still als "wartend", genau der Bug, den dieses Gate schließt.
const DEFAULT_STATUS = 'wartend';

describe('Vormerkung-Status-Vokabular', () => {
	it('behandelt jeden von der DB erlaubten Nicht-Default-Status explizit in der Oberfläche', () => {
		const werte = statusWerteAusSchema();
		// Sanity-Floor: die Regex muss wirklich etwas gefunden haben.
		expect(werte.length).toBeGreaterThanOrEqual(2);
		expect(werte).toContain('abholbereit');
		expect(werte).toContain(DEFAULT_STATUS);

		const ui = uiQuellen.join('\n');
		const fehlend = werte
			.filter((w) => w !== DEFAULT_STATUS)
			.filter((w) => !ui.includes(`'${w}'`) && !ui.includes(`"${w}"`));
		expect(
			fehlend,
			`Diese von der DB erlaubten Vormerkungs-Status werden in keiner Anzeige-Datei explizit\n` +
				`behandelt — die Oberfläche würde sie still als "Wartet seit …" rendern:\n  ${fehlend.join('\n  ')}`
		).toEqual([]);
	});
});
