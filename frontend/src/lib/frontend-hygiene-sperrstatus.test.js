import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { relative } from 'node:path';
import { sammleQuelldateien, srcRoot, repoFrontend } from './hygiene-quellen.js';

// Ratsche auf den Sperrvermerk: Ob ein Leser gesperrt ist, sagt die Anzeige nur über
// sperrStatus.ausleiheGesperrt, dieselbe Regel wie die Theke (beide Sperren, ein Kollege nie).
// Lasen Abgängerliste, Ehemalige und Zusammenführen nur ist_gesperrt, stand ein von Hand
// Gesperrter ohne Vermerk da. Erlaubt ist außer dort nur das Übernehmen eines Feldes aus der
// Antwort des Servers (akte.ist_gesperrt = neu.ist_gesperrt).
const QUELLE = 'src/lib/components/leser/sperrStatus.js';
const FELD = /\.(ist_gesperrt|is_manually_blocked)\b/;
const UEBERNAHME = /^\s*[\w.]+\.(ist_gesperrt|is_manually_blocked)\s*=\s*[\w.?]+\.\1;\s*$/;

/** @param {string} text @returns {string[]} die Zeilen, die ein Sperrfeld selbst lesen */
const lesendeZeilen = (text) =>
	text.split('\n').filter((zeile) => FELD.test(zeile) && !UEBERNAHME.test(zeile));

const dateien = sammleQuelldateien(srcRoot).map((absolut) => ({
	pfad: relative(repoFrontend, absolut),
	text: readFileSync(absolut, 'utf8')
}));

describe('Sperrvermerk: eine Regel für die Anzeige', () => {
	it('erkennt jede Form, die ein Sperrfeld selbst liest', () => {
		expect(lesendeZeilen('{#if s.ist_gesperrt}')).toHaveLength(1);
		expect(lesendeZeilen('{#if !kandidat?.is_manually_blocked}')).toHaveLength(1);
		expect(lesendeZeilen('const x = a && (leser.ist_gesperrt || b);')).toHaveLength(1);
		expect(lesendeZeilen('\t\takte.ist_gesperrt = updated.ist_gesperrt;')).toHaveLength(0);
		expect(dateien.find((d) => d.pfad === QUELLE)?.text).toMatch(FELD);
	});

	it('liest die Sperrfelder nur in sperrStatus.js', () => {
		const funde = dateien
			.filter((d) => d.pfad !== QUELLE)
			.flatMap((d) => lesendeZeilen(d.text).map((zeile) => `${d.pfad}: ${zeile.trim()}`));
		expect(funde).toEqual([]);
	});
});
