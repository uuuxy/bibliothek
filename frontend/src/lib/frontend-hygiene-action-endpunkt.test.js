import { describe, it, expect } from 'vitest';
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';

// POST /api/action ist die BUCHUNGS-Tür der Theke: Ein gescannter B-Barcode ohne aktiven
// Schüler löst dort eine Rückgabe aus, ein S-Barcode wählt den Schüler. Zwei Suchfelder
// (Etiketten-Titelsuche, Vormerkungs-Schülersuche) schickten ihren Freitext an genau
// diese Tür und werteten nur `search_results` aus — ein Scan im Druck-Center hätte still
// ein Buch zurückgebucht (Suche-Inventur 03.09.2026). Suchen gehen an GET /api/search.
const ERLAUBT = new Set(['lib/stores/omnibox.svelte.js']);
const WURZEL = join(import.meta.dirname, '..');

function dateien(dir, out = []) {
	for (const name of readdirSync(dir)) {
		const p = join(dir, name);
		if (statSync(p).isDirectory()) dateien(p, out);
		else if (/\.(svelte|js)$/.test(name) && !/\.test\.js$/.test(name)) out.push(p);
	}
	return out;
}

const MUSTER = /['"`]\/api\/action['"`]/;

describe('Buchungs-Tür /api/action', () => {
	it('wird nur von der Theken-Omnibox aufgerufen — Suchfelder nutzen /api/search', () => {
		const treffer = dateien(WURZEL)
			.filter((p) => MUSTER.test(readFileSync(p, 'utf8')))
			.map((p) => relative(WURZEL, p))
			.filter((p) => !ERLAUBT.has(p));
		expect(treffer, 'neue Aufrufer der Buchungs-Tür').toEqual([]);
	});
	it('Selbstprobe: der Detektor fasst die Aufrufformen', () => {
		for (const form of [
			"apiClient.post('/api/action', {",
			'apiFetch("/api/action")',
			'fetch(`/api/action`)'
		]) {
			expect(MUSTER.test(form), form).toBe(true);
		}
		expect(MUSTER.test("post('/api/action/batch'")).toBe(false);
	});
});
