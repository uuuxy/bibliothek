import { describe, it, expect } from 'vitest';
import {
	PERSONENARTEN,
	leeresBenutzerFormular,
	benutzerFormularAus,
	benutzerNutzlast,
	personenartLabel,
	personenartOptionen
} from './benutzerFormular.js';

// Die Personenart sagt, wer jemand im Kollegium ist (Lehrkraft oder LiV), die Rolle, was er in
// der Software darf. Das Formular muss beides getrennt führen und die Personenart immer
// mitschicken: Fehlt das Feld, lässt der Server den alten Wert stehen — ein geleertes Feld soll
// aber leeren.
describe('benutzerFormular', () => {
	it('bietet leer, Lehrkraft und LiV an', () => {
		expect(PERSONENARTEN.map((p) => p.value)).toEqual(['', 'lehrkraft', 'liv']);
	});

	// Die Personenart entscheidet, wer als Lehrkraft ausleiht (Migration 120): Ein Kollegiumskonto
	// hat immer eine. „Keine Angabe" gibt es dort nicht — die Datenbank machte stillschweigend
	// „Lehrkraft" daraus.
	it('bietet beim Kollegium keine leere Personenart an', () => {
		expect(personenartOptionen('kollegium').map((p) => p.value)).toEqual(['lehrkraft', 'liv']);
		expect(personenartOptionen('mitarbeiter').map((p) => p.value)).toEqual([
			'',
			'lehrkraft',
			'liv'
		]);
	});

	it('ein neues Formular hat keine Personenart', () => {
		const form = leeresBenutzerFormular();
		expect(form.rolle).toBe('mitarbeiter');
		expect(form.personenart).toBe('');
		expect(form.aktiv).toBe(true);
	});

	it('übernimmt die Personenart eines Kontos, leer bleibt leer', () => {
		expect(
			benutzerFormularAus({
				id: '1',
				vorname: 'L',
				nachname: 'V',
				email: 'l@x',
				rolle: 'kollegium',
				aktiv: true,
				personenart: 'liv'
			}).personenart
		).toBe('liv');
		const ohne = benutzerFormularAus({
			id: '2',
			vorname: 'M',
			nachname: 'A',
			email: 'm@x',
			rolle: 'mitarbeiter',
			aktiv: false,
			personenart: null,
			barcode_id: null
		});
		expect(ohne.personenart).toBe('');
		expect(ohne.barcode_id).toBe('');
	});

	it('schickt die Personenart immer mit, auch leer, und nie die id', () => {
		const nutzlast = benutzerNutzlast({
			...leeresBenutzerFormular(),
			id: 'x',
			vorname: 'A',
			nachname: 'B',
			email: 'a@b'
		});
		expect(nutzlast).toHaveProperty('personenart', '');
		expect(nutzlast).not.toHaveProperty('id');
	});

	it('nennt die Personenart in Worten', () => {
		expect(personenartLabel('lehrkraft')).toBe('Lehrkraft');
		expect(personenartLabel('liv')).toBe('LiV');
		expect(personenartLabel(null)).toBe('');
		expect(personenartLabel('')).toBe('');
	});
});
