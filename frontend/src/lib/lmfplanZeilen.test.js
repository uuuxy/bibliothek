import { describe, it, expect } from 'vitest';
import {
	einordnen,
	nachbarZeile,
	klassenTeile,
	verschiebe,
	zusammenlegen
} from './lmfplanZeilen.js';

// Die Nachbar-Regel (06.09.2026): Eine Klasse, die in den Plan kommt, landet hinter der
// letzten Klasse desselben Jahrgangs und Zweigs — nicht am Ende, von wo man sie durch
// sechzig Zeilen schieben müsste. Das ist die Antwort auf „es steht oben was und unten
// was" (Peter), nach dem Vorbild von Google Forms und Slides (Neues kommt hinter das,
// woran man arbeitet).
const z = (/** @type {string[]} */ ...klassen) => ({ klassen, vermerk: '', fest: null });

describe('lmfplanZeilen.einordnen', () => {
	const plan = [z('10R1', '10R2'), z('09H1'), z('06F1'), z('06F2'), z('06G1'), z('05F1')];

	it('setzt die Klasse hinter die letzte des gleichen Jahrgangs und Zweigs', () => {
		const { zeilen, index } = einordnen(plan, '06F3');
		expect(index).toBe(4);
		expect(zeilen[4].klassen).toEqual(['06F3']);
		expect(zeilen[3].klassen).toEqual(['06F2']);
		expect(zeilen[5].klassen).toEqual(['06G1']);
	});

	it('ohne gleichen Zweig hinter die letzte des Jahrgangs', () => {
		const { index } = einordnen(plan, '06H1');
		expect(index).toBe(5); // hinter 06G1
	});

	it('ohne Jahrgang im Plan ans Ende — auch Klassen ohne Jahrgangszahl', () => {
		expect(einordnen(plan, '08G1').index).toBe(plan.length);
		expect(einordnen(plan, 'ET1').index).toBe(plan.length);
	});

	it('findet den Jahrgang auch in einer geteilten Stunde und ohne führende Null', () => {
		expect(nachbarZeile(plan, '10R3')).toBe(0);
		expect(nachbarZeile(plan, '5f2')).toBe(5);
		expect(klassenTeile('5f2')).toEqual({ jahrgang: 5, zweig: 'F' });
	});

	it('nimmt eine ausdrückliche Stelle vor der Regel (Ziehen auf eine Zeile)', () => {
		const { zeilen, index } = einordnen(plan, '06F3', 1);
		expect(index).toBe(1);
		expect(zeilen[1].klassen).toEqual(['06F3']);
		expect(zeilen).toHaveLength(plan.length + 1);
	});

	it('verändert die Eingabe nicht', () => {
		const vorher = JSON.stringify(plan);
		einordnen(plan, '06F3');
		verschiebe(plan, 0, 3);
		zusammenlegen(plan, 1);
		expect(JSON.stringify(plan)).toBe(vorher);
	});
});

describe('lmfplanZeilen.verschiebe', () => {
	it('an den Anfang und ans Ende — die weiten Wege aus dem Zeilenmenü', () => {
		const plan = [z('A'), z('B'), z('C'), z('D')];
		expect(verschiebe(plan, 3, 0).map((x) => x.klassen[0])).toEqual(['D', 'A', 'B', 'C']);
		expect(verschiebe(plan, 0, 3).map((x) => x.klassen[0])).toEqual(['B', 'C', 'D', 'A']);
		expect(verschiebe(plan, 1, 9)).toBe(plan);
	});
});
