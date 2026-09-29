import { describe, it, expect } from 'vitest';
import { eigentumZeile } from './exemplarEigentum.js';

describe('eigentumZeile', () => {
	it('nennt Wert und Herkunft je Quelle', () => {
		expect(
			eigentumZeile({
				eigentum: 'land',
				eigentum_herkunft: 'littera',
				littera_eigentumsvermerk: 'Land Hessen'
			})
		).toEqual({ wer: 'Land', herkunft: 'laut Littera („Land Hessen“)' });
		expect(eigentumZeile({ eigentum: 'schultraeger', eigentum_herkunft: 'hand' })).toEqual({
			wer: 'Schulträger',
			herkunft: 'von Hand gesetzt'
		});
		expect(eigentumZeile({ eigentum: 'land', eigentum_herkunft: 'bestellung' })).toEqual({
			wer: 'Land',
			herkunft: 'aus der Bestellung'
		});
		expect(eigentumZeile({ eigentum: 'land', eigentum_herkunft: 'vorgabe' })?.herkunft).toBe(
			'Vorgabe (Lernmittel)'
		);
		expect(
			eigentumZeile({ eigentum: 'schultraeger', eigentum_herkunft: 'vorgabe' })?.herkunft
		).toBe('Vorgabe (kein Lernmittel)');
	});

	it('zeigt einen Littera-Wortlaut, der das Eigentum nicht gesetzt hat', () => {
		expect(
			eigentumZeile({
				eigentum: 'schultraeger',
				eigentum_herkunft: 'vorgabe',
				littera_eigentumsvermerk: 'Förderverein'
			})?.herkunft
		).toBe('Vorgabe (kein Lernmittel) · Littera: „Förderverein“');
		expect(
			eigentumZeile({
				eigentum: 'schultraeger',
				eigentum_herkunft: 'hand',
				littera_eigentumsvermerk: 'Land Hessen'
			})?.herkunft
		).toBe('von Hand gesetzt · Littera: „Land Hessen“');
	});

	it('erfindet nichts, wenn der Server kein Eigentum liefert', () => {
		expect(eigentumZeile({})).toBeNull();
		expect(eigentumZeile({ eigentum: 'stadt', eigentum_herkunft: 'hand' })).toBeNull();
	});
});
