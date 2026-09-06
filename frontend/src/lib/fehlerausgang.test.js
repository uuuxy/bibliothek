import { describe, it, expect } from 'vitest';
import { sammleQuelldateien, srcRoot, relPfad } from './hygiene-quellen.js';
import { findeVerschluckteFehlantworten } from './fehlerausgangScanner.js';

// Ratsche zum Sweep „verschluckte Fehlantwort" (06.09.2026, docs/sweeps.md).
//
// Gesucht wird `if (res.ok) { … }` ohne `else`, dessen Zweig nicht mit return/throw
// endet: Scheitert der Abruf, läuft der Code weiter, als wäre nichts gewesen, und der
// Zustand bleibt auf dem Stand VOR dem Abruf. Der Sweep fand 26 solche Stellen, neun
// wurden behoben — darunter zwei, die einem Menschen geschadet hätten:
//
//   - Die Theke zeigte unter dem neuen Suchtext die Treffer des alten; ein Klick buchte
//     auf den falschen Schüler (22173001).
//   - Der Ausweis-Designer schrieb nach einem fehlgeschlagenen Laden das VORGABE-Design
//     an alle Arbeitsplätze — ohne Klick, nur durch Öffnen des Bildschirms (c2be1069).
//
// Die siebzehn Reste stehen hier mit Begründung. Das ist ein Bestand bewusster
// Ausnahmen, KEINE Erlaubnis: Neues gehört behandelt, nicht eingetragen.
/** @type {Record<string, [number, string]>} */
const BESTAND = {
	// — Vorschlagslisten: Ohne sie tippt man den Wert von Hand, nichts geht verloren.
	'src/inventur/lib/components/admin/BuchEingabefelder.svelte': [
		1,
		'Systematik-Vorschläge (Datalist)'
	],
	'src/inventur/lib/components/admin/ClassAssignPicker.svelte': [1, 'Klassennamen als Vorschlag'],
	'src/lib/GlobalLMFExtendWidget.svelte': [1, 'Klassennamen als Vorschlag'],
	'src/lib/StudentDirectory.svelte': [1, 'Lesergruppen als Filter-Vorschlag'],
	'src/lib/components/students/KlassenDruckEinstieg.svelte': [
		1,
		'Klassenliste; der Weg über die Schülerdatei bleibt'
	],
	'src/lib/stores/labels.svelte.js': [
		1,
		'Klassengruppen als Filter (die Titelsuche daneben ist behoben)'
	],
	'src/lib/useInventurSession.svelte.js': [2, 'Signaturen- und Fächerliste als Filter'],

	// — Zähler an Menüpunkten: Eine Pille, die fehlt, hält niemanden auf.
	'src/lib/stores/uiStore.svelte.js': [3, 'Zählerpillen (Reservierungen, Anliegen, Etiketten)'],

	// — Bewusste Entscheidungen mit eigener Begründung im Code.
	'src/lib/stores/authStore.svelte.js': [1, 'Boot-Restore: Login-Screen IST der richtige Rückfall'],
	'src/lib/stores/backupStatus.svelte.js': [1, 'kein Abzeichen statt falscher Entwarnung'],
	'src/lib/components/portal/klassensatzReservierung.svelte.js': [
		1,
		'Zusatzinfo; das Portal bleibt ohne sie benutzbar'
	],
	'src/lib/lmfplanPlaner.svelte.js': [
		1,
		'kein Response, sondern ein Ergebnis-Objekt — der Toast steht eine Zeile davor'
	],

	// — Ausweis-Design in den Druckwegen: Fällt der Abruf aus, greifen die Vorgabewerte,
	//   und genau die zeigt die Vorschau auf dem Bildschirm, bevor irgendetwas gedruckt
	//   wird. Der gefährliche Zwilling war der Designer selbst (Auto-Speicherung) — der
	//   ist behoben; diese beiden schreiben nichts zurück.
	'src/lib/StudentPrintCard.svelte': [1, 'Ausweis-Design; Vorschau zeigt, was gedruckt wird'],
	'src/lib/components/students/StudentBatchPrint.svelte': [
		1,
		'Ausweis-Design; Vorschau zeigt, was gedruckt wird'
	]
};

describe('Sweep: verschluckte Fehlantworten', () => {
	// Selbstprobe zuerst: Ein Detektor, der nichts fasst, meldet ewig „alles gut" —
	// und einer, der zu viel fasst, macht die Ratsche unbrauchbar.
	it('fasst die verbotene Form und lässt die sichere in Ruhe', () => {
		const verschluckt = `
			async function f() {
				const res = await fetch('/x');
				if (res.ok) { daten = await res.json(); }
				weiter(daten);
			}`;
		expect(findeVerschluckteFehlantworten('probe.js', verschluckt)).toHaveLength(1);

		const frueheRueckkehr = `
			async function f() {
				const res = await fetch('/x');
				if (res.ok) { return await res.json(); }
				melde('fehlgeschlagen');
				return [];
			}`;
		expect(findeVerschluckteFehlantworten('probe.js', frueheRueckkehr)).toHaveLength(0);

		const mitElse = `
			async function f() {
				const res = await fetch('/x');
				if (res.ok) { daten = await res.json(); } else { melde('fehlgeschlagen'); }
			}`;
		expect(findeVerschluckteFehlantworten('probe.js', mitElse)).toHaveLength(0);

		const inSvelte = `<script>
			async function f() {
				const res = await fetch('/x');
				if (res.ok) { daten = await res.json(); }
				weiter(daten);
			}
		</script>`;
		expect(findeVerschluckteFehlantworten('probe.svelte', inSvelte)).toHaveLength(1);
	});

	it('hält den Bestand — neue Fundstellen sind rot, behobene tragen sich aus', () => {
		/** @type {Record<string, number>} */
		const gefunden = {};
		let dateien = 0;
		for (const datei of sammleQuelldateien(srcRoot)) {
			dateien++;
			const treffer = findeVerschluckteFehlantworten(datei);
			if (treffer.length) gefunden[relPfad(datei)] = treffer.length;
		}
		expect(dateien, 'der Sammler greift ins Leere — dieses Gate wäre still grün').toBeGreaterThan(
			200
		);

		/** @type {string[]} */
		const neu = [];
		/** @type {string[]} */
		const ueberholt = [];
		for (const [datei, anzahl] of Object.entries(gefunden)) {
			const eintrag = BESTAND[datei];
			if (!eintrag) neu.push(`${datei} (${anzahl})`);
			else if (anzahl > eintrag[0]) neu.push(`${datei} (${anzahl}, Bestand sagt ${eintrag[0]})`);
		}
		for (const [datei, eintrag] of Object.entries(BESTAND)) {
			const anzahl = gefunden[datei] ?? 0;
			if (anzahl < eintrag[0]) ueberholt.push(`${datei} (${anzahl}, Eintrag sagt ${eintrag[0]})`);
		}

		expect(
			neu,
			'Verschluckte Fehlantwort: Der Zweig läuft weiter, als wäre der Abruf gelungen — ' +
				'der Zustand bleibt auf dem Stand davor. Behandeln (else-Zweig, Zustand zurücksetzen, ' +
				'Meldung) oder hier mit Begründung eintragen.'
		).toEqual([]);
		expect(
			ueberholt,
			'Der Bestand ist überholt — Zahl nachziehen bzw. Eintrag austragen. Eine Liste, die ' +
				'Erledigtes weiterführt, verliert ihre Aussage.'
		).toEqual([]);
	});
});
