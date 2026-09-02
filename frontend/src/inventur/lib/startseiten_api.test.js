import { describe, it, expect } from 'vitest';
import {
	buecherNachKlassenGruppieren,
	klassenFiltern,
	zweigOptionenAus,
	jahrgangOptionenAus,
	STANDARD_GRUPPE,
	buecherSuchen,
	buecherFiltern,
	buecherSortieren,
	fachOptionenAus,
	medientypOptionenAus,
	leererFilter,
	SORTIERUNGEN,
	BESTAND_FILTER
} from './startseiten_api.js';

// Der Reiter „Jahrgänge" liest seit dem 24.08.2026 die JAHRGANGSSPANNE
// (jahrgangVon/Bis), nicht mehr `gradeLevel` — das Feld, das der Littera-Import nie
// setzt. Vorher war der Reiter für importierte Bestände leer, während handgepflegte
// Spannen (5–5, 6–6) unsichtbar daneben standen. Diese Tests halten die drei
// Zusagen des Umbaus fest: Spanne ist die Quelle, der Import-Default 5–10 sammelt
// sich erkennbar am Ende, und die Filter-Optionen entstehen aus den Daten.

const buch = (/** @type {any} */ felder) => ({ id: Math.random().toString(), ...felder });

describe('buecherNachKlassenGruppieren', () => {
	it('gruppiert nach der Jahrgangsspanne, nicht nach gradeLevel', () => {
		const gruppen = buecherNachKlassenGruppieren([
			buch({ jahrgangVon: 5, jahrgangBis: 5 }),
			// gradeLevel gesetzt, Spanne auf dem Import-Default: Das ist der alte
			// Datenbestand — er darf NICHT als „Klasse 9" erscheinen, sonst wären
			// zwei Vokabulare wieder gleichzeitig wahr.
			buch({ gradeLevel: 9, jahrgangVon: 5, jahrgangBis: 10 }),
			buch({ jahrgangVon: 5, jahrgangBis: 6, track: 'Förderstufe' })
		]);
		expect(gruppen.map((g) => g.name)).toEqual([
			'Klasse 5',
			'Klasse 5–6 Förderstufe',
			STANDARD_GRUPPE
		]);
	});

	it('sammelt den Import-Default 5–10 in einer Gruppe am Ende', () => {
		const gruppen = buecherNachKlassenGruppieren([
			buch({ jahrgangVon: 5, jahrgangBis: 10 }),
			buch({ jahrgangVon: 5, jahrgangBis: 10 }),
			buch({ jahrgangVon: 10, jahrgangBis: 10 })
		]);
		expect(gruppen.at(-1)?.name).toBe(STANDARD_GRUPPE);
		expect(gruppen.at(-1)?.books).toHaveLength(2);
		expect(gruppen.at(-1)?.standard).toBe(true);
		// Eine BEWUSSTE 10–10 bleibt eine echte Gruppe.
		expect(gruppen[0].name).toBe('Klasse 10');
	});
});

describe('klassenFiltern', () => {
	const gruppen = buecherNachKlassenGruppieren([
		buch({ jahrgangVon: 5, jahrgangBis: 6, track: 'Förderstufe' }),
		buch({ jahrgangVon: 7, jahrgangBis: 7, track: 'Gymnasium' }),
		buch({ jahrgangVon: 5, jahrgangBis: 10 })
	]);

	it('versteht „Jahrgang 6" als „Spanne deckt die 6 ab"', () => {
		const namen = klassenFiltern(gruppen, '', '6').map((g) => g.name);
		expect(namen).toContain('Klasse 5–6 Förderstufe');
		expect(namen).not.toContain('Klasse 7 Gymnasium');
	});

	it('lässt die Standard-Gruppe unter jedem Filter stehen', () => {
		// Sie ist der sichtbare Rest, der noch Pflege braucht — ausgefiltert wäre er
		// im Reiter unauffindbar (Betreiber-Entscheidung 24.08.2026).
		expect(klassenFiltern(gruppen, 'Gymnasium', '9').map((g) => g.name)).toEqual([STANDARD_GRUPPE]);
	});
});

describe('Filter-Optionen aus den Daten', () => {
	it('bietet genau die Zweige an, die im Bestand vorkommen — Förderstufe inklusive', () => {
		const optionen = zweigOptionenAus([
			buch({ track: 'Gymnasium' }),
			buch({ track: 'Förderstufe' }),
			buch({ track: 'Förderstufe' }),
			buch({ track: '' })
		]);
		expect(optionen.map((o) => o.value)).toEqual(['', 'Förderstufe', 'Gymnasium']);
	});

	it('bietet nur Jahrgänge an, die eine gepflegte Spanne abdeckt', () => {
		const gruppen = buecherNachKlassenGruppieren([
			buch({ jahrgangVon: 5, jahrgangBis: 6 }),
			buch({ jahrgangVon: 5, jahrgangBis: 10 }) // Standard — zählt nicht
		]);
		expect(jahrgangOptionenAus(gruppen).map((o) => o.value)).toEqual(['', '5', '6']);
	});
});

