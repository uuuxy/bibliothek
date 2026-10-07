import { describe, it, expect } from 'vitest';
import { exemplarStatus, exemplarZahlen } from './exemplarStatus.js';

describe('exemplarStatus', () => {
	const imRegal = { ist_ausleihbar: true, ist_verfuegbar: true, im_bestand: true };

	it('nennt ein Exemplar im Regal verfügbar, ein verliehenes ausgeliehen', () => {
		expect(exemplarStatus(imRegal)).toEqual({ ton: 'erfolg', text: 'Verfügbar' });
		expect(exemplarStatus({ ...imRegal, ist_verfuegbar: false })).toEqual({
			ton: 'warten',
			text: 'Ausgeliehen'
		});
	});

	it('nennt gesperrt nur, was im Bestand steht und nicht ausleihbar ist', () => {
		expect(exemplarStatus({ ...imRegal, ist_ausleihbar: false })).toEqual({
			ton: 'fehler',
			text: 'Gesperrt'
		});
	});

	// Ein bestelltes Exemplar ist nicht ausleihbar, ein ausgesondertes auch nicht: Beide
	// hießen „Gesperrt", solange nur ist_ausleihbar gelesen wurde.
	it('nennt ein bestelltes Exemplar bestellt und ein ausgesondertes ausgesondert', () => {
		const draussen = { ist_ausleihbar: false, ist_verfuegbar: true, im_bestand: false };
		expect(exemplarStatus(draussen)).toEqual({ ton: 'neutral', text: 'Bestellt' });
		expect(exemplarStatus({ ...draussen, ist_ausgesondert: true })).toEqual({
			ton: 'neutral',
			text: 'Ausgesondert'
		});
	});

	it('zählt ein Exemplar ohne das Feld im_bestand zum Bestand', () => {
		expect(exemplarStatus({ ist_ausleihbar: false, ist_verfuegbar: true }).text).toBe('Gesperrt');
	});
});

describe('exemplarZahlen', () => {
	// Dieselbe Grenze wie die Wörter der Karten: Was „Bestellt" oder „Ausgesondert" heißt,
	// zählt nicht zum Bestand.
	it('zählt den Bestand und die bestellten getrennt, ausgesonderte in keiner der Zahlen', () => {
		const liste = [
			{ im_bestand: true },
			{ im_bestand: true, ist_ausleihbar: false },
			{ im_bestand: false },
			{ im_bestand: false, ist_ausgesondert: true }
		];
		expect(exemplarZahlen(liste)).toEqual({ bestand: 2, bestellt: 1 });
	});

	it('zählt ein Exemplar ohne das Feld im_bestand zum Bestand', () => {
		expect(exemplarZahlen([{}, { ist_ausgesondert: true }])).toEqual({ bestand: 1, bestellt: 0 });
	});
});
