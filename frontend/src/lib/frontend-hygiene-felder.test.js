import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad } from './hygiene-quellen.js';

// Struktur-Invariante, gleiche Bauart wie Farb-, Symbol- und Suchfeld-Ratsche:
// Eingabefelder kommen aus components/ui/Feld.svelte, nicht aus der Zwischenablage.
//
// Warum es diese Ratsche gibt (25.08.2026): Beim Versuch, nur die neun Datumsfelder auf
// M3 zu ziehen, fiel auf, dass ihre NACHBARN (Vorname, Nachname, Titel …) denselben
// alten Slate-Stil trugen — 81 handgebaute <input> in 47 Dateien mit sieben Radien, vier
// Fokusfarben und drei Flächen. Buttons, Auswahlfelder und Suchfelder waren längst
// normiert, die Textfelder nicht. Ein Datumsfeld allein im M3-Rahmen hätte das Formular
// UNeinheitlicher gemacht, nicht einheitlicher.

/**
 * Ein rohes Textfeld: jedes <input>, das kein Schalter (checkbox/radio), keine Datei,
 * kein verstecktes Feld, kein Schieberegler und keine Farbwahl ist. Diese Typen haben
 * eine eigene Gestalt; alles andere (text, number, date, month, email, password,
 * search, tel, url — oder gar kein type) ist ein Textfeld und gehört ins Bauteil.
 */
const ROH_INPUT =
	/<input\b(?![^>]*type="(?:checkbox|radio|file|hidden|range|color|submit|button)")[^>]*>/gs;

// ── Ratsche ─────────────────────────────────────────────────────────────────
// Diese Zahl darf NUR sinken. Wer eine Fundstelle auf das Bauteil umstellt, trägt den
// neuen Stand hier ein — der Test sagt einem die Zahl.
const HANDGEBAUT_BESTAND = 0;

/**
 * Bewusste Ausnahmen. Jede braucht einen Grund, den das Bauteil nicht ausdrücken kann —
 * „sonst ist es rot" zählt nicht.
 */
const AUSNAHMEN = [
	{
		datei: 'src/lib/components/OmniboxInput.svelte',
		grund:
			'Füllt die 48-px-Scan-Pille (h-full, ohne eigenen Rahmen) — die Pille ist das ' +
			'Bedienelement, nicht das Feld. Gate: e2e/suchpille-einheitlich.spec.js.'
	},
	{
		datei: 'src/lib/components/AusweisGueltigkeit.svelte',
		grund:
			'Zusammengesetzte Pille „[Symbol] Gültig bis 31.07. [Jahr] [Zurücksetzen]" mit EINEM ' +
			'Rahmen um alles und amber-Zustand bei fehlendem Wert. Das Jahr allein im Feld-Rahmen ' +
			'zeigte Datum und Jahr als zwei Elemente. Auflösbar mit einer prefix-Prop am Feld.'
	},
	{
		datei: 'src/inventur/lib/components/admin/ClassAssignmentSelector.svelte',
		grund:
			'Unsichtbares Tipp-Feld IM Chip-Kasten (bg-transparent, border-none): Der Kasten ist ' +
			'das Feld, die Chips sind sein Wert. Ein zweiter Rahmen darin wäre falsch. ' +
			'e2e/klassensatz-dialog.spec.js hängt an #class-input.'
	}
];

/** Die Bauteile selbst enthalten naturgemäß ein <input>. */
const BAUTEILE = [
	'src/lib/components/ui/Feld.svelte',
	'src/lib/components/ui/Suchfeld.svelte',
	'src/lib/components/ui/Suchpille.svelte'
];

/** Zählt rohe Textfelder je Datei. */
function zaehleProDatei() {
	/** @type {{ datei: string, treffer: number }[]} */
	const out = [];
	for (const f of sammleQuelldateien(srcRoot)) {
		const pfad = relPfad(f);
		if (!pfad.endsWith('.svelte')) continue;
		if (AUSNAHMEN.some((a) => a.datei === pfad) || BAUTEILE.includes(pfad)) continue;
		const treffer = (readFileSync(f, 'utf8').match(ROH_INPUT) ?? []).length;
		if (treffer > 0) out.push({ datei: pfad, treffer });
	}
	return out.sort((a, b) => b.treffer - a.treffer);
}

