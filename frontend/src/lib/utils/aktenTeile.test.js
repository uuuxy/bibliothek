import { describe, it, expect } from 'vitest';
import { leseTeil, nenneFehlende } from './aktenTeile.js';
import { serverAntwort } from '../serverAntwort.js';

describe('leseTeil', () => {
	it('liefert den Wert eines gekommenen Teils ohne Grund', async () => {
		const teil = await leseTeil({ status: 'fulfilled', value: serverAntwort(200, [1, 2]) });
		expect(teil).toEqual({ wert: [1, 2], grund: '' });
	});

	it('liefert bei einer Fehlantwort den Grund des Servers', async () => {
		const teil = await leseTeil({
			status: 'fulfilled',
			value: serverAntwort(503, { error: 'Datenbank nicht erreichbar' })
		});
		expect(teil).toEqual({ wert: null, grund: 'Datenbank nicht erreichbar' });
	});

	it('liefert ohne Antwort den Satz des Abrufs', async () => {
		const teil = await leseTeil({
			status: 'rejected',
			reason: new Error('Keine Verbindung zum Server.')
		});
		expect(teil).toEqual({ wert: null, grund: 'Keine Verbindung zum Server.' });
	});
});

describe('nenneFehlende', () => {
	it('nennt einen Grund für mehrere Listen einmal', () => {
		const grund = 'Datenbank nicht erreichbar';
		expect(
			nenneFehlende([
				{ name: 'Gebühren', grund },
				{ name: 'Bescheide', grund }
			])
		).toBe('Gebühren, Bescheide (Datenbank nicht erreichbar)');
	});

	it('nennt verschiedene Gründe je Liste', () => {
		expect(
			nenneFehlende([
				{ name: 'Ausleiher', grund: 'Fehler 500' },
				{ name: 'Historie', grund: 'Keine Verbindung zum Server.' }
			])
		).toBe('Ausleiher (Fehler 500), Historie (Keine Verbindung zum Server.)');
	});
});
