import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import BestellDetailExemplare from './BestellDetailExemplare.svelte';

/** @param {string} barcode @param {string} titelId @param {string} titel */
const stueck = (barcode, titelId, titel) => ({
	barcode_id: barcode,
	titel_id: titelId,
	titel_name: titel,
	etikett_gedruckt: true,
	ist_ausgesondert: false
});

// Die Exemplare einer Lieferung, je Titel gebündelt. Schülerbuch und Arbeitsheft heißen oft
// gleich; jedes bekommt seine eigene Gruppe mit seiner Zahl und seinen Nummern.
describe('BestellDetailExemplare', () => {
	it('bündelt nach der Kennung des Titels, nicht nach seinem Wortlaut', () => {
		const ansicht = render(BestellDetailExemplare, {
			exemplare: [
				stueck('GL-1', 't-buch', 'Green Line 3'),
				stueck('GL-3', 't-buch', 'Green Line 3'),
				stueck('GL-2', 't-heft', 'Green Line 3'),
				stueck('Z-1', 't-zahl', 'Zahlen und Größen 7')
			]
		});

		const koepfe = [...ansicht.container.querySelectorAll('p')].map((p) =>
			p.textContent?.replace(/\s+/g, ' ').trim()
		);
		expect(koepfe).toEqual(['Green Line 3 (2)', 'Green Line 3 (1)', 'Zahlen und Größen 7 (1)']);
		const nummern = [...ansicht.container.querySelectorAll('ul')].map((ul) =>
			[...ul.querySelectorAll('li')].map((li) => li.textContent?.trim())
		);
		expect(nummern).toEqual([['GL-1', 'GL-3'], ['GL-2'], ['Z-1']]);
	});
});
