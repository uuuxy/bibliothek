import { describe, it, expect, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';
import CardFace from './CardFace.svelte';
import { idStore } from './idDesignerStore.svelte.js';

/**
 * Der Testdruck des Ausweis-Designers und der echte Ausweis rendern beide über
 * CardFace — und müssen sich an genau EINER Stelle unterscheiden: bei leeren Bild-,
 * Logo- und Passbildfeldern.
 *
 * Der gemeldete Fehler (24.08.2026): „Testausdruck im Druck-Center bringt nicht das,
 * was man sieht — beim Schüler stimmt es." Ursache war kein Layoutfehler, sondern
 * genau dieser Unterschied ins Leere gedreht. Die Leinwand zeichnet für Logo und
 * Passbild einen gestrichelten Rahmen; CardFace liess beide Felder ersatzlos weg,
 * wenn kein Inhalt da war. Der Platzhalter-Schüler des Designers hat kein Foto, ein
 * frisch aufgesetztes Design kein Logo — auf dem Testdruck fehlten deshalb ausgerechnet
 * die zwei flächenkritischen Felder, die man darauf prüfen will. Beim echten Schüler
 * fiel nichts auf, weil der ein Foto hat.
 *
 * Die Gegenrichtung ist der wichtigere Teil der Zusicherung: Ein ECHTER Ausweis darf
 * niemals einen Rahmen mit der Aufschrift „PASSBILD" tragen, nur weil beim Schüler
 * kein Foto hinterlegt ist. Deshalb steht `platzhalter` auf false, wenn es niemand setzt.
 */

const SCHUELER_OHNE_FOTO = {
	id: 's-1',
	vorname: 'Max',
	nachname: 'Mustermann',
	barcode_id: 'DEMO-S-001',
	ausweis_gueltig_bis: 2027
};

beforeEach(() => {
	idStore.front.elements = [
		{
			id: 'logo',
			type: 'logo',
			content: '',
			x: 65,
			y: 3,
			width: 15,
			height: 13,
			zIndex: 2,
			show: true
		},
		{
			id: 'photo',
			type: 'photo',
			content: '',
			x: 5,
			y: 21,
			width: 22,
			height: 25,
			zIndex: 2,
			show: true
		}
	];
});

describe('CardFace: leere Bildfelder', () => {
	it('zeigt sie im Testdruck des Designers als beschrifteten Rahmen', () => {
		const { getByText } = render(CardFace, {
			props: {
				side: 'front',
				student: SCHUELER_OHNE_FOTO,
				barcodeType: 'code39',
				platzhalter: true
			}
		});

		expect(getByText('LOGO')).toBeTruthy();
		expect(getByText('PASSBILD')).toBeTruthy();
	});

	it('lässt sie auf dem echten Ausweis leer — ohne Rahmen und ohne Aufschrift', () => {
		const { queryByText } = render(CardFace, {
			props: { side: 'front', student: SCHUELER_OHNE_FOTO, barcodeType: 'code39' }
		});

		expect(queryByText('LOGO')).toBeNull();
		expect(queryByText('PASSBILD')).toBeNull();
	});

	it('zeigt das echte Logo statt des Rahmens, sobald eines hinterlegt ist', () => {
		idStore.front.elements[0].content = 'data:image/png;base64,AAAA';

		const { queryByText, getByAltText } = render(CardFace, {
			props: {
				side: 'front',
				student: SCHUELER_OHNE_FOTO,
				barcodeType: 'code39',
				platzhalter: true
			}
		});

		expect(queryByText('LOGO')).toBeNull();
		expect(getByAltText('Bild')).toBeTruthy();
	});
});
