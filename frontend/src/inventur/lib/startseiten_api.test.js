import { describe, it, expect } from 'vitest';
import {
	buecherNachKlassenGruppieren,
	klassenFiltern,
	zweigOptionenAus,
	jahrgangOptionenAus,
	STANDARD_GRUPPE
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
