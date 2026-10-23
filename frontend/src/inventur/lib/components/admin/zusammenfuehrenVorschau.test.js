import { describe, it, expect } from 'vitest';
import { exemplare } from './zusammenfuehrenVorschau.svelte.js';

describe('exemplare', () => {
	it('nennt die Zahl mit Einzahl und Mehrzahl, eine fehlende als 0', () => {
		expect([1, 2, 0, undefined].map((n) => exemplare(n))).toEqual([
			'1 Exemplar',
			'2 Exemplare',
			'0 Exemplare',
			'0 Exemplare'
		]);
	});
});
