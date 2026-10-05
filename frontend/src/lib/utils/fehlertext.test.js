import { describe, it, expect } from 'vitest';
import { fehlertext } from './fehlertext.js';

describe('fehlertext', () => {
	it('nimmt von einem Error die Meldung, ohne den Namen davor', () => {
		expect(fehlertext(new Error('Netz weg'))).toBe('Netz weg');
		expect(fehlertext(new TypeError('kein Feld'))).toBe('kein Feld');
	});

	it('lässt eine geworfene Zeichenkette, wie sie ist', () => {
		expect(fehlertext('QR code parse error')).toBe('QR code parse error');
		expect(fehlertext('')).toBe('');
	});

	// Die Kamera-Bibliothek wirft eigene Ausnahme-Objekte, die kein Error sind.
	it('gibt ein Objekt mit eigenem toString als dessen Text aus', () => {
		const ausnahme = { toString: () => 'D: No MultiFormat Readers were able to detect the code.' };
		expect(fehlertext(ausnahme)).toBe('D: No MultiFormat Readers were able to detect the code.');
	});

	it('gibt jeden anderen Wert aus wie String()', () => {
		expect(fehlertext(404)).toBe('404');
		expect(fehlertext(null)).toBe('null');
		expect(fehlertext(undefined)).toBe('undefined');
		expect(fehlertext({})).toBe('[object Object]');
	});
});
