import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad } from './hygiene-quellen.js';

// Sechste Struktur-Invariante, gleiche Bauart wie die Farb- und Symbol-Ratschen:
// Suchfelder kommen aus einem Bauteil, nicht aus der Zwischenablage.
//
// Warum es diese Ratsche gibt (10.08.2026): Peter legte zwei Bildschirme nebeneinander —
// „die omnibox bei mein portal und katalog ist eine komplett andere". Gemessen stimmte das
// an sieben Werten gleichzeitig: Höhe, Radius, Fläche, Fokusfarbe, Schriftgröße,
// Platzhaltertext und Leerzustand. Nichts davon war je entschieden worden. Es war kopiert
// und dann auseinandergelaufen — zehn Fundstellen, keine zwei gleich.
//
// Zwei Bauteile, weil es zwei Rollen gibt:
//   * components/ui/Suchpille.svelte — 48 px, rund, gefüllt. Das Werkzeug einer ganzen
//     Seite (Kiosk, Medienkatalog, Portal, OPAC). Bewusst neben der Control-Skala.
//   * components/ui/Suchfeld.svelte  — 36 px, eckig. Steht IN einer Werkzeugleiste neben
//     Knöpfen und Auswahlfeldern und teilt deren Grundlinie.

/**
 * Ein Feld gilt als Suchfeld, wenn sein Platzhalter oder aria-label nach Suchen oder
 * Filtern klingt — und es kein Ankreuzfeld, Datum oder Zahlenfeld ist.
 *
 * Bewusst am TEXT und nicht an `type="search"`: Genau die Hälfte der Fundstellen stand auf
 * `type="text"`, und eine Regel, die nur die halbe Menge sieht, meldet Erfolg, wo keiner
 * ist.
 *
 * Der erste Anlauf suchte nach der Silbe „uch" und fing damit drei Felder, die nichts mit
 * Suche zu tun haben: „Buch auswählen" (ein Ankreuzfeld), „z. B. BIB Jugendbuch" und
 * „braucht es dringend für Referat". Ein Detektor mit Fehltreffern ist schlimmer als
 * keiner — man gewöhnt sich an rote Meldungen und liest sie irgendwann nicht mehr.
 */
