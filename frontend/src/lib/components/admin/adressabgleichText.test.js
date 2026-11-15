import { describe, it, expect } from 'vitest';
import { letzterLaufText, ergebnisZahlen } from './adressabgleichText.js';

const stand = {
	zeitpunkt: '2026-12-01T03:30:00Z',
	eintraege: 223,
	uebernommen: 14,
	uebergangen: 3
};

describe('adressabgleichText', () => {
	it('sagt, dass noch kein Abgleich lief', () => {
		expect(letzterLaufText(null)).toBe('Noch kein Abgleich gelaufen');
	});

	it('nennt Zeitpunkt und Zahlen des letzten Laufs', () => {
		expect(letzterLaufText(stand)).toMatch(
			/^Zuletzt am 01\.12\.2026, \d\d:30: 14 Adressen übernommen, 3 Konten ohne eindeutigen Treffer$/
		);
	});

	it('nennt den Grund, wenn der letzte Lauf scheiterte', () => {
		const text = letzterLaufText({ ...stand, fehler: 'Das Webmail hat die Anmeldung abgelehnt' });
		expect(text).toMatch(/: gescheitert\. Das Webmail hat die Anmeldung abgelehnt$/);
	});

	it('spricht im Probelauf im Konjunktiv und zählt in Ein- und Mehrzahl', () => {
		expect(ergebnisZahlen({ ...stand, uebernommen: 1, uebergangen: 1 }, true)).toBe(
			'1 Konto bekäme eine Adresse, 1 Konto ohne eindeutigen Treffer'
		);
		expect(ergebnisZahlen(stand, true)).toBe(
			'14 Konten bekämen eine Adresse, 3 Konten ohne eindeutigen Treffer'
		);
		expect(ergebnisZahlen({ ...stand, uebernommen: 1 }, false)).toBe(
			'1 Adresse übernommen, 3 Konten ohne eindeutigen Treffer'
		);
	});
});
