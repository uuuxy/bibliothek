import { describe, it, expect } from 'vitest';
import { vorlagenName } from './mailVorlagenInfo.js';

describe('vorlagenName', () => {
	it('nennt eine bekannte Vorlage mit dem Namen aus der Tabelle', () => {
		expect(vorlagenName('MAHNUNG_ELTERN')).toBe('Mahnbrief an die Eltern');
	});

	it('zeigt einen unbekannten Typ mit seinem Schlüssel, jeder Unterstrich als Leerzeichen', () => {
		expect(vorlagenName('GANZ_NEUE_VORLAGE')).toBe('GANZ NEUE VORLAGE');
	});
});
