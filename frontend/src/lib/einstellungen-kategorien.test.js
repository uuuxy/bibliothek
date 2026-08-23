import { describe, it, expect } from 'vitest';
import { readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import { srcRoot } from './hygiene-quellen.js';
import { KATEGORIEN } from './components/settings/kategorien.js';

// Zwei Struktur-Invarianten der Einstellungsseite. Beide sind Antworten auf Fragen aus
// dem Prüfraster vom 22.08.2026 („Konvention statt Regel", „zwei Wahrheitsquellen") —
// und der Punkt ist, dass sie ab jetzt von selbst gestellt werden, statt beim nächsten
// Durchgang von Hand.
//
// Warum es sie braucht: Seit dem 23.08.2026 beruht die Zusicherung „wer eine Kategorie
// speichert, ändert nichts in den anderen" darauf, dass jede Kategorie NUR ihre eigenen
// Felder in den Patch legt. Das ist im Code eine Verabredung zwischen sieben Dateien —
// nichts hindert eine davon, morgen ein fremdes Feld mitzuschicken, und das Backend
// würde es pflichtschuldig schreiben. Der e2e-Test daneben prüft GENAU EIN Paar
// (Datenschutz gegen den Rest); dieser hier prüft alle gegen alle.

const KATEGORIE_DIR = join(srcRoot, 'lib/components/settings/kategorien');
const SHELL = join(srcRoot, 'lib/SystemSettings.svelte');

// Das Vokabular der Einstellungs-Schlüssel (Rumpf von PUT /api/einstellungen,
// repository/system_settings_patch.go). Ein neuer Schlüssel zwingt hier zu einer
// Entscheidung, in welche Kategorie er gehört — statt irgendwo zu landen.
const ALLE_SCHLUESSEL = [
	'schule_name',
	'schule_strasse',
	'schule_plz',
	'schule_ort',
	'etikett_eigentumsvermerk',
	'frist_buch_tage',
	'frist_medien_tage',
	'max_ausleihen_schueler',
	'lmf_stichtag',
	'ferien_leseclub_aktiv',
	'ferien_leseclub_zieldatum',
	'max_overdue_days',
	'max_overdue_items',
	'bestellbedarf_warnung_aktiv',
	'bestellbedarf_schwelle',
	'preise_erfassen',
	'lesehistorie_tage',
	'lesehistorie_lernmittel_tage',
	'anliegen_tage',
	'theke_leeren_minuten',
	'sperre_minuten',
	'oeffentliche_adresse',
	'alarm_empfaenger'
];

/**
 * Der Aufruf `speichereKategorie({ … })` einer Kategorie — und nur er.
 *
 * Nicht die ganze Datei durchsuchen: Jede Kategorie LIEST ihre Felder auch
 * (`start.theke_leeren_minuten`), und ein Detektor, der beides in einen Topf wirft,
 * meldet ein Feld weiterhin als versorgt, nachdem der Schreibweg entfernt wurde. Genau
 * das ist beim Rückbau-Versuch passiert: Die Probe blieb grün, obwohl der Schlüssel aus
 * dem Patch verschwunden war. Ein Gate, das seine eigene Aussage nicht hält, ist
 * schlimmer als keins.
 *
 * @param {string} datei
 * @returns {string} der Text zwischen den Klammern des Aufrufs (leer, wenn es keinen gibt)
 */
function speicherAufruf(datei) {
	const quelle = readFileSync(join(KATEGORIE_DIR, datei), 'utf8');
	const start = quelle.indexOf('speichereKategorie(');
	if (start === -1) return '';
	let tiefe = 0;
	for (let i = start; i < quelle.length; i++) {
		if (quelle[i] === '(') tiefe++;
		else if (quelle[i] === ')' && --tiefe === 0) return quelle.slice(start, i);
	}
	return quelle.slice(start);
}

/** @param {string} datei @returns {string[]} die Schlüssel, die diese Kategorie SCHREIBT */
function schluesselIn(datei) {
	const aufruf = speicherAufruf(datei);
	return ALLE_SCHLUESSEL.filter((k) => new RegExp(`\\b${k}\\b`).test(aufruf));
}

describe('Einstellungs-Kategorien', () => {
	it('lässt keine zwei Kategorien dasselbe Feld anfassen', () => {
		/** @type {Map<string, string[]>} */
		const besitzer = new Map();
		for (const datei of readdirSync(KATEGORIE_DIR).filter((f) => f.endsWith('.svelte'))) {
			for (const k of schluesselIn(datei)) {
				besitzer.set(k, [...(besitzer.get(k) ?? []), datei]);
			}
		}

		const doppelt = [...besitzer.entries()].filter(([, dateien]) => dateien.length > 1);
		expect(
			doppelt,
			`Diese Einstellungen werden von mehr als einer Kategorie geschrieben:\n` +
				doppelt.map(([k, d]) => `  ${k}: ${d.join(', ')}`).join('\n') +
				`\nDamit überschreibt ein Speichern in der einen still, was in der anderen steht —\n` +
				`genau der Zustand, den der Umbau vom 23.08.2026 beendet hat.`
		).toEqual([]);
	});

	it('lässt kein Einstellungsfeld ohne Kategorie zurück', () => {
		const vergeben = new Set(
			readdirSync(KATEGORIE_DIR)
				.filter((f) => f.endsWith('.svelte'))
				.flatMap(schluesselIn)
		);
		const heimatlos = ALLE_SCHLUESSEL.filter((k) => !vergeben.has(k));
		expect(
			heimatlos,
			`Diese Einstellungen kann niemand mehr ändern — sie stehen in keiner Kategorie:\n  ` +
				`${heimatlos.join(', ')}\nEin Feld, das die API kennt und die Oberfläche nicht, ` +
				`ist eine tote Einstellung (Bugklasse „Feature hängt an ungesetzter Einstellung").`
		).toEqual([]);
	});

	it('nennt jede Kategorie in der Liste genauso wie auf ihrer Detailfläche', () => {
		// Der Titel steht zweimal: einmal in der Liste links (kategorien.js), einmal als
		// Überschrift der Detailfläche — und aus ihm entsteht die Beschriftung des
		// Speichern-Knopfes („<Name> speichern"). Driften sie auseinander, heißt die
		// Kategorie links anders als der Knopf rechts.
		const quellen = [
			readFileSync(SHELL, 'utf8'),
			...readdirSync(KATEGORIE_DIR)
				.filter((f) => f.endsWith('.svelte'))
				.map((f) => readFileSync(join(KATEGORIE_DIR, f), 'utf8'))
		].join('\n');

		const ueberschriften = new Set(
			[...quellen.matchAll(/titel="([^"]+)"/g)].map((m) => m[1].replace(/\s+/g, ' '))
		);
		const fehlend = KATEGORIEN.map((k) => k.titel).filter((t) => !ueberschriften.has(t));

		expect(
			fehlend,
			`Diese Kategorien heißen auf ihrer Detailfläche anders als in der Liste ` +
				`(oder haben gar keine Überschrift):\n  ${fehlend.join(', ')}\n` +
				`Gefunden wurden: ${[...ueberschriften].join(' · ')}`
		).toEqual([]);
	});
});
