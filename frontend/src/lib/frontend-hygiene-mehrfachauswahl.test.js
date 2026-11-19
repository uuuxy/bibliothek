import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad, ohneKommentare, koepfe } from './hygiene-quellen.js';

// Ratsche: Jede Liste mit Kästchen zum Auswählen kennt Umschalt+Klick und Strg/Cmd+A.
//
// M3, Desktop: „Select a range: Shift + click“, „Select all: Control + A, or Cmd + A (Mac)“.
// Die Regel steht einmal in actions/mehrfachauswahl.js; eine Liste trägt use:mehrfachauswahl,
// ihr Kästchen für alle im Tabellenkopf data-auswahl-alle. Erkannt wird eine Liste an einem
// <Kaestchen> in einem {#each}.
//
// M3, Checkbox: „If some, but not all, child checkboxes are checked, the parent checkbox becomes
// an indeterminate checkbox“: Das Kästchen für alle trägt indeterminate. Die Leiste zur Auswahl
// sagt „ausgewählt“ wie das Kästchen „auswählen“ (Android: „Alle auswählen“, „%d ausgewählt“);
// „markiert“ heißt im Programm ein dauerhaftes Merkmal (als Filter, als gefunden).
//
// Sieht nicht: Zeilen aus einem eigenen Bauteil (BestandstabelleZeile, ExemplarKarte), deren
// Schleife im Elternteil steht, und Kästchen ohne ui/Kaestchen.

/** Kästchen in Schleifen, die keine Auswahl sind, mit Grund. */
const KEINE_AUSWAHL = {
	// Das Kästchen meldet ein Exemplar als gefunden; Umschalt+Klick löste das für viele aus.
	'src/lib/components/inventur/FehlbestandBericht.svelte': 'Aktion je Zeile',
	// Optionen eines Formulars (welche Mails über den zweiten Absender gehen); Strg+A wählte
	// sonst auf der Einstellungsseite alle Mailarten.
	'src/lib/components/admin/MailZweiterAbsender.svelte': 'Optionen eines Formulars'
};

/** Steht ein <Kaestchen> innerhalb eines {#each}-Blocks? @param {string} quelle */
function kaestchenInSchleife(quelle) {
	let tiefe = 0;
	for (const m of quelle.matchAll(/\{#each\b|\{\/each\}|<Kaestchen\b/g)) {
		if (m[0] === '{#each') tiefe++;
		else if (m[0] === '{/each}') tiefe--;
		else if (tiefe > 0) return true;
	}
	return false;
}

/** Kästchen im Tabellenkopf ohne data-auswahl-alle. @param {string} quelle */
function kopfOhneMarke(quelle) {
	return [...quelle.matchAll(/<thead[\s\S]*?<\/thead>/g)].some((kopf) =>
		koepfe(kopf[0], 'Kaestchen').some((k) => !/data-auswahl-alle/.test(k))
	);
}

/** Kästchen für alle ohne den Strich bei teilweiser Auswahl. @param {string} quelle */
const alleOhneStrich = (quelle) =>
	koepfe(quelle, 'Kaestchen').some(
		(k) => /data-auswahl-alle/.test(k) && !/\bindeterminate\b/.test(k)
	);

/** Der Kopf ohne Ausdrücke: Dort stehen Namen aus dem Code, kein Text. @param {string} kopf */
const sichtbarerText = (kopf) => kopf.replace(/\{[^{}]*\}/g, '');

function dateien() {
	return sammleQuelldateien(srcRoot)
		.filter((f) => f.endsWith('.svelte'))
		.map((f) => ({ pfad: relPfad(f), quelle: ohneKommentare(readFileSync(f, 'utf8')) }));
}

describe('Mehrfachauswahl in jeder Liste mit Kästchen (M3, Desktop)', () => {
	it('der Detektor erkennt Kästchen in Schleifen und unmarkierte Kopf-Kästchen', () => {
		expect(kaestchenInSchleife('{#each a as x}<Kaestchen checked />{/each}')).toBe(true);
		expect(kaestchenInSchleife('{#each a as x}{x}{/each}<Kaestchen etikett="Alle" />')).toBe(false);
		expect(kopfOhneMarke('<thead><tr><th><Kaestchen checked={a} /></th></tr></thead>')).toBe(true);
		expect(kopfOhneMarke('<thead><Kaestchen\n\tdata-auswahl-alle\n\tchecked={a}\n/></thead>')).toBe(
			false
		);
		// Ein „>“ in einem Ausdruck beendet den Kopf nicht.
		expect(
			kopfOhneMarke('<thead><Kaestchen indeterminate={n > 0} onchange={() => a()} /></thead>')
		).toBe(true);
		expect(alleOhneStrich('<Kaestchen data-auswahl-alle checked={a} onchange={() => b()} />')).toBe(
			true
		);
		expect(alleOhneStrich('<Kaestchen data-auswahl-alle indeterminate={n > 0 && !a} />')).toBe(
			false
		);
		expect(alleOhneStrich('<Kaestchen checked={a} />')).toBe(false);
		expect(sichtbarerText('<AuswahlLeiste satz="{markiert.length} ausgewählt"')).not.toMatch(
			/markier/
		);
	});

	it('jede Liste mit Kästchen in einer Schleife trägt use:mehrfachauswahl', () => {
		const listen = dateien().filter((d) => kaestchenInSchleife(d.quelle));
		// Fände der Detektor keine, bliebe die Ratsche still grün.
		expect(listen.length).toBeGreaterThan(8);
		expect(
			listen
				.filter((d) => !/use:mehrfachauswahl\b/.test(d.quelle) && !(d.pfad in KEINE_AUSWAHL))
				.map((d) => d.pfad),
			'Eine Liste mit Kästchen trägt use:mehrfachauswahl (actions/mehrfachauswahl.js).'
		).toEqual([]);
	});

	it('das Kästchen im Kopf einer solchen Liste trägt data-auswahl-alle', () => {
		expect(
			dateien()
				.filter((d) => /use:mehrfachauswahl\b/.test(d.quelle) && kopfOhneMarke(d.quelle))
				.map((d) => d.pfad)
		).toEqual([]);
	});

	it('das Kästchen für alle zeigt bei teilweiser Auswahl den Strich', () => {
		const mitAlle = dateien().filter((d) =>
			koepfe(d.quelle, 'Kaestchen').some((k) => /data-auswahl-alle/.test(k))
		);
		expect(mitAlle.length).toBeGreaterThan(4);
		expect(mitAlle.filter((d) => alleOhneStrich(d.quelle)).map((d) => d.pfad)).toEqual([]);
	});

	it('die Leiste zur Auswahl sagt „ausgewählt“, nicht „markiert“', () => {
		const leisten = dateien().flatMap((d) =>
			koepfe(d.quelle, 'AuswahlLeiste').map((kopf) => ({ pfad: d.pfad, kopf }))
		);
		expect(leisten.length).toBeGreaterThan(3);
		expect(
			leisten.filter((l) => /markier/i.test(sichtbarerText(l.kopf))).map((l) => l.pfad)
		).toEqual([]);
	});

	it('jede Ausnahme trifft noch eine Liste', () => {
		const mitSchleife = new Set(
			dateien()
				.filter((d) => kaestchenInSchleife(d.quelle))
				.map((d) => d.pfad)
		);
		expect(Object.keys(KEINE_AUSWAHL).filter((p) => !mitSchleife.has(p))).toEqual([]);
	});
});
