import { describe, it, expect } from 'vitest';
import { kommtHinzu } from './zusammenfuehrenVorschau.svelte.js';

/** @param {Record<string, any>} anders */
const vorschau = (anders) => ({
	exemplare: 0,
	vormerkungen: 0,
	vormerkungen_doppelt: 0,
	klassensaetze: 0,
	bestellpositionen: 0,
	anliegen: 0,
	uebernommen: [],
	...anders
});

describe('kommtHinzu', () => {
	it('nennt nur, was wandert, mit Einzahl und Mehrzahl, dann die übernommenen Felder', () => {
		expect(
			kommtHinzu(
				vorschau({
					exemplare: 58,
					vormerkungen: 2,
					vormerkungen_doppelt: 1,
					anliegen: 1,
					uebernommen: [
						{ bezeichnung: 'ISBN', wert: '9783060158423' },
						{ bezeichnung: 'Cover', wert: '' }
					]
				})
			)
		).toEqual([
			'58 Exemplare — Nummer, Etikett und Ausleihen bleiben',
			'2 Vormerkungen; 1 doppelte fällt weg',
			'1 Meldung von Lehrkräften',
			'ISBN: 9783060158423',
			'Cover'
		]);
	});

	it('bleibt leer, wenn nichts wandert', () => {
		expect(kommtHinzu(vorschau({}))).toEqual([]);
	});

	it('zählt Klassensätze und Bestellungen', () => {
		expect(
			kommtHinzu(
				vorschau({ exemplare: 1, klassensaetze: 2, bestellpositionen: 1, vormerkungen: 1 })
			)
		).toEqual([
			'1 Exemplar — Nummer, Etikett und Ausleihen bleiben',
			'1 Vormerkung',
			'2 Klassensatz-Reservierungen',
			'1 Position einer Bestellung'
		]);
	});
});
