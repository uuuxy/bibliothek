import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, relPfad, sammleQuelldateien, kommentarZeilen } from './hygiene-quellen.js';

// Gate: Ein Kommentar der Oberfläche nennt kein Datum (CLAUDE.md, Abschnitt 4).
//
// Ein Kommentar sagt, warum der Code so ist; wann er so wurde und was vorher dastand, steht in
// der Commit-Nachricht. Am Datum lässt sich diese Vorgeschichte erkennen. Das Muster ist
// dasselbe wie stilDatum und stilQuelle in stil_ratsche_test.go: Ein Datum, das zu einer Quelle
// gehört („Erlass vom …", „abgerufen am …"), zählt nicht (docs/STIL.md, Abschnitt 8).
//
// Sieht nicht: ein Datum ohne Jahr („bis zum 16.09."), Vorgeschichte ohne Datum („bis dahin"),
// einen Zeilenkommentar hinter Code und die Tests. Ob ein Kommentar das Warum nennt, liest ein
// Mensch.
const DATUM = /\b\d{1,2}\.\d{1,2}\.20\d{2}\b|\b20\d{2}-\d{2}-\d{2}\b/;
const QUELLE =
	/(Erlass|Beschluss|Gesetz|Verordnung) vom \d{1,2}\.\d{1,2}\.20\d{2}|abgerufen am \d{1,2}\.\d{1,2}\.20\d{2}/g;

/** Trägt die Zeile ein Datum, das nicht zu einer Quelle gehört? @param {string} zeile */
const mitDatum = (zeile) => DATUM.test(zeile.replace(QUELLE, ''));

describe('Kommentare der Oberfläche nennen kein Datum', () => {
	/** @type {string[]} */
	const funde = [];
	// Gelesen je Form des Kommentars: Ein Scanner, der eine Form verliert, fände dort kein
	// Datum mehr und bliebe grün.
	/** @type {Record<string, number>} */
	const formen = { '//': 0, '/*': 0, '<!--': 0 };
	for (const datei of sammleQuelldateien(srcRoot)) {
		const zeilen = kommentarZeilen(readFileSync(datei, 'utf8'));
		for (const form of Object.keys(formen)) {
			formen[form] += zeilen.filter((z) => z.trimStart().startsWith(form)).length;
		}
		for (const z of zeilen.filter(mitDatum)) funde.push(`${relPfad(datei)}: ${z.trim()}`);
	}

	it('liest jede Form des Kommentars und erkennt ein Datum', () => {
		expect(formen['//']).toBeGreaterThan(1500);
		expect(formen['/*']).toBeGreaterThan(1000);
		expect(formen['<!--']).toBeGreaterThan(300);
		expect(mitDatum('// Bis zum 16.09.2026 stand hier ein Vollbild.')).toBe(true);
		expect(mitDatum(' * gemessen am 2026-09-04')).toBe(true);
		expect(mitDatum('// Nach dem Erlass vom 17.12.2014 gilt …')).toBe(false);
		expect(mitDatum('// Erlass vom 17.12.2014, umgesetzt am 02.09.2026')).toBe(true);
		expect(mitDatum('// Gültig bis 31.07.')).toBe(false);
	});

	it('kennt keine Kommentarzeile mit Datum', () => {
		expect(
			funde,
			'Datum in einem Kommentar. Der Kommentar nennt das Warum, die Vorgeschichte steht in der ' +
				'Commit-Nachricht (CLAUDE.md, Abschnitt 4).'
		).toEqual([]);
	});
});
