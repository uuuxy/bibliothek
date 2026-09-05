import { describe, it, expect } from 'vitest';
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';

// POST /api/action ist die BUCHUNGS-Tür der Theke: Ein gescannter B-Barcode ohne aktiven
// Schüler löst dort eine Rückgabe aus, ein S-Barcode wählt den Schüler. Zwei Suchfelder
// (Etiketten-Titelsuche, Vormerkungs-Schülersuche) schickten ihren Freitext an genau
// diese Tür und werteten nur `search_results` aus — ein Scan im Druck-Center hätte still
// ein Buch zurückgebucht (Suche-Inventur 03.09.2026). Suchen gehen an eine Such-Tür —
// welche, entscheidet der Bildschirm (zweite Ratsche unten).
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

// GET /api/search ist die SUCH-Tür der THEKE (perform_actions): Schüler-Kiosk, Titel und
// Scan-Auflösung in EINER Antwort, weil die Theke noch nicht weiß, was gescannt wurde. Die
// Etiketten-Titelsuche und die Vormerkungs-Schülersuche wissen es — sie lasen je eine Hälfte
// und warfen die andere weg (der Etikettenbildschirm: Schüler-Kiosk-Daten, die er nie zeigt)
// und hingen dabei am Theken-Recht: Ohne perform_actions meldeten sie „Suche nicht möglich".
// Seit 05.09.2026 fragen sie ihre eigene Tür: GET /api/buecher/titel/suche (view_books) und
// GET /api/schueler?q= (view_students). Wer ein neues Suchfeld an die Theken-Route hängt,
// wird hier rot — und entscheidet bewusst, ob es wirklich die Theke ist.
const SUCH_MUSTER = /['"`]\/api\/search(\?|['"`])/;
const SUCHE_ERLAUBT = new Set(['lib/stores/omnibox.svelte.js']);

describe('Theken-Suche /api/search', () => {
	it('wird nur von der Theken-Omnibox aufgerufen — andere Suchfelder haben ihre eigene Tür', () => {
		const treffer = dateien(WURZEL)
			.filter((p) => SUCH_MUSTER.test(readFileSync(p, 'utf8')))
			.map((p) => relative(WURZEL, p))
			.filter((p) => !SUCHE_ERLAUBT.has(p));
		expect(treffer, 'neue Aufrufer der Theken-Suche').toEqual([]);
	});
	it('Selbstprobe: der Detektor fasst die Aufrufformen', () => {
		for (const form of [
			'apiFetch(`/api/search?q=${q}`)',
			"apiFetch('/api/search')",
			'fetch("/api/search?q=x")'
		]) {
			expect(SUCH_MUSTER.test(form), form).toBe(true);
		}
		expect(SUCH_MUSTER.test('// Bewusst der OPAC und nicht /api/search: nur der OPAC')).toBe(false);
	});
});
