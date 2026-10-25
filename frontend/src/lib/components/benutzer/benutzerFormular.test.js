import { describe, it, expect } from 'vitest';
import {
	leeresBenutzerFormular,
	benutzerFormularAus,
	benutzerNutzlast
} from './benutzerFormular.js';

// Das Formular führt ein Konto: Name, E-Mail, Rolle, aktiv — und die Ausweisnummer, die in
// die Leserzeile geschrieben wird. Das Feld „Personenart" ist mit Migration 125 gefallen;
// wer jemand ist, steht an seiner Leserzeile und gehört in die Leserdatei.
describe('benutzerFormular', () => {
	it('ein neues Formular ist ein Mitarbeiter-Konto ohne Ausweis', () => {
		const form = leeresBenutzerFormular();
		expect(form.rolle).toBe('mitarbeiter');
		expect(form.aktiv).toBe(true);
		expect(form.barcode_id).toBe('');
	});

	// Das Formular darf keine Personenart mehr führen: Ein übrig gebliebenes Feld ginge als
	// unbekannter Schlüssel an den Server und wäre dort still wirkungslos.
	it('führt keine Personenart mehr — weder leer noch aus einem alten Konto', () => {
		expect(leeresBenutzerFormular()).not.toHaveProperty('personenart');
		expect(
			benutzerFormularAus({
				id: '1',
				vorname: 'L',
				nachname: 'V',
				email: 'l@x',
				rolle: 'kollegium',
				aktiv: true,
				personenart: 'liv'
			})
		).not.toHaveProperty('personenart');
		expect(benutzerNutzlast(leeresBenutzerFormular())).not.toHaveProperty('personenart');
	});

	it('übernimmt ein Konto; eine fehlende Ausweisnummer wird zum leeren Feld', () => {
		const ohne = benutzerFormularAus({
			id: '2',
			vorname: 'M',
			nachname: 'A',
			email: 'm@x',
			rolle: 'mitarbeiter',
			aktiv: false,
			barcode_id: null
		});
		expect(ohne.barcode_id).toBe('');
		expect(ohne.aktiv).toBe(false);
	});

	it('schickt die Ausweisnummer mit und nie die id oder den Stand vom Öffnen', () => {
		const neu = benutzerNutzlast({
			...leeresBenutzerFormular(),
			barcode_id: 'A-7',
			vorname: 'A',
			nachname: 'B',
			email: 'a@b'
		});
		expect(neu).toHaveProperty('barcode_id', 'A-7');
		expect(neu).not.toHaveProperty('id');

		const form = benutzerFormularAus({
			id: 'x',
			vorname: 'A',
			nachname: 'B',
			email: 'a@b',
			rolle: 'helfer',
			aktiv: true
		});
		form.barcode_id = 'A-8';
		const vorhanden = benutzerNutzlast(form);
		expect(vorhanden).toEqual({ barcode_id: 'A-8' });
		expect(vorhanden).not.toHaveProperty('geladen');
	});
	// Die Maske füllt sich aus der Zeile der Liste; die lädt beim Öffnen der Seite. Schickte das
	// Speichern jedes Feld zurück, schriebe es den alten Stand über das, was ein anderer Platz
	// inzwischen geändert hat: die Rolle, „aktiv", die Ausweisnummer aus der Leserakte.
	describe('ein vorhandenes Konto', () => {
		const konto = {
			id: '7',
			vorname: 'Gerda',
			nachname: 'Genannt',
			email: 'gerda@x',
			rolle: 'mitarbeiter',
			aktiv: true,
			barcode_id: 'A-10001'
		};

		it('schickt nur die Felder, die seit dem Öffnen geändert wurden', () => {
			const form = benutzerFormularAus(konto);
			form.vorname = 'Gerdi';
			expect(benutzerNutzlast(form)).toEqual({ vorname: 'Gerdi' });

			form.aktiv = false;
			form.barcode_id = '';
			expect(benutzerNutzlast(form)).toEqual({ vorname: 'Gerdi', aktiv: false, barcode_id: '' });
		});

		it('schickt nichts, wenn nichts geändert wurde, auch nach Hin und Zurück', () => {
			const form = benutzerFormularAus(konto);
			expect(benutzerNutzlast(form)).toEqual({});
			form.rolle = 'helfer';
			form.rolle = 'mitarbeiter';
			expect(benutzerNutzlast(form)).toEqual({});
		});

		it('weigert sich ohne den Stand vom Öffnen: Sonst gälte jedes Feld als geändert', () => {
			expect(() => benutzerNutzlast({ ...leeresBenutzerFormular(), id: '7' })).toThrow(
				/Stand vom Öffnen/
			);
		});
	});

	it('ein neues Konto schickt alle Felder', () => {
		const form = { ...leeresBenutzerFormular(), vorname: 'N', nachname: 'K', email: 'n@k' };
		expect(benutzerNutzlast(form)).toEqual({
			barcode_id: '',
			vorname: 'N',
			nachname: 'K',
			email: 'n@k',
			rolle: 'mitarbeiter',
			aktiv: true
		});
	});
});
