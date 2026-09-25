import { describe, it, expect } from 'vitest';
import { auflagenBeschriftung } from './auflagenText.js';

// Wie eine Auflage in Listen heißt: Titel derselben Reihe unterscheiden sich oft nur in
// Auflage und Jahr (docs/OFFEN.md 4.18).
describe('auflagenBeschriftung', () => {
	it('nennt Auflage und Jahr', () => {
		expect(auflagenBeschriftung({ auflage: '4. Aufl.', erscheinungsjahr: 2023 })).toBe(
			'4. Aufl. · 2023'
		);
	});

	it('nennt das Jahr nicht zweimal, wenn es schon in der Auflage steht', () => {
		expect(auflagenBeschriftung({ auflage: '4. Aufl. 2023', erscheinungsjahr: 2023 })).toBe(
			'4. Aufl. 2023'
		);
	});

	it('kommt mit nur einem der beiden aus — und mit keinem', () => {
		expect(auflagenBeschriftung({ auflage: ' 2. Auflage ', erscheinungsjahr: 0 })).toBe(
			'2. Auflage'
		);
		expect(auflagenBeschriftung({ erscheinungsjahr: 2019 })).toBe('Ausgabe 2019');
		expect(auflagenBeschriftung({})).toBe('Ohne Angabe zur Auflage');
	});
});
