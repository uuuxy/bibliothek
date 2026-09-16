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

	it('schickt die Ausweisnummer mit und nie die id', () => {
		const nutzlast = benutzerNutzlast({
			...leeresBenutzerFormular(),
			id: 'x',
			barcode_id: 'A-7',
			vorname: 'A',
			nachname: 'B',
			email: 'a@b'
		});
		expect(nutzlast).toHaveProperty('barcode_id', 'A-7');
		expect(nutzlast).not.toHaveProperty('id');
	});
});