const SUCH_INPUT =
	/<input\b(?![^>]*type="(?:checkbox|radio|date|number|file|hidden|color|range)")[^>]*(?:placeholder|aria-label)="[^"]*(?:[Ss]uch|[Ff]ilter)[^"]*"[^>]*>/gs;

// ── Ratsche ─────────────────────────────────────────────────────────────────
// Diese Zahl darf NUR sinken. Wer eine Fundstelle auf ein Bauteil umstellt, trägt den
// neuen Stand hier ein — der Test sagt einem die Zahl.
const HANDGEBAUT_BESTAND = 1;

/**
 * Bewusste Ausnahmen. Jede braucht einen Grund, den das Bauteil nicht ausdrücken kann —
 * „sonst ist es rot" zählt nicht.
 */
// Die Kiosk-Omnibox taucht hier gar nicht auf: Ihr Platzhalter ist ein Ausdruck
// (`placeholder={isActive ? … : …}`), kein Literal. Sie ist trotzdem kein blinder Fleck —
// e2e/suchpille-einheitlich.spec.js misst sie im Browser gegen die Pille.
const AUSNAHMEN = [
	{
		datei: 'src/inventur/lib/components/KlassenSuchfeld.svelte',
		grund:
			'Steht in einer Zeile mit zwei Select-Feldern und einem Button und trägt deshalb ' +
			'bewusst deren M3-Rollen-Vokabular (border-outline-variant statt border-slate-200), ' +
			'damit die Leiste als EINE liest. Auflösbar erst mit der Paletten-Migration, siehe ' +
			'frontend-hygiene-farben.test.js.'
	}
];

/** Zählt handgebaute Suchfelder je Datei. */
function zaehleProDatei() {
	/** @type {{ datei: string, treffer: number }[]} */
	const out = [];
	for (const f of sammleQuelldateien(srcRoot)) {
		const pfad = relPfad(f);
		if (AUSNAHMEN.some((a) => a.datei === pfad)) continue;
		// Die Bauteile selbst enthalten naturgemäß ein solches <input>.
		if (pfad.endsWith('ui/Suchpille.svelte') || pfad.endsWith('ui/Suchfeld.svelte')) continue;

		const treffer = (readFileSync(f, 'utf8').match(SUCH_INPUT) ?? []).length;
		if (treffer > 0) out.push({ datei: pfad, treffer });
	}
	return out.sort((a, b) => b.treffer - a.treffer);
}

describe('Suchfeld-Hygiene', () => {
	it('baut keine neuen Suchfelder von Hand (sie kommen aus Suchpille/Suchfeld)', () => {
		const proDatei = zaehleProDatei();
		const summe = proDatei.reduce((n, e) => n + e.treffer, 0);
		const liste = proDatei.map((e) => `  ${String(e.treffer).padStart(3)}  ${e.datei}`).join('\n');

		expect(
			summe,
			`Neue handgebaute Suchfelder: ${summe} statt ${HANDGEBAUT_BESTAND}.\n` +
				`Suchfelder kommen aus einem Bauteil:\n` +
				`  Werkzeug der Seite (48 px, Pille)   components/ui/Suchpille.svelte\n` +
				`  Feld in einer Werkzeugleiste (36 px) components/ui/Suchfeld.svelte\n` +
				`Offene Fundstellen:\n${liste}`
		).toBeLessThanOrEqual(HANDGEBAUT_BESTAND);

		expect(
			summe,
			`${HANDGEBAUT_BESTAND - summe} Fundstelle(n) sind umgestellt — danke.\n` +
				`Bitte HANDGEBAUT_BESTAND in dieser Datei auf ${summe} setzen, damit die Ratsche greift.`
		).toBe(HANDGEBAUT_BESTAND);
	});

	it('erkennt ein handgebautes Suchfeld und verwechselt es nicht mit anderen Feldern', () => {
		// Gegenprobe am DETEKTOR, nicht am Bestand.
		//
		// Der erste Anlauf prüfte, ob die beiden Bauteile selbst dem Muster entsprechen —
		// sie tun es nicht, denn dort steht `placeholder={platzhalter}` als Bindung und kein
		// Literal. Genau richtig fürs Zählen, als Gegenprobe aber unbrauchbar. Und eine
		// Gegenprobe an einer echten Fundstelle veraltet, sobald diese umgestellt wird.
		// Also feste Beispiele, die unabhängig vom Zustand des Baums gelten.
		const trefferMuss = [
			'<input type="search" placeholder="Schüler suchen …" />',
			'<input type="text" aria-label="Klasse suchen" bind:value={q} />',
			'<input\n\ttype="text"\n\tplaceholder="In 12 Titeln filtern …"\n/>' // mehrzeilig
		];
		for (const beispiel of trefferMuss) {
			expect(
				(beispiel.match(new RegExp(SUCH_INPUT.source, 'gis')) ?? []).length,
				`nicht erkannt: ${beispiel}`
			).toBe(1);
		}

		// Die drei Fehltreffer des ersten Anlaufs stehen hier als feste Beispiele: Sie sollen
		// nie wieder durchrutschen, auch wenn die echten Fundstellen einmal verschwinden.
		const trefferDarfNicht = [
			'<input type="text" placeholder="Vorname" />',
			'<input type="date" aria-label="Rückgabedatum" />',
			'<input type="checkbox" aria-label="Buch auswählen" />',
			'<input type="text" placeholder="z. B. BIB Jugendbuch" />',
			'<input type="text" placeholder="z.B. Braucht es dringend für Referat" />',
			'<textarea placeholder="Notiz zur Suche"></textarea>' // kein <input>
		];
		for (const beispiel of trefferDarfNicht) {
			expect(
				(beispiel.match(new RegExp(SUCH_INPUT.source, 'gis')) ?? []).length,
				`fälschlich erkannt: ${beispiel}`
			).toBe(0);
		}
	});
});
