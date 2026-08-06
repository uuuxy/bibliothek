import { describe, it, expect, beforeEach } from 'vitest';
import {
	idStore,
	resetDesign,
	applyDesign,
	defaultFrontElements,
	wendeSchulstammdatenAn,
	PLATZHALTER_SCHULNAME,
	PLATZHALTER_ADRESSE
} from './idDesignerStore.svelte.js';

function header() {
	return idStore.front.elements.find((e) => e.id === 'header');
}
function address() {
	return idStore.front.elements.find((e) => e.id === 'address');
}

describe('wendeSchulstammdatenAn', () => {
	beforeEach(() => {
		resetDesign();
	});

	it('ersetzt den Platzhalter durch die echten Schul-Stammdaten', () => {
		wendeSchulstammdatenAn('Philipp-Reis-Schule', 'Schulstraße 1, 61476 Kronberg');

		expect(header().content).toBe('Philipp-Reis-Schule');
		expect(address().content).toBe('Schulstraße 1, 61476 Kronberg');
	});

	it('rührt einen bereits selbst eingetragenen Kopf nicht an', () => {
		header().content = 'Von Hand eingetragener Schulname';

		wendeSchulstammdatenAn('Philipp-Reis-Schule', 'Schulstraße 1, 61476 Kronberg');

		expect(header().content).toBe('Von Hand eingetragener Schulname');
		// Die Adresse ist weiterhin der Platzhalter, weil sie einzeln geprüft wird —
		// ein bearbeiteter Kopf darf die Adresse nicht mit heilen.
		expect(address().content).toBe('Schulstraße 1, 61476 Kronberg');
	});

	it('lässt den Platzhalter stehen, wenn die Einstellungen selbst leer sind', () => {
		wendeSchulstammdatenAn('', '');

		expect(header().content).toBe(PLATZHALTER_SCHULNAME);
		expect(address().content).toBe(PLATZHALTER_ADRESSE);
	});

	it('ist ein no-op ohne Header-/Adress-Element auf der Seite (defensiv)', () => {
		idStore.front.elements = idStore.front.elements.filter((e) => e.id !== 'header');
		expect(() => wendeSchulstammdatenAn('Philipp-Reis-Schule', 'X')).not.toThrow();
	});
});

// Die Klassenzeile gehört nicht auf den Ausweis: Eine Karte, die die Klasse trägt,
// wäre nach jedem Schuljahreswechsel falsch und müsste neu gedruckt werden — für ein
// Dokument, das sonst die ganze Schulzeit gilt, ist das sinnlos. Die Klasse steht im
// System, wo sie sich ohne Nachdruck ändert.
//
// Geprüft wird nicht die Vorgabe allein, sondern der Ladeweg: Das Design ist ZENTRAL
// gespeichert, und applyDesign() überschreibt die Vorgaben mit dem Stand aus der
// Datenbank. Eine bestehende Installation trüge das Element sonst für immer weiter —
// derselbe Weg, auf dem der Musterstadt-Kopf hängenblieb.
describe('Klassenzeile auf dem Ausweis', () => {
	beforeEach(() => {
		resetDesign();
	});

	it('kommt in den Standardwerten nicht mehr vor', () => {
		expect(defaultFrontElements().some((e) => e.type === 'details')).toBe(false);
	});

	it('wird aus einem zentral gespeicherten Altdesign beim Laden entfernt', () => {
		applyDesign({
			front: {
				elements: [
					{ id: 'name', type: 'name', content: '', show: true },
					{ id: 'details', type: 'details', content: '', show: true },
					{ id: 'barcode', type: 'barcode', content: '', show: true }
				],
				theme: 'bg-white text-black border-slate-200'
			}
		});

		expect(idStore.front.elements.some((e) => e.type === 'details')).toBe(false);
		// Die übrigen Elemente bleiben unangetastet — gefiltert wird genau eines.
		expect(idStore.front.elements.map((e) => e.id)).toEqual(['name', 'barcode']);
	});
});
