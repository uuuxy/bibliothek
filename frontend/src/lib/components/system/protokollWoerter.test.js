import { describe, it, expect } from 'vitest';
import { BEREICHE, VERWALTUNG_AKTIONEN, kurzeKennung } from './protokollWoerter.js';

describe('protokollWoerter', () => {
	it('kürzt eine UUID auf acht Zeichen und lässt alles andere stehen', () => {
		expect(kurzeKennung('3f2a9c1e-7b4d-4e8a-9f00-1234567890ab')).toBe('3f2a9c1e');
		expect(kurzeKennung('B-01234')).toBe('B-01234');
		expect(kurzeKennung('')).toBe('');
	});

	it('nennt Tabellen und Aktionen mit den Wörtern der Oberfläche', () => {
		expect(BEREICHE.buecher_titel).toBe('Titel');
		expect(BEREICHE.schueler).toBe('Leser');
		expect(VERWALTUNG_AKTIONEN.USER_CREATE).toBe('Benutzer angelegt');
		// Kein Eintrag bleibt roh oder englisch.
		for (const text of [...Object.values(BEREICHE), ...Object.values(VERWALTUNG_AKTIONEN)]) {
			expect(text).not.toMatch(/_|^[A-Z ]+$/);
		}
	});
});
