import { describe, it, expect } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import Feld from './Feld.svelte';
import { alsZahl } from './nachkommastellen.js';

// Ein Betrag (step mit Nachkommastellen) ist ein Textfeld mit Zifferntastatur: Firefox kürzte den
// Wert eines Zahlenfelds zu „7,9“. Das Feld zeigt „7,90“ wie formatEuro, gebunden bleibt die Zahl.
describe('Feld: Betrag', () => {
	/** @param {unknown} value */
	function zeige(value) {
		const props = $state({
			'aria-label': 'Preis',
			type: /** @type {'number'} */ ('number'),
			step: '0.01',
			value
		});
		const r = render(Feld, { props });
		return { props, feld: /** @type {HTMLInputElement} */ (r.getByLabelText('Preis')) };
	}

	it('ist ein Textfeld mit Zifferntastatur und zeigt zwei Nachkommastellen mit Komma', async () => {
		const { props, feld } = zeige(7.9);
		await tick();
		expect(feld.type).toBe('text');
		expect(feld.getAttribute('inputmode')).toBe('decimal');
		expect(feld.value).toBe('7,90');

		props.value = 12;
		await tick();
		expect(feld.value).toBe('12,00');
	});

	it('lässt die Eingabe beim Tippen stehen, bindet die Zahl und ergänzt beim Verlassen', async () => {
		const { props, feld } = zeige(7.9);
		feld.focus();
		await fireEvent.input(feld, { target: { value: '8,5' } });
		expect(feld.value).toBe('8,5');
		expect(props.value).toBe(8.5);

		await fireEvent.input(feld, { target: { value: '8.25' } });
		expect(props.value).toBe(8.25);
		await fireEvent.blur(feld);
		expect(feld.value).toBe('8,25');

		await fireEvent.input(feld, { target: { value: '12,50 €' } });
		expect(props.value, 'eingefügt mit Eurozeichen').toBe(12.5);
		await fireEvent.blur(feld);
		expect(feld.value).toBe('12,50');
	});

	it('bindet null, wenn das Feld leer ist oder keine Zahl trägt, und lässt den Text stehen', async () => {
		const { props, feld } = zeige(7.9);
		await fireEvent.input(feld, { target: { value: '' } });
		expect(props.value).toBeNull();

		await fireEvent.input(feld, { target: { value: '7,9x' } });
		expect(props.value).toBeNull();
		await fireEvent.blur(feld);
		expect(feld.value).toBe('7,9x');
	});

	it('nimmt beim Tippen nur Ziffern, Komma, Punkt und Vorzeichen an, wie Chrome am Zahlenfeld', () => {
		const { feld } = zeige(7.9);
		const tippe = (/** @type {string} */ data) => {
			const e = new InputEvent('beforeinput', { data, inputType: 'insertText', cancelable: true });
			feld.dispatchEvent(e);
			return e.defaultPrevented;
		};
		expect(tippe('o'), 'ein Buchstabe').toBe(true);
		expect(tippe('€'), 'das Eurozeichen').toBe(true);
		for (const zeichen of ['5', ',', '.', '-']) expect(tippe(zeichen), zeichen).toBe(false);
	});

	it('lässt ganze Zahlen und Text, wie sie sind', async () => {
		const r = render(Feld, { 'aria-label': 'Anzahl', type: 'number', value: 3 });
		const t = render(Feld, { 'aria-label': 'Nummer', value: '0123' });
		await tick();
		const anzahl = /** @type {HTMLInputElement} */ (r.getByLabelText('Anzahl'));
		const nummer = /** @type {HTMLInputElement} */ (t.getByLabelText('Nummer'));
		expect(anzahl.type).toBe('number');
		expect(anzahl.value).toBe('3');
		await fireEvent.blur(nummer);
		expect(nummer.value).toBe('0123');
	});

	it('liest Komma und Punkt, aber keine Tausenderpunkte und keine Exponenten', () => {
		expect(alsZahl(' 7,90 ')).toBe(7.9);
		expect(alsZahl('12,50 €'), 'mit Eurozeichen').toBe(12.5);
		expect(alsZahl('€ 12,50'), 'Eurozeichen davor').toBe(12.5);
		expect(alsZahl('.5')).toBe(0.5);
		expect(alsZahl('1.234,50')).toBeNull();
		expect(alsZahl('1e3')).toBeNull();
	});
});
