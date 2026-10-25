import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { ABGESCHIRMT, stelleSperrSchildAuf } from './sperrSchild.js';

// Die Anwendung bleibt hinter dem Sperrbildschirm stehen. Ihre Bauteile hören an window und
// document auf Tasten und Zeiger: Der Router deutet Escape als „zurück an die Theke", ein
// Dialog schließt sich, ein Menü klappt zu. Am Sperrbildschirm darf davon nichts ankommen.
describe('sperrSchild', () => {
	let gesperrt = false;
	/** @type {() => void} */
	let abbauen;
	/** @type {HTMLElement} */
	let sperre;
	/** @type {HTMLInputElement} */
	let passwort;
	/** @type {HTMLButtonElement} */
	let band;
	/** Zuhörer, wie Bauteile der Anwendung sie anmelden — alle NACH dem Schild. */
	const anwendung = {
		fensterFang: vi.fn(),
		fenster: vi.fn(),
		dokument: vi.fn()
	};
	const amFeld = vi.fn();
	const amSperrbildschirm = vi.fn();

	beforeEach(() => {
		gesperrt = false;
		abbauen = stelleSperrSchildAuf(() => gesperrt);
		sperre = document.createElement('div');
		sperre.setAttribute('data-sperrbildschirm', '');
		passwort = document.createElement('input');
		sperre.append(passwort);
		band = document.createElement('button');
		document.body.append(sperre, band);
		for (const name of ['keydown', 'pointerdown']) {
			window.addEventListener(name, anwendung.fensterFang, true);
			window.addEventListener(name, anwendung.fenster);
			document.addEventListener(name, anwendung.dokument);
			passwort.addEventListener(name, amFeld);
			sperre.addEventListener(name, amSperrbildschirm);
		}
		vi.clearAllMocks();
	});

	afterEach(() => {
		for (const name of ['keydown', 'pointerdown']) {
			window.removeEventListener(name, anwendung.fensterFang, true);
			window.removeEventListener(name, anwendung.fenster);
			document.removeEventListener(name, anwendung.dokument);
		}
		abbauen();
		sperre.remove();
		band.remove();
	});

	/** @param {EventTarget} ziel @param {string} name */
	const loese = (ziel, name) => ziel.dispatchEvent(new Event(name, { bubbles: true }));

	it.each(['keydown', 'pointerdown'])(
		'ohne Sperre erreicht %s jeden Zuhörer',
		(/** @type {string} */ name) => {
			loese(passwort, name);
			expect(anwendung.fensterFang).toHaveBeenCalledTimes(1);
			expect(anwendung.fenster).toHaveBeenCalledTimes(1);
			expect(anwendung.dokument).toHaveBeenCalledTimes(1);
		}
	);

	it.each(['keydown', 'pointerdown'])(
		'gesperrt, %s im Passwortfeld: Der Sperrbildschirm bekommt es, die Anwendung an document und window nicht',
		(/** @type {string} */ name) => {
			gesperrt = true;
			loese(passwort, name);
			expect(amFeld).toHaveBeenCalledTimes(1);
			expect(amSperrbildschirm).toHaveBeenCalledTimes(1);
			expect(anwendung.fenster).not.toHaveBeenCalled();
			expect(anwendung.dokument).not.toHaveBeenCalled();
		}
	);

	it.each([
		['auf dem Seitenkörper (kein Feld hat den Fokus)', () => document.body],
		['auf einem Knopf außerhalb des Sperrbildschirms', () => band]
	])('gesperrt, Taste %s: erreicht niemanden', (_name, ziel) => {
		gesperrt = true;
		loese(ziel(), 'keydown');
		expect(anwendung.fensterFang).not.toHaveBeenCalled();
		expect(anwendung.fenster).not.toHaveBeenCalled();
		expect(anwendung.dokument).not.toHaveBeenCalled();
	});

	it('nach dem Aufschließen hört die Anwendung wieder', () => {
		gesperrt = true;
		loese(document.body, 'keydown');
		gesperrt = false;
		loese(document.body, 'keydown');
		expect(anwendung.fenster).toHaveBeenCalledTimes(1);
	});

	it('schirmt Tasten, Zeiger und Berührung ab', () => {
		for (const name of ['keydown', 'pointerdown', 'click', 'wheel', 'touchstart', 'contextmenu']) {
			expect(ABGESCHIRMT, name).toContain(name);
		}
	});
});
