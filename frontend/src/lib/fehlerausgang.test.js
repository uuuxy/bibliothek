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
// Die sechzehn Reste stehen hier mit Begründung. Das ist ein Bestand bewusster
// Ausnahmen, KEINE Erlaubnis: Neues gehört behandelt, nicht eingetragen.
//
// Ausgetragen am 12.09.2026: der Boot-Restore des authStore. Seine Begründung
// („Login-Screen IST der richtige Rückfall") stimmte für den Netzwerkfehler, nicht mehr
// für die Antwort 503 — die seit dem 11.09.2026 heißt „die Sitzung ließ sich nicht
// prüfen", nicht „sie gilt nicht". Der Zweig behandelt sie jetzt.
/** @type {Record<string, [number, string]>} */
const BESTAND = {
	// — Vorschlagslisten: Ohne sie tippt man den Wert von Hand, nichts geht verloren.
	'src/inventur/lib/components/admin/BuchEingabefelder.svelte': [
		1,
		'Systematik-Vorschläge (Datalist)'
	],
	'src/inventur/lib/components/admin/ClassAssignPicker.svelte': [1, 'Klassennamen als Vorschlag'],
	'src/lib/GlobalLMFExtendWidget.svelte': [1, 'Klassennamen als Vorschlag'],
	'src/lib/components/students/lesergruppen.svelte.js': [
		1,
		'Lesergruppen als Vorschlag im Anlegen-Dialog (am 12.09.2026 aus StudentDirectory ausgelagert)'
	],
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
	],

	// — Seit dem 12.09.2026 sieht der Detektor zwei weitere Formen (UND-Kette und
	//   `? … : neutraler Wert`). Was er dabei fand, ist behoben; diese drei bleiben, weil
	//   sie den Fehler BEHANDELN — sie sehen nur aus wie die Form.
	'src/lib/Monitor.svelte': [
		1,
		'Flur-Monitor: null heißt „nichts Neues" — der Takt behält die Folien und versucht es wieder (monitorTakt.svelte.js)'
	],
	'src/lib/useBookAkte.svelte.js': [
		1,
		'leerer Kopf MIT Meldung: die Zeile darunter setzt `kopfFehler`, und die Akte zeigt „Titel nicht geladen" statt „Buch nicht gefunden" (12.09.2026)'
	],
	'src/lib/components/students/zusammenfuehrenSuche.svelte.js': [
		1,
		'leere Trefferliste MIT Meldung: die Zeile darunter setzt `fehler` aus extractApiError'
	],
	'src/lib/useStudentProfile.svelte.js': [
		4,
		'bewusst zugewiesen (06.09.2026): Scheitert eine der vier Anfragen, dürfen NICHT die Werte des vorher geöffneten Schülers stehen bleiben — die Gebühren-Karte schreibt auf die Fall-ID der Zeile'
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

		// Seit dem 12.09.2026: die UND-Kette. Die Wettlauf-Form der Suchfelder
		// (`if (res.ok && nr === ladeNr)`) rutschte vorher durch — und das ist die Stelle,
		// an der der Sweep angefangen hat.
		const undKette = `
			async function f() {
				const res = await fetch('/x');
				if (res.ok && nr === ladeNr) { daten = await res.json(); }
				weiter(daten);
			}`;
		expect(findeVerschluckteFehlantworten('probe.js', undKette)).toHaveLength(1);

		// Und der Dreisatz mit neutralem Ersatzwert: Der Fehlerzweig existiert, setzt aber
		// „nichts" — auf dem Bildschirm nicht von „nichts gefunden" zu unterscheiden.
		const neutralerErsatz = `
			async function f() {
				const res = await fetch('/x');
				liste = res.ok ? await res.json() : [];
			}`;
		expect(findeVerschluckteFehlantworten('probe.js', neutralerErsatz)).toHaveLength(1);

		// Ein Ersatzwert, der etwas AUSSAGT, ist Fehlerbehandlung — kein Fund. Sonst
		// stünde die halbe Anwendung im Bestand und die Ratsche wäre wertlos.
		const sprechenderErsatz = `
			async function f() {
				const res = await fetch('/x');
				melde(res.ok ? 'gespeichert' : 'fehlgeschlagen');
				zustand = res.ok ? 'saved' : 'error';
			}`;
		expect(findeVerschluckteFehlantworten('probe.js', sprechenderErsatz)).toHaveLength(0);

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
