import { describe, it, expect } from 'vitest';
import { scanTreffer, scanUebernehmen } from './scanTreffer.js';

// Ein gescannter ISBN-Code soll den Titel ohne weiteren Handgriff in die Übernahme legen
// (Betreiber-Entscheidung 18.09.2026: „direkt in den Warenkorb, sonst macht es ja keinen
// Sinn — ob ich den Titel suche oder die ISBN scanne, ist sonst egal").
//
// Die Kehrseite ist der Grund für diese Datei: Wer ohne Rückfrage übernimmt, darf sich
// nicht irren. Deshalb steht hier beides — wann übernommen wird UND wann ausdrücklich
// nicht.
const buch = (isbn, titel = 'Titel') => ({ isbn, titel, source: 'local' });

describe('scanTreffer', () => {
	it('nimmt den Treffer, dessen ISBN dem Scan entspricht — auch mit Bindestrichen', () => {
		const liste = [buch('978-3-06-013076-4', 'Richtig'), buch('9783060999999', 'Falsch')];
		expect(scanTreffer('9783060130764', liste)?.titel).toBe('Richtig');
	});

	it('nimmt den einzigen Treffer, auch wenn er keine ISBN trägt', () => {
		expect(scanTreffer('9783060130764', [buch('', 'Einziger')])?.titel).toBe('Einziger');
	});

	it('übernimmt NICHTS, wenn mehrere Ausgaben dieselbe ISBN tragen', () => {
		const doppelt = [buch('9783060130764', 'Ausgabe A'), buch('9783060130764', 'Ausgabe B')];
		expect(scanTreffer('9783060130764', doppelt)).toBeNull();
	});

	it('übernimmt NICHTS bei mehreren Treffern ohne passende ISBN', () => {
		expect(scanTreffer('9783060130764', [buch('111'), buch('222')])).toBeNull();
	});

	it('übernimmt NICHTS ohne Treffer — und stolpert nicht über fehlende Listen', () => {
		expect(scanTreffer('9783060130764', [])).toBeNull();
		expect(scanTreffer('9783060130764', undefined)).toBeNull();
	});

	it('nimmt bei einem Code ohne Ziffern trotzdem den einzigen Treffer', () => {
		// Ein Ausweis- oder Etikettencode („B97601826457") ist keine ISBN. Ein einziger
		// Treffer bleibt trotzdem eindeutig.
		expect(scanTreffer('B97601826457', [buch('9783060130764', 'Einziger')])?.titel).toBe(
			'Einziger'
		);
	});
});

describe('scanUebernehmen', () => {
	/** @param {any[]} treffer */
	const store = (treffer) => ({ showDropdown: true, sucheSofort: async () => treffer });

	it('sucht sofort und übergibt den eindeutigen Treffer zur Übernahme', async () => {
		const s = store([buch('9783060130764', 'Gescannt')]);
		let uebernommen = /** @type {any} */ (null);
		const direkt = await scanUebernehmen('9783060130764', s, (t) => (uebernommen = t));
		expect(direkt).toBe(true);
		expect(uebernommen?.titel).toBe('Gescannt');
		// Die Liste wird zugeklappt — sonst stünde sie über der offenen Übernahme.
		expect(s.showDropdown).toBe(false);
	});

	it('lässt die Trefferliste stehen, wenn der Scan nicht eindeutig ist', async () => {
		const s = store([buch('111', 'A'), buch('222', 'B')]);
		let uebernommen = /** @type {any} */ (null);
		const direkt = await scanUebernehmen('9783060130764', s, (t) => (uebernommen = t));
		expect(direkt).toBe(false);
		expect(uebernommen).toBeNull();
		expect(s.showDropdown).toBe(true);
	});
});
