import { describe, it, expect } from 'vitest';
import { nurGeaendertes } from './geaendert.js';

describe('nurGeaendertes', () => {
	const geladen = { name: 'Buchhandlung', email: 'a@haendler.example', haupt: true, zweit: '' };

	it('nennt nur, was sich seit dem Öffnen geändert hat', () => {
		expect(nurGeaendertes(geladen, { ...geladen, email: 'b@haendler.example' })).toEqual({
			email: 'b@haendler.example'
		});
	});

	it('ohne Änderung ist das Ergebnis leer', () => {
		expect(nurGeaendertes(geladen, { ...geladen })).toEqual({});
	});

	it('ein geleertes Feld und ein abgewählter Schalter gehen ausdrücklich mit', () => {
		const vorher = { ...geladen, zweit: 'S-1' };
		expect(nurGeaendertes(vorher, { ...vorher, haupt: false, zweit: '' })).toEqual({
			haupt: false,
			zweit: ''
		});
	});

	// Verglichen werden die Namen der Maske: Was sie nicht führt, kann sie nicht geändert haben.
	it('ein Feld, das nur der geladene Stand kennt, geht nicht mit', () => {
		expect(nurGeaendertes({ ...geladen, id: 'l1' }, { name: 'Buchhandlung' })).toEqual({});
	});
});
