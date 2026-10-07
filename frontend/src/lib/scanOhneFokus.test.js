import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { istTippzeichen, nimmtTastenAn, tasteInsScanfeld, thekenTasten } from './scanOhneFokus.js';

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

// Eine Rückfrage der Theke (Sperre, Vormerkung, Zubehör) hat den Fokus auf einem Knopf, und
// ein Scan endet mit Enter: Er drückte den Knopf, die Rückfrage war weg, gebucht war nichts.
// Den Weg im Browser prüft e2e/theke-scan-reihe.spec.js.
describe('thekenTasten: ein Scan bei offener Rückfrage', () => {
	/** @type {HTMLInputElement} */
	let feld;
	/** @type {HTMLButtonElement} */
	let knopf;
	let offen = false;
	const scanBeiRueckfrage = vi.fn();
	const amKnopf = vi.fn();
	/** @type {(e: KeyboardEvent) => void} */
	let tasten;

	beforeEach(() => {
		document.body.innerHTML =
			'<input id="omnibox-input" role="combobox" /><button id="knopf">Verstanden</button>';
		feld = /** @type {HTMLInputElement} */ (document.getElementById('omnibox-input'));
		knopf = /** @type {HTMLButtonElement} */ (document.getElementById('knopf'));
		document.elementFromPoint = () => feld;
		knopf.addEventListener('keydown', amKnopf);
		offen = false;
		vi.clearAllMocks();
		tasten = thekenTasten({
			scanfeldBereit: () => !offen,
			rueckfrageOffen: () => offen,
			scanBeiRueckfrage
		});
		window.addEventListener('keydown', tasten, true);
	});
	afterEach(() => {
		window.removeEventListener('keydown', tasten, true);
		document.body.innerHTML = '';
	});

	/** @param {HTMLElement} ziel @param {string} key @param {number} zeit */
	function druecke(ziel, key, zeit) {
		const e = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true });
		Object.defineProperty(e, 'timeStamp', { value: zeit });
		ziel.dispatchEvent(e);
		return e;
	}
	/** Tippt im Abstand eines Scanners. @param {HTMLElement} ziel @param {string} text @param {number} ab */
	const tippe = (ziel, text, ab) => [...text].forEach((key, i) => druecke(ziel, key, ab + i * 5));

	it('hält das Enter an, bevor der Knopf es bekommt, und meldet den Scan', () => {
		offen = true;
		knopf.focus();

		tippe(knopf, 'B-100045', 1000);
		const enter = druecke(knopf, 'Enter', 1040);

		expect(enter.defaultPrevented, 'das Enter drückte den Knopf').toBe(true);
		expect(amKnopf, 'acht Zeichen, kein Enter').toHaveBeenCalledTimes(8);
		expect(scanBeiRueckfrage).toHaveBeenCalledTimes(1);
		expect(document.activeElement, 'die Rückfrage behält die Tastatur').toBe(knopf);
	});

	it('zählt den Scan ganz, wenn die Rückfrage mittendrin aufgeht', () => {
		feld.focus();
		tippe(feld, 'B-100', 1000);
		// Die Antwort auf den Scan davor ist da: Die Rückfrage geht auf und nimmt den Fokus.
		offen = true;
		knopf.focus();
		tippe(knopf, '045', 1025);
		const enter = druecke(knopf, 'Enter', 1040);

		expect(enter.defaultPrevented, 'drei Zeichen im Dialog reichen allein nicht').toBe(true);
		expect(scanBeiRueckfrage).toHaveBeenCalledTimes(1);
	});

	it('lässt das Enter eines Menschen durch, auch gleich nach einem Scan', () => {
		offen = true;
		knopf.focus();
		tippe(knopf, 'B-100045', 1000);
		druecke(knopf, 'Enter', 1040);
		amKnopf.mockClear();

		const enter = druecke(knopf, 'Enter', 3000);

		expect(enter.defaultPrevented).toBe(false);
		expect(amKnopf, 'der Knopf bekommt das Enter').toHaveBeenCalledTimes(1);
		expect(scanBeiRueckfrage, 'nur der Scan davor').toHaveBeenCalledTimes(1);
	});

	it('lässt ohne Rückfrage das Enter eines Scans im Scanfeld', () => {
		feld.focus();
		tippe(feld, 'B-100045', 1000);
		const enter = druecke(feld, 'Enter', 1040);

		expect(enter.defaultPrevented, 'das Scanfeld bucht').toBe(false);
		expect(scanBeiRueckfrage).not.toHaveBeenCalled();
	});

	it('lenkt ein Zeichen weiter ins Scanfeld, wenn keine Rückfrage offen ist', () => {
		knopf.focus();
		druecke(knopf, 'B', 1000);
		expect(document.activeElement).toBe(feld);
	});
});
