import { describe, it, expect } from 'vitest';
import { leeresGeraetFormular, geraetFormularAus, geraetNutzlast } from './geraetFormular.js';

const TABLET = {
	id: 'g1',
	modellname: 'Tablet 10',
	barcode_id: 'G-0001',
	seriennummer: null,
	zubehoer: 'Ladekabel',
	zustand_notiz: null,
	ist_ausleihbar: true
};

describe('geraetFormular', () => {
	it('ein neues Gerät schickt alle Felder der Maske', () => {
		const form = { ...leeresGeraetFormular(), modellname: 'iPad', barcode_id: 'G-0002' };
		expect(geraetNutzlast(form, false)).toEqual({
			modellname: 'iPad',
			barcode_id: 'G-0002',
			seriennummer: '',
			zubehoer: '',
			zustand_notiz: ''
		});
	});

	// Die Maske füllt sich aus der Zeile der Liste. Schickte sie jedes Feld zurück, schriebe
	// sie den Stand vom Öffnen über das, was ein anderer Platz inzwischen gespeichert hat.
	it('ein vorhandenes Gerät schickt nur, was seit dem Öffnen geändert wurde', () => {
		const form = geraetFormularAus(TABLET);
		form.zustand_notiz = 'Display-Kratzer';
		expect(geraetNutzlast(form, true)).toEqual({ zustand_notiz: 'Display-Kratzer' });
	});

	it('ohne Änderung ist die Nutzlast leer', () => {
		expect(geraetNutzlast(geraetFormularAus(TABLET), true)).toEqual({});
	});

	it('ein geleertes Feld geht leer mit', () => {
		const form = geraetFormularAus(TABLET);
		form.zubehoer = '';
		expect(geraetNutzlast(form, true)).toEqual({ zubehoer: '' });
	});

	it('der Barcode und das Defekt-Kennzeichen gehen beim Bearbeiten nie mit', () => {
		const form = geraetFormularAus(TABLET);
		form.barcode_id = 'G-9999';
		form.modellname = 'Tablet 11';
		expect(geraetNutzlast(form, true)).toEqual({ modellname: 'Tablet 11' });
	});

	it('fehlt der Stand vom Öffnen, wird nichts geschickt', () => {
		expect(() => geraetNutzlast({ modellname: 'X' }, true)).toThrow(/Stand vom Öffnen/);
	});
});
