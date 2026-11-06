import { describe, it, expect } from 'vitest';
import { passendeVorschlaege, vorschlagTaste } from './vorschlagRegeln.js';

const werte = (/** @type {{ wert: string }[]} */ liste) => liste.map((v) => v.wert);

describe('passendeVorschlaege', () => {
	// „Ask“ steht vor „Sk“: Nur so zeigt sich, dass vorn Passendes nach vorn rückt.
	const signaturen = [{ wert: 'Ask' }, { wert: 'Sk' }, { wert: 'Sdk 2' }, { wert: 'Pg' }];

	it('zeigt ohne Text alle in der Reihenfolge des Aufrufers', () => {
		expect(werte(passendeVorschlaege(signaturen, ''))).toEqual(['Ask', 'Sk', 'Sdk 2', 'Pg']);
	});

	it('stellt die Werte, die mit dem Text beginnen, vor die, die ihn nur enthalten', () => {
		expect(werte(passendeVorschlaege(signaturen, 'sk'))).toEqual(['Sk', 'Ask']);
		expect(werte(passendeVorschlaege(signaturen, 'k'))).toEqual(['Ask', 'Sk', 'Sdk 2']);
	});

	it('übergeht Groß- und Kleinschreibung und Leerzeichen am Rand', () => {
		expect(werte(passendeVorschlaege(signaturen, '  SDK '))).toEqual(['Sdk 2']);
	});

	it('nennt jeden Wert nur einmal, weil die Liste ihre Zeilen nach ihm schlüsselt', () => {
		const doppelt = [
			{ wert: 'Sk', zusatz: '3 Titel' },
			{ wert: 'Sk', zusatz: '9 Titel' }
		];
		expect(passendeVorschlaege(doppelt, '')).toEqual([{ wert: 'Sk', zusatz: '3 Titel' }]);
	});

	it('gibt ohne Treffer eine leere Liste zurück', () => {
		expect(passendeVorschlaege(signaturen, 'xyz')).toEqual([]);
	});
});

describe('vorschlagTaste', () => {
	it.each([
		// taste, offen, aktiv, anzahl, tat, verhindern
		['ArrowDown', false, -1, 3, 'oeffnen', true],
		['ArrowDown', false, -1, 0, 'nichts', false],
		['ArrowDown', true, 0, 3, 'wandern', true],
		['ArrowUp', true, 0, 3, 'wandern', true],
		['ArrowUp', false, -1, 3, 'nichts', false],
		['Enter', true, 1, 3, 'waehlen', true],
		['Enter', true, -1, 3, 'nichts', false],
		['Enter', false, -1, 3, 'nichts', false],
		['Escape', true, -1, 3, 'schliessen', true],
		['Escape', false, -1, 3, 'nichts', false],
		['Tab', true, 0, 3, 'schliessen', false],
		['a', true, 0, 3, 'nichts', false]
	])('%s (offen %s, aktiv %i, %i Treffer) → %s', (taste, offen, aktiv, anzahl, tat, verhindern) => {
		const befehl = vorschlagTaste(taste, offen, aktiv, anzahl);
		expect(befehl.tat).toBe(tat);
		expect(befehl.verhindern).toBe(verhindern);
	});

	it('wandert abwärts mit +1 und aufwärts mit −1', () => {
		expect(vorschlagTaste('ArrowDown', true, 0, 3).richtung).toBe(1);
		expect(vorschlagTaste('ArrowUp', true, 0, 3).richtung).toBe(-1);
	});
});
