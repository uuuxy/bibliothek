import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { sammleQuelldateien, srcRoot, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Eine Zahl aus einer Variablen vor einem Hauptwort in fester Mehrzahl ergibt bei 1 „1 Exemplare".
// anzahlText aus utils/format.js wählt die Form nach der Zahl. Ausgenommen ist eine Stelle, die die
// Einzahl kurz davor selbst behandelt: ein Vergleich mit 1 oder `eines ?` in den 300 Zeichen davor.
//
// Sieht nicht: Hauptwörter außerhalb der Liste, Eigenschaftswörter vor dem Hauptwort („aktive
// Schüler") und Wörter, deren Einzahl gleich lautet („Titel", „Leser").
const MEHRZAHL = [
	'Ausleihen',
	'Barcodes',
	'Bestellungen',
	'Bücher',
	'Büchern',
	'Datensätze',
	'Einträge',
	'Einträgen',
	'Etiketten',
	'Exemplare',
	'Exemplaren',
	'Gebühren',
	'Kinder',
	'Kindern',
	'Klassen',
	'Minuten',
	'Paaren',
	'Positionen',
	'Schlagworte',
	'Schuljahren',
	'Stunden',
	'Tage',
	'Tagen',
	'Titeln',
	'Treffern',
	'Verweise',
	'Vorgänge',
	'Vormerkungen',
	'Wochen',
	'Zeilen'
];
const ZAHL_VOR_MEHRZAHL = new RegExp(
	`(?:\\$\\{|\\{(?![:/#@]))(?:[^{}]|\\{[^{}]*\\})*\\}(?:\\s|&nbsp;)+(?:weitere\\s+)?(?:${MEHRZAHL.join('|')})(?![\\wäöüß])`,
	'g'
);
const EINZAHL_BEHANDELT = /[!=]==?\s*1\b|>\s*1\b|\beines\b/;

/** Bewusste Stellen; jede braucht einen Grund, den anzahlText nicht ausdrücken kann. */
const AUSNAHMEN = [
	{
		datei: 'src/lib/components/bestellungen/Bestellbedarf.svelte',
		stelle: '${grundlage.kinder} Kinder',
		grund: 'Genitiv „von 250 der 300 Kinder": die Zahl aller Kinder der Schule.'
	}
];

/** @param {string} datei @param {string} quelle */
function funde(datei, quelle) {
	const q = ohneKommentare(quelle);
	const liste = [];
	for (const m of q.matchAll(ZAHL_VOR_MEHRZAHL)) {
		const davor = q.slice(Math.max(0, m.index - 300), m.index);
		const nah = q.slice(Math.max(0, m.index - 30), m.index + m[0].length);
		if (EINZAHL_BEHANDELT.test(davor)) continue;
		if (AUSNAHMEN.some((a) => a.datei === datei && nah.includes(a.stelle))) continue;
		liste.push(`${datei}: …${nah.trim()}`);
	}
	return liste;
}

describe('Zahl vor Hauptwort nur mit anzahlText', () => {
	const dateien = sammleQuelldateien(srcRoot).map((f) => ({
		datei: relPfad(f),
		quelle: readFileSync(f, 'utf8')
	}));

	it('durchsucht überhaupt Dateien und sieht die Stellen mit eigener Einzahl', () => {
		expect(dateien.length).toBeGreaterThan(300);
		const roh = dateien.flatMap((d) => [...ohneKommentare(d.quelle).matchAll(ZAHL_VOR_MEHRZAHL)]);
		expect(roh.length, 'der Detektor sieht keine einzige Zahl vor einem Hauptwort').toBeGreaterThan(
			5
		);
	});

	it('keine Quelldatei schreibt eine Zahl vor ein Hauptwort in fester Mehrzahl', () => {
		expect(
			dateien.flatMap((d) => funde(d.datei, d.quelle)),
			'feste Mehrzahl nach einer Zahl („1 Exemplare") — anzahlText aus utils/format.js nehmen'
		).toEqual([]);
	});

	it('jede Ausnahme existiert noch', () => {
		for (const a of AUSNAHMEN) {
			const quelle = readFileSync(`${srcRoot}/../${a.datei}`, 'utf8');
			expect(quelle.includes(a.stelle), `${a.datei}: Ausnahme ohne Fundstelle`).toBe(true);
		}
	});

	it('Selbstprobe: der Detektor fasst die Formen', () => {
		for (const form of [
			'melde(`${geloescht} Exemplare gelöscht`);',
			'<p>{liste.length} Bücher</p>',
			'options={STUNDEN.map((s) => ({ value: s, label: `${s} Stunden` }))}',
			'{buch.ohne_zeitpunkt} weitere Exemplare sind ausgesondert',
			'von {gesamt}&nbsp;Treffern'
		]) {
			expect(funde('probe', form), form).toHaveLength(1);
		}
		for (const form of [
			"anzahlText(n, 'Exemplar', 'Exemplare')",
			"n === 1 ? '1 Exemplar' : `${n} Exemplare`",
			"weg === 1\n\t\t? 'Ein Exemplar wird ausgesondert'\n\t\t: `${weg} Exemplare werden ausgesondert`",
			"eines ? 'Ein Exemplar ist verliehen:' : `${verliehen.length} Exemplare sind verliehen:`",
			'{n} Titel gefunden',
			'{tage} Tageszeiten'
		]) {
			expect(funde('probe', form), form).toEqual([]);
		}
	});
});