// Filter, Sortierung und Signatur-Suche der Buch-Suche (02.09.2026): Der Reiter hieß
// „Suche & Filter", hatte aber keinen Filter; die Signatur (Regaladresse) fand die
// Suche nicht, obwohl der Payload sie trug; Titel ohne Exemplare waren von komplett
// verliehenen nicht zu unterscheiden.
const bestand = [
	buch({
		title: 'Faust',
		author: 'Goethe',
		subject: 'Deutsch',
		signatur: 'Deu Goe',
		verfuegbar: 2,
		gesamt: 3,
		jahrgangVon: 9,
		jahrgangBis: 10,
		track: 'Gymnasium'
	}),
	buch({
		title: 'Analysis',
		subject: 'Mathematik',
		signatur: 'Mat Ana',
		verfuegbar: 0,
		gesamt: 5,
		gradeLevel: 11
	}),
	buch({ title: 'Ohne Bestand', subject: 'Deutsch', signatur: '', verfuegbar: 0, gesamt: 0 }),
	buch({ title: 'Hörbuch', subject: 'Englisch', medientyp: 'CD', verfuegbar: 1, gesamt: 1 })
];

describe('buecherSuchen: Signatur', () => {
	it('findet ein Buch über seine Signatur (Regaladresse)', () => {
		expect(buecherSuchen(bestand, 'goe').map((b) => b.title)).toEqual(['Faust']);
	});
});

describe('buecherFiltern', () => {
	it('ohne Filter bleibt alles', () => {
		expect(buecherFiltern(bestand, leererFilter())).toHaveLength(4);
	});

	it('Fach, Zweig und Medienart filtern exakt; ohne Angabe ist ein Medium ein Buch', () => {
		expect(buecherFiltern(bestand, { ...leererFilter(), fach: 'Deutsch' })).toHaveLength(2);
		expect(buecherFiltern(bestand, { ...leererFilter(), zweig: 'Gymnasium' })).toHaveLength(1);
		expect(buecherFiltern(bestand, { ...leererFilter(), medientyp: 'Buch' })).toHaveLength(3);
		expect(buecherFiltern(bestand, { ...leererFilter(), medientyp: 'CD' })).toHaveLength(1);
	});

	it('Jahrgang trifft gradeLevel ODER die Spanne — dieselbe Regel wie die Suche', () => {
		expect(
			buecherFiltern(bestand, { ...leererFilter(), jahrgang: '10' }).map((b) => b.title)
		).toEqual(['Faust']);
		expect(
			buecherFiltern(bestand, { ...leererFilter(), jahrgang: '11' }).map((b) => b.title)
		).toEqual(['Analysis']);
	});

	it('Bestand: „nur verfügbare" und „ohne Exemplare" sind verschiedene Mengen', () => {
		expect(
			buecherFiltern(bestand, { ...leererFilter(), bestand: 'verfuegbar' }).map((b) => b.title)
		).toEqual(['Faust', 'Hörbuch']);
		expect(
			buecherFiltern(bestand, { ...leererFilter(), bestand: 'ohne' }).map((b) => b.title)
		).toEqual(['Ohne Bestand']);
	});
});

describe('buecherSortieren', () => {
	it('ohne Wahl bleibt die gepflegte Reihenfolge — dieselbe Liste, nicht umsortiert', () => {
		expect(buecherSortieren(bestand, '')).toBe(bestand);
	});

	it('sortiert nach Titel, Fach, Signatur (Leere ans Ende) und Verfügbarkeit', () => {
		expect(buecherSortieren(bestand, 'titel').map((b) => b.title)).toEqual([
			'Analysis',
			'Faust',
			'Hörbuch',
			'Ohne Bestand'
		]);
		expect(buecherSortieren(bestand, 'fach').map((b) => b.title)).toEqual([
			'Faust',
			'Ohne Bestand',
			'Hörbuch',
			'Analysis'
		]);
		expect(buecherSortieren(bestand, 'signatur').map((b) => b.title)).toEqual([
			'Faust',
			'Analysis',
			'Hörbuch',
			'Ohne Bestand'
		]);
		expect(buecherSortieren(bestand, 'verfuegbar')[0].title).toBe('Faust');
		// Die Vorlage bleibt unangetastet.
		expect(bestand[0].title).toBe('Faust');
	});

	it('jede angebotene Sortierung und jeder Bestandsfilter existiert wirklich', () => {
		for (const s of SORTIERUNGEN)
			expect(Array.isArray(buecherSortieren(bestand, s.value))).toBe(true);
		for (const f of BESTAND_FILTER)
			expect(Array.isArray(buecherFiltern(bestand, { ...leererFilter(), bestand: f.value }))).toBe(
				true
			);
	});
});

describe('Optionslisten der Buch-Suche', () => {
	it('entstehen aus den Daten, alphabetisch, mit „Alle" voran', () => {
		expect(fachOptionenAus(bestand).map((o) => o.label)).toEqual([
			'Alle Fächer',
			'Deutsch',
			'Englisch',
			'Mathematik'
		]);
		expect(medientypOptionenAus(bestand).map((o) => o.value)).toEqual(['', 'Buch', 'CD']);
	});
});
