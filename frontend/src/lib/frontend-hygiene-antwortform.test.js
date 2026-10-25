import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import {
	srcRoot,
	sammleQuelldateien,
	sammleTestdateien,
	relPfad,
	ohneKommentare
} from './hygiene-quellen.js';

// Jede Tür antwortet mit dem Wert selbst, ohne Hülle {"data": …}; am Server hält das
// antwort_huelle_ratsche_test.go. Eine Lesestelle, die .data einer Antwort liest, bekommt
// undefined und zeigt eine leere Liste ohne Fehler: So zeigte die Leserakte keinen Bescheid. Ein
// Test, der die Antwort als { data: … } nachstellt, prüft nur die Form, die er selbst annimmt.
//
// BLINDHEIT: Liest Muster. Eine Antwort unter einem anderen Namen sieht der Test nur bei den Namen
// json, body, daten, antwort und j, einen Zugriff über eckige Klammern (['data']) gar nicht. Die
// eigene Hülle des Inventur-Clients ({ ok, status, data }, inventurApi.js) ist keine Antwort des
// Servers und bleibt erlaubt.
const LESESTELLE =
	/\.json\(\)\)\s*\??\.data\b|\b(?:json|body|daten|antwort|j)\??\.data\b|\{\s*data\s*\}\s*=\s*await\b/g;
const ATTRAPPE = /\{\s*data\s*:/g;

/**
 * @param {string[]} dateien
 * @param {RegExp} muster
 */
function funde(dateien, muster) {
	/** @type {string[]} */
	const aus = [];
	for (const f of dateien) {
		const treffer = ohneKommentare(readFileSync(f, 'utf8')).match(muster) ?? [];
		if (treffer.length > 0) aus.push(`${relPfad(f)}: ${[...new Set(treffer)].join(', ')}`);
	}
	return aus;
}

describe('Antwortform: der Wert selbst, ohne Hülle { data }', () => {
	it('erkennt die Formen, die es finden soll', () => {
		const trifft = (/** @type {RegExp} */ muster, /** @type {string} */ text) =>
			(text.match(muster) ?? []).length;
		expect(trifft(LESESTELLE, 'return (await res.json()).data ?? [];')).toBe(1);
		expect(trifft(LESESTELLE, 'gruppen = json.data || []; liste = daten?.data;')).toBe(2);
		expect(trifft(LESESTELLE, 'const { data } = await res.json();')).toBe(1);
		expect(trifft(LESESTELLE, "listeOderVermerk(res, 'Bescheide', (j) => j.data)")).toBe(1);
		expect(trifft(LESESTELLE, 'JSON.parse(e.data); r.data.barcode_id; this.data')).toBe(0);
		expect(trifft(ATTRAPPE, 'antwort(200, { data: [] }); json: async () => ({data: liste})')).toBe(
			2
		);
		expect(trifft(ATTRAPPE, '({ ok: false, status: 409, data: { status: "x" } })')).toBe(0);
	});

	it('liest an keiner Antwort .data', () => {
		const dateien = sammleQuelldateien(srcRoot);
		expect(dateien.length, 'Der Sammler findet kaum Quelldateien').toBeGreaterThan(200);
		expect(
			funde(dateien, LESESTELLE),
			'Die Tür schickt den Wert selbst; .data ist dort undefined.'
		).toEqual([]);
	});

	it('stellt keine Antwort als { data: … } nach', () => {
		// Ohne diese Datei: Ihre Beispiele oben tragen die Form, die sie findet.
		const dateien = sammleTestdateien(srcRoot).filter(
			(f) => !relPfad(f).endsWith('frontend-hygiene-antwortform.test.js')
		);
		expect(dateien.length, 'Der Sammler findet kaum Testdateien').toBeGreaterThan(200);
		expect(
			funde(dateien, ATTRAPPE),
			'Eine nachgestellte Antwort hat die Form der Tür: den Wert selbst.'
		).toEqual([]);
	});
});
