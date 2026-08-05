import { describe, it, expect, beforeEach } from 'vitest';
import {
	idStore,
	resetDesign,
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
