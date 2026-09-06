import { describe, it, expect } from 'vitest';
import {
	einordnen,
	nachbarZeile,
	festWechseln,
	klassenTeile,
	klasseTauschen,
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

// bewusstDraussen: Gespeicherte Auslassungen UND die Regel der Art klappen ein — auch bei
// einem laufenden Plan, der die Klassen nie kannte (06.09.2026: ein Plan mit alten
// Klassennamen bot sonst 60 Chips offen an). Was keine Regel hat, bleibt offen.
describe('lmfplanDienst.bewusstDraussen', () => {
	it('klappt gespeicherte Auslassungen und die Regel ein, lässt den Rest offen', async () => {
		const { bewusstDraussen } = await import('./lmfplanDienst.js');
		const laufend = bewusstDraussen(
			/** @type {any} */ ({
				plan: { id: 'p' },
				vorbei: false,
				ausgelassen: ['12T1'],
				vorschlag: { ausgelassen: ['13T1'] },
				ausgelassen_regel: ['06F1', 'ET1']
			})
		);
		expect(laufend('12T1')).toBe(true); // gespeichert
		expect(laufend('6f1')).toBe(true); // Regel, über den Normschlüssel
		expect(laufend('13T1')).toBe(false); // Vorschlag zählt bei laufendem Plan nicht
		expect(laufend('05F1')).toBe(false); // keine Regel: offen
		const neu = bewusstDraussen(
			/** @type {any} */ ({ plan: null, vorschlag: { ausgelassen: ['13T1'] } })
		);
		expect(neu('13T1')).toBe(true);
		expect(neu('05F1')).toBe(false);
	});
});

describe('lmfplanZeilen.klasseTauschen', () => {
	// Klick auf die Klasse in der Tabelle (06.09.2026): die Zelle überschreiben, Platz
	// und Vermerk bleiben — auch in einer geteilten Stunde nur die eine Klasse.
	const plan = [
		{ klassen: ['10R1', '10R2'], vermerk: 'zusammen', fest: { datum: '2027-06-28', stunde: 3 } },
		z('09H1')
	];

	it('ersetzt genau die eine Klasse und lässt den Rest der Zeile', () => {
		const neu = klasseTauschen(plan, 0, '10R2', '10R4');
		expect(neu[0]).toEqual({ ...plan[0], klassen: ['10R1', '10R4'] });
		expect(neu[1]).toBe(plan[1]);
		expect(plan[0].klassen).toEqual(['10R1', '10R2']);
	});

	it('tut nichts, wenn die alte fehlt oder die neue schon in der Zeile steht', () => {
		expect(klasseTauschen(plan, 0, '09H1', '10R4')).toBe(plan);
		expect(klasseTauschen(plan, 0, '10R1', '10R2')).toBe(plan);
		expect(klasseTauschen(plan, 5, '10R1', '10R4')).toBe(plan);
	});
});

describe('lmfplanDienst.klasseTauschen', () => {
	it('nimmt die neue Klasse aus dem Vorrat und legt die alte dorthin', async () => {
		const { klasseTauschen: tausche, leererEntwurf } = await import('./lmfplanDienst.js');
		const e = { ...leererEntwurf(), zeilen: [z('10R1'), z('09H1')], ausgelassen: ['10R4', '12T1'] };
		const neu = tausche(e, 0, '10R1', '10R4');
		expect(neu.zeilen.map((x) => x.klassen)).toEqual([['10R4'], ['09H1']]);
		expect(neu.ausgelassen).toEqual(['10R1', '12T1']);
		// Gleiche Klasse oder eine, die nicht in der Zeile steht: unverändert.
		expect(tausche(e, 0, '10R1', '10r1')).toBe(e);
		expect(tausche(e, 1, '10R1', '10R4')).toBe(e);
	});
});

describe('lmfplanZeilen.festWechseln', () => {
	// „Fest ohne Datum" kannte nur das Frontend: Der Server nimmt zwei Zustände (fließt,
	// oder fester Platz MIT Datum) und antwortet sonst 400. Das Fenster ohne Plätze ist
	// echt — nach jedem Laden, bis die erste Vorschau da ist (Rasterdurchgang 06.09.2026).
	const zeilen = [z('10R1'), z('09H1')];

	it('legt ohne gerechneten Platz nichts fest', () => {
		expect(festWechseln(zeilen, 0, undefined)).toBe(zeilen);
		expect(festWechseln(zeilen, 0, { datum: '', stunde: 1 })).toBe(zeilen);
	});

	it('legt mit Platz fest und löst wieder — auch ohne Platz (Gegenprobe)', () => {
		const fest = festWechseln(zeilen, 0, { datum: '2027-06-28', stunde: 3 });
		expect(fest[0].fest).toEqual({ datum: '2027-06-28', stunde: 3 });
		expect(festWechseln(fest, 0, undefined)[0].fest).toBeNull();
	});
});
