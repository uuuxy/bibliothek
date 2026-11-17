import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { sammleQuelldateien, srcRoot, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Fehlt ein Wert, steht ein Halbgeviertstrich („–“) wie in ui/NummerKopieren.svelte. Material 3
// nennt kein Zeichen dafür und rät vom Geviertstrich ab („Em dashes are best avoided in UX
// writing“, Content design, Grammar and punctuation); im Deutschen ist der Halbgeviertstrich der
// Gedankenstrich. Standen alle drei Striche nebeneinander, sah dieselbe Lücke in zwei Listen
// verschieden aus.
//
// Sieht nicht: einen Strich aus einer Variablen und einen Bindestrich als Ersatz außerhalb von
// `||`, `??` und Markup; dort ist er meist ein Trenner (`split('-')`).
const GEVIERT = /(?:\|\||\?\?|[?:,(]|return)\s*(['"`])\s*—\s*\1|>\s*—\s*</;
const BINDESTRICH = /(?:\|\||\?\?)\s*(['"`])-\1|>\s*-\s*</;

describe('Ein Zeichen für „kein Wert“', () => {
	const dateien = sammleQuelldateien(srcRoot);

	it('durchsucht überhaupt Dateien', () => {
		expect(dateien.length).toBeGreaterThan(300);
	});

	it('keine Quelldatei setzt einen Geviertstrich oder Bindestrich für einen fehlenden Wert', () => {
		const funde = dateien
			.filter((f) => {
				const q = ohneKommentare(readFileSync(f, 'utf8'));
				return GEVIERT.test(q) || BINDESTRICH.test(q);
			})
			.map(relPfad);
		expect(funde, 'fehlender Wert mit „—“ oder „-“ — den Halbgeviertstrich „–“ nehmen').toEqual([]);
	});

	it('Selbstprobe: die Detektoren fassen die Formen', () => {
		for (const form of [
			"{e.signatur || '—'}",
			"return formatDatum(iso) || '—';",
			'<span class="text-on-surface-variant">—</span>',
			"{log.ip_adresse || '-'}",
			"b.klasse || '-',"
		]) {
			expect(GEVIERT.test(form) || BINDESTRICH.test(form), form).toBe(true);
		}
		for (const form of [
			"{e.signatur || '–'}",
			"const [j, m, t] = iso.split('-');",
			"name.replace(/\\s+/g, '-')",
			"{ value: '-', label: 'Ohne Schulzweig' }",
			'`${titel} — ${klasse}`'
		]) {
			expect(GEVIERT.test(form) || BINDESTRICH.test(form), form).toBe(false);
		}
	});
});
