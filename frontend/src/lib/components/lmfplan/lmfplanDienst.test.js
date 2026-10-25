import { describe, it, expect } from 'vitest';
import { artErklaerung, entwurfAus } from './lmfplanDienst.js';

/** @typedef {import('./lmfplanDienst.js').PlanStand} PlanStand */

describe('entwurfAus: „Nicht im Plan“', () => {
	it('sortiert nach Jahrgang als Zahl, nicht nach dem ersten Zeichen', () => {
		/** @type {PlanStand} */
		const stand = {
			plan: null,
			zeilen: [],
			ausgelassen: [],
			vorbei: false,
			vorschlag: { quelle: 'regel', zeilen: [], ausgelassen: ['10R1', '9H2', '5F'] },
			klassen: ['12T', '7G1']
		};

		// Als Text sortiert stünde „10R1“ vor „5F“.
		expect(entwurfAus(stand).ausgelassen).toEqual(['5F', '7G1', '9H2', '10R1', '12T']);
	});

	it('lässt die Liste des Vorschlags unverändert stehen', () => {
		const ausgelassen = ['9H2', '5F'];
		/** @type {PlanStand} */
		const stand = {
			plan: null,
			zeilen: [],
			ausgelassen: [],
			vorbei: false,
			vorschlag: { quelle: 'regel', zeilen: [], ausgelassen },
			klassen: []
		};

		entwurfAus(stand);

		expect(ausgelassen).toEqual(['9H2', '5F']);
	});
});

describe('artErklaerung', () => {
	it('zählt die Eingangsjahrgänge mit Komma und „und“ auf', () => {
		expect(artErklaerung('ausgabe', [5])).toContain('(Jahrgang 5)');
		expect(artErklaerung('ausgabe', [5, 7])).toContain('(Jahrgang 5 und 7)');
		expect(artErklaerung('ausgabe', [5, 7, 11])).toContain('(Jahrgang 5, 7 und 11)');
	});
});
