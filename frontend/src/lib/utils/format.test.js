import { describe, it, expect } from 'vitest';
import {
	formatProzent,
	formatDatum,
	formatZeitpunkt,
	formatZahl,
	bestandSatz,
	zulaufSatz,
	jahrgangSpanne
} from './format.js';
import jahrgangFaelle from './jahrgangSpanne.faelle.json';

describe('format.js — eine deutsche Schreibweise', () => {
	it('Prozent: Komma, eine Nachkommastelle, geschütztes Leerzeichen; Rohstring vom Backend geht auch', () => {
		expect(formatProzent(23.94)).toBe('23,9 %');
		expect(formatProzent('1.18')).toBe('1,2 %');
		expect(formatProzent(0)).toBe('0 %');
		expect(formatProzent(undefined)).toBe('0 %');
		expect(formatProzent('kaputt')).toBe('0 %');
	});

	it('Datum und Zeitpunkt: immer zweistellig, Zeitpunkt ohne Sekunden', () => {
		const t = new Date(2026, 8, 1, 21, 5, 3);
		expect(formatDatum(t)).toBe('01.09.2026');
		expect(formatZeitpunkt(t)).toBe('01.09.2026, 21:05');
		expect(formatDatum(null)).toBe('');
		expect(formatZeitpunkt('nicht-datum')).toBe('');
	});

	it('Zahl: Tausenderpunkt', () => {
		expect(formatZahl(12345)).toBe('12.345');
	});

	// Drei Fälle, und der dritte ist der Grund für die Funktion: „nicht gezählt" ist
	// nicht „keins da". Katalog und Theke beantworteten ihn bis 17.09.2026 verschieden.
	it('Bestand: Satz, „Keine Exemplare" bei 0 — und nichts, wenn niemand gezählt hat', () => {
		expect(bestandSatz(28, 27)).toBe('27 von 28 verfügbar');
		expect(bestandSatz(5, 0)).toBe('0 von 5 verfügbar');
		expect(bestandSatz(0, 0)).toBe('Keine Exemplare');
		expect(bestandSatz(null, null)).toBe('');
		expect(bestandSatz(undefined, undefined)).toBe('');
		// Zahl da, Verfügbarkeit fehlt: lieber 0 behaupten als „NaN von 3".
		expect(bestandSatz(3, undefined)).toBe('0 von 3 verfügbar');
	});

	// Nichts im Regal, alles bestellt: Der Titel steht in der Trefferliste, weil der Zulauf
	// als vorhanden zählt — der Satz muss sagen, warum (docs/OFFEN.md 5.5).
	it('Bestand: „2 bestellt" statt „Keine Exemplare", wenn nur der Zulauf etwas hat', () => {
		expect(bestandSatz(0, 0, 2)).toBe('2 bestellt');
		expect(bestandSatz(0, 0, 0)).toBe('Keine Exemplare');
		expect(bestandSatz(0, 0, undefined)).toBe('Keine Exemplare');
		// Etwas im Regal: Die Zahl dort beantwortet die Frage an der Theke, der Zulauf nicht.
		expect(bestandSatz(1, 1, 2)).toBe('1 von 1 verfügbar');
		expect(bestandSatz(null, null, 2)).toBe('');
	});

	// Wo der Zulauf zur Frage gehört (Klassensatz im Portal), steht er neben dem Bestand —
	// in denselben Worten wie beim Titel, der nur bestellt ist.
	it('Zulauf: „20 bestellt", und nichts, wenn nichts bestellt ist', () => {
		expect(zulaufSatz(20)).toBe('20 bestellt');
		expect(zulaufSatz(2)).toBe(bestandSatz(0, 0, 2));
		expect(zulaufSatz(0)).toBe('');
		expect(zulaufSatz(null)).toBe('');
		expect(zulaufSatz(undefined)).toBe('');
	});

	// Dieselben Fälle prüft die Go-Seite am Ausdruck der Schulbuchliste
	// (inventur/lernmittel_pdf_test.go).
	it('Jahrgang: ein Jahr, eine Spanne, und leer ohne Eintrag', () => {
		expect(jahrgangFaelle.faelle.length).toBeGreaterThan(0);
		for (const f of jahrgangFaelle.faelle) {
			expect(jahrgangSpanne(f.von, f.bis), f.fall).toBe(f.soll);
		}
	});

	// Die Maske liefert leere Felder als null oder leeren Text, die Liste als 0.
	it('Jahrgang: eine halbe oder fehlende Angabe ergibt keinen Text', () => {
		expect(jahrgangSpanne(null, null)).toBe('');
		expect(jahrgangSpanne('', '')).toBe('');
		expect(jahrgangSpanne(undefined, undefined)).toBe('');
		expect(jahrgangSpanne(7, 0)).toBe('');
		expect(jahrgangSpanne('7', '9')).toBe('7–9');
	});
});
