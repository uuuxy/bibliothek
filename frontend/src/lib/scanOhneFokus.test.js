import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { istTippzeichen, nimmtTastenAn, tasteInsScanfeld } from './scanOhneFokus.js';

// Die Theke lenkt ein getipptes Zeichen ins Scanfeld, wenn es sonst ins Leere fiele. Wo die
// Tastatur einem anderen gehört — Feld, Dialog, offenes Menü, laufende Buchung —, bleibt sie
// dort. Den Weg im Browser prüft e2e/kiosk-scannerfokus.spec.js; jsdom kennt keine Lage der
// Elemente, deshalb steht elementFromPoint hier als Attrappe.

/** @param {string} key @param {KeyboardEventInit} [mehr] */
const taste = (key, mehr = {}) => new KeyboardEvent('keydown', { key, ...mehr });

describe('istTippzeichen', () => {
	it('nimmt Buchstaben, Ziffern und den Bindestrich eines Barcodes', () => {
		for (const k of ['B', 'b', '7', '-']) expect(istTippzeichen(taste(k))).toBe(true);
		expect(istTippzeichen(taste('B', { shiftKey: true }))).toBe(true);
	});

	it('lässt Leertaste, Steuertasten und Kürzel beim Element', () => {
		for (const k of [' ', 'Enter', 'Tab', 'Escape', 'ArrowDown', 'Shift', 'Dead'])
			expect(istTippzeichen(taste(k))).toBe(false);
		expect(istTippzeichen(taste('c', { ctrlKey: true }))).toBe(false);
		expect(istTippzeichen(taste('c', { metaKey: true }))).toBe(false);
		expect(istTippzeichen(taste('q', { altKey: true }))).toBe(false);
	});
});

describe('nimmtTastenAn', () => {
	/** @param {string} html */
	const element = (html) => {
		document.body.innerHTML = html;
		return document.querySelector('[data-ziel]');
	};
	afterEach(() => (document.body.innerHTML = ''));

	it('Felder, Auswahllisten und Menüs behalten die Tastatur', () => {
		for (const html of [
			'<input data-ziel />',
			'<input type="date" data-ziel />',
			'<textarea data-ziel></textarea>',
			'<button role="combobox" data-ziel></button>',
			'<div role="listbox"><div role="option" tabindex="0" data-ziel></div></div>',
			'<div role="menu"><button role="menuitem" data-ziel></button></div>'
		])
			expect(nimmtTastenAn(element(html)), html).toBe(true);
	});

	it('Knöpfe, Reiter, Kästchen und die leere Seite tippen nichts', () => {
		for (const html of [
			'<button data-ziel></button>',
			'<button role="tab" data-ziel></button>',
			'<input type="checkbox" data-ziel />',
			'<main tabindex="-1" data-ziel></main>'
		])
			expect(nimmtTastenAn(element(html)), html).toBe(false);
		expect(nimmtTastenAn(null)).toBe(false);
	});
});

describe('tasteInsScanfeld', () => {
	/** @type {HTMLInputElement} */
	let feld;
	/** @type {HTMLButtonElement} */
	let knopf;

	beforeEach(() => {
		document.body.innerHTML =
			'<input id="omnibox-input" role="combobox" /><button id="knopf">Bezahlt</button><input id="grund" />';
		feld = /** @type {HTMLInputElement} */ (document.getElementById('omnibox-input'));
		knopf = /** @type {HTMLButtonElement} */ (document.getElementById('knopf'));
		document.elementFromPoint = () => feld;
	});
	afterEach(() => (document.body.innerHTML = ''));

	const bereit = () => true;

	it('holt den Fokus vom Knopf ins Scanfeld', () => {
		knopf.focus();
		tasteInsScanfeld(taste('B'), bereit);
		expect(document.activeElement).toBe(feld);
	});

	it('holt ihn auch, wenn nichts den Fokus hat', () => {
		tasteInsScanfeld(taste('4'), bereit);
		expect(document.activeElement).toBe(feld);
	});

	it('lässt ein anderes Feld tippen', () => {
		const grund = /** @type {HTMLInputElement} */ (document.getElementById('grund'));
		grund.focus();
		tasteInsScanfeld(taste('B'), bereit);
		expect(document.activeElement).toBe(grund);
	});

	it('wartet, solange eine Buchung läuft oder ein Dialog der Theke offen ist', () => {
		knopf.focus();
		tasteInsScanfeld(taste('B'), () => false);
		expect(document.activeElement).toBe(knopf);
	});

	it('bleibt weg, wenn etwas über dem Scanfeld liegt', () => {
		knopf.focus();
		document.elementFromPoint = () => document.body;
		tasteInsScanfeld(taste('B'), bereit);
		expect(document.activeElement).toBe(knopf);
	});

	it('bleibt weg, solange ein Menü offen ist', () => {
		document.body.insertAdjacentHTML('beforeend', '<div role="menu"></div>');
		knopf.focus();
		tasteInsScanfeld(taste('B'), bereit);
		expect(document.activeElement).toBe(knopf);
	});

	it('lässt Enter und Kürzel beim Knopf', () => {
		knopf.focus();
		tasteInsScanfeld(taste('Enter'), bereit);
		tasteInsScanfeld(taste('c', { ctrlKey: true }), bereit);
		expect(document.activeElement).toBe(knopf);
	});
});
