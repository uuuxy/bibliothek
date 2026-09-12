import { describe, it, expect } from 'vitest';
import { berichtURL } from './bestellberichte.js';

// Der Topf gehört in die Adresse des Berichts (#596, Bauplan 7.3 Schritt 4).
//
// Die Lieferantenabrechnung ist das Blatt, gegen das eine Händlerrechnung geprüft wird.
// Kommen zwei Rechnungen — Lernmittel und Schülerbücherei getrennt —, muss das Blatt sich
// auf den Topf einengen lassen. Die Überschrift setzt der Server (EINE Beschriftung für
// Bericht, Chip und Anschreiben), hier steht nur der Parameter.

const eingabe = {
	typ: 'monat',
	monatJahr: '2026-09',
	jahr: '2026',
	vonDatum: '2026-09-01',
	bisDatum: '2026-09-30',
	lieferantId: 'l-1',
	suppliers: [{ id: 'l-1', name: 'Testhändler' }],
	mitPreisen: true,
	mittel: ''
};

/** @param {string} url */
const param = (url, name) => new URL(url, 'http://test.invalid').searchParams.get(name);

describe('berichtURL', () => {
	it('lässt den Topf weg, solange keiner gewählt ist', () => {
		expect(param(berichtURL(eingabe), 'mittel')).toBeNull();
	});

	it('trägt den gewählten Topf in jeden der drei Berichte', () => {
		for (const typ of ['monat', 'jahr', 'lieferant']) {
			const url = berichtURL({ ...eingabe, typ, mittel: 'land' });
			expect(param(url, 'mittel'), `${typ}-Bericht ohne Topf`).toBe('land');
		}
	});

	// Die Kernangaben dürfen dabei nicht verloren gehen — der Zeitraum ist der Bericht.
	it('behält Zeitraum und Lieferant', () => {
		const url = berichtURL({ ...eingabe, typ: 'lieferant', mittel: 'schultraeger' });
		expect(param(url, 'von')).toBe('2026-09-01');
		expect(param(url, 'bis')).toBe('2026-09-30');
		expect(param(url, 'lieferant_id')).toBe('l-1');
		expect(param(url, 'mittel')).toBe('schultraeger');
	});
});