describe('Feld-Hygiene', () => {
	it('baut keine Eingabefelder von Hand (sie kommen aus components/ui/Feld.svelte)', () => {
		const proDatei = zaehleProDatei();
		const summe = proDatei.reduce((n, e) => n + e.treffer, 0);
		const liste = proDatei.map((e) => `  ${String(e.treffer).padStart(3)}  ${e.datei}`).join('\n');

		expect(
			summe,
			`Handgebaute Eingabefelder: ${summe} statt ${HANDGEBAUT_BESTAND}.\n` +
				`Textfelder kommen aus components/ui/Feld.svelte (mit label im Formular, ` +
				`ohne label + aria-label in Tabelle oder Werkzeugleiste).\n` +
				`Offene Fundstellen:\n${liste}`
		).toBeLessThanOrEqual(HANDGEBAUT_BESTAND);

		expect(
			summe,
			`${HANDGEBAUT_BESTAND - summe} Fundstelle(n) sind umgestellt — danke.\n` +
				`Bitte HANDGEBAUT_BESTAND in dieser Datei auf ${summe} setzen, damit die Ratsche greift.`
		).toBe(HANDGEBAUT_BESTAND);
	});

	it('jede Ausnahme existiert noch und enthält tatsächlich ein rohes Feld', () => {
		for (const a of AUSNAHMEN) {
			const quelle = readFileSync(`${srcRoot}/../${a.datei}`, 'utf8');
			expect(
				(quelle.match(ROH_INPUT) ?? []).length,
				`${a.datei}: Ausnahme ohne Fundstelle`
			).toBeGreaterThan(0);
		}
	});

	it('gibt keinem Zahlenfeld den Text-Standard (min/max/step verlangen type="number")', () => {
		// Am 25.08.2026 gefunden von e2e/lehrer-reservierung.spec.js: SettingField hatte
		// type="number" als STANDARD, Feld hat "text". Zwölf Einstellungsfelder und die
		// Anzahl der Klassensatz-Reservierung liefen nach der Umstellung als String ans
		// Backend — „cannot unmarshal string into … anzahl of type int". Ein Standardwert,
		// der beim Bauteilwechsel kippt, ist ein stiller Fehler; deshalb steht die
		// Absicht jetzt an jeder Fundstelle.
		const fehlend = [];
		for (const f of sammleQuelldateien(srcRoot)) {
			if (!f.endsWith('.svelte')) continue;
			for (const t of readFileSync(f, 'utf8').match(/<Feld\b[^]*?\/>/g) ?? []) {
				if (/\b(min|max|step)=/.test(t) && !/\btype="number"/.test(t))
					fehlend.push(`${relPfad(f)}: ${t.split(/\s+/).slice(0, 4).join(' ')} …`);
			}
		}
		expect(fehlend, 'Feld mit min/max/step, aber ohne type="number"').toEqual([]);
	});

	it('erkennt ein Textfeld und verwechselt es nicht mit Schaltern', () => {
		// Gegenprobe am DETEKTOR, nicht am Bestand — feste Beispiele, unabhängig vom Baum.
		const trefferMuss = [
			'<input type="text" bind:value={x} />',
			'<input bind:value={x} placeholder="ohne type" />',
			'<input type="date" bind:value={d} />',
			'<input\n\ttype="number"\n\tmin={0}\n/>'
		];
		for (const beispiel of trefferMuss) {
			expect(
				(beispiel.match(new RegExp(ROH_INPUT.source, 'gs')) ?? []).length,
				`nicht erkannt: ${beispiel}`
			).toBe(1);
		}
		const trefferDarfNicht = [
			'<input type="checkbox" bind:checked={x} />',
			'<input type="radio" name="a" />',
			'<input type="file" accept="image/*" />',
			'<input type="hidden" name="csrf" />',
			'<input type="range" min="0" max="10" />'
		];
		for (const beispiel of trefferDarfNicht) {
			expect(
				(beispiel.match(new RegExp(ROH_INPUT.source, 'gs')) ?? []).length,
				`fälschlich erkannt: ${beispiel}`
			).toBe(0);
		}
	});
});
