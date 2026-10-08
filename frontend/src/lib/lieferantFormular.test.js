import { describe, it, expect } from 'vitest';
import { lieferantStand } from './lieferantFormular.js';
import { nurGeaendertes } from './utils/geaendert.js';

const HAENDLER = {
	id: 'l1',
	name: 'Buchhandlung',
	email: 'bestellung@haendler.example',
	customerNumber: 'K-1',
	ist_hauptlieferant: true
};

describe('lieferantFormular', () => {
	it('der Stand trägt die Namen der Anfrage; eine fehlende zweite Nummer ist leer', () => {
		expect(lieferantStand(HAENDLER)).toEqual({
			name: 'Buchhandlung',
			email: 'bestellung@haendler.example',
			customerNumber: 'K-1',
			ist_hauptlieferant: true,
			kundennummer_schultraeger: ''
		});
	});

	// Die Zeile füllt ihre Maske aus der Liste. Schickte sie jedes Feld zurück, nähme das
	// Korrigieren der E-Mail einem anderen Händler das Merkmal, das er inzwischen bekommen hat.
	it('eine geänderte E-Mail schickt nur die E-Mail, nicht das Merkmal', () => {
		const geladen = lieferantStand(HAENDLER);
		expect(nurGeaendertes(geladen, { ...geladen, email: 'neu@haendler.example' })).toEqual({
			email: 'neu@haendler.example'
		});
	});
});
