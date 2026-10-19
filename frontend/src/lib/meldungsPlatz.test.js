import { describe, it, expect } from 'vitest';
import { meldungsPlatz, RAND, ABSTAND } from './meldungsPlatz.js';

/**
 * Ein Kasten wie aus getBoundingClientRect.
 * @param {number} top @param {number} hoehe @param {number} [right] @param {number} [breite]
 */
const kasten = (top, hoehe, right = 900, breite = 400) => ({
	top,
	bottom: top + hoehe,
	right,
	width: breite,
	height: hoehe
});

// Fenster 800 px hoch, Meldungen 48 px hoch. Am Bildschirm misst e2e/meldungen-unten.spec.js;
// hier steht die Rechnung.
const lage = { fensterHoehe: 800, hoehe: 48, navigation: [], leisten: [] };

describe('meldungsPlatz', () => {
	it('steht ohne Navigation und Leiste unten am Rand und über die volle Breite', () => {
		expect(meldungsPlatz(lage)).toEqual({ links: 0, unten: RAND });
	});

	it('beginnt rechts der Navigation, eingeklappt wie ausgeklappt', () => {
		const navigation = [kasten(0, 800, 256, 256)];
		expect(meldungsPlatz({ ...lage, navigation }).links).toBe(256);
		expect(meldungsPlatz({ ...lage, navigation: [kasten(0, 800, 64, 64)] }).links).toBe(64);
	});

	it('steigt über eine Leiste, die unten am Rand klebt', () => {
		// Auswahlleiste: 64 px hoch, 40 px über dem Fensterrand (16 px über dem Rand der
		// Arbeitsfläche, die unten 24 px Innenabstand hat).
		const leisten = [kasten(800 - 40 - 64, 64)];
		expect(meldungsPlatz({ ...lage, leisten }).unten).toBe(40 + 64 + ABSTAND);
	});

	it('steigt über eine Leiste, die nur zum Teil unter den Meldungen liegt', () => {
		// Die Liste endet knapp über dem Rand: Die Leiste steht im Fluss bei 686 bis 750 und reicht
		// mit 22 px in die Meldungen (ab 728).
		const leisten = [kasten(686, 64)];
		expect(meldungsPlatz({ ...lage, leisten }).unten).toBe(800 - 686 + ABSTAND);
	});

	it('bleibt am Rand, wenn die Leiste weiter oben im Fluss steht', () => {
		const leisten = [kasten(300, 64)];
		expect(meldungsPlatz({ ...lage, leisten }).unten).toBe(RAND);
	});

	it('steigt mit einer höheren Leiste weiter', () => {
		// Leiste der Leserdatei mit umbrochenem Hinweis: 24 px über dem Rand, 96 px hoch.
		const leisten = [kasten(800 - 24 - 96, 96)];
		expect(meldungsPlatz({ ...lage, leisten }).unten).toBe(24 + 96 + ABSTAND);
	});

	it('rechnet mit der Höhe der Meldungen: mehrere übereinander reichen höher hinauf', () => {
		// Die Leiste steht 120 px über dem Rand. Eine Meldung (48 px) reicht nicht hin, drei
		// (160 px) schon.
		const leisten = [kasten(800 - 120 - 40, 40)];
		expect(meldungsPlatz({ ...lage, leisten }).unten).toBe(RAND);
		expect(meldungsPlatz({ ...lage, hoehe: 160, leisten }).unten).toBe(120 + 40 + ABSTAND);
	});

	it('steigt über zwei Leisten, wenn die erste sie in die zweite schiebt', () => {
		// Über der geklebten Leiste (696 bis 760) steht 10 px höher eine zweite (646 bis 686).
		const leisten = [kasten(646, 40), kasten(696, 64)];
		expect(meldungsPlatz({ ...lage, leisten }).unten).toBe(800 - 646 + ABSTAND);
		// Gegenprobe: Ohne die untere bleiben die Meldungen am Rand, die obere liegt nicht im Weg.
		expect(meldungsPlatz({ ...lage, leisten: [kasten(646, 40)] }).unten).toBe(RAND);
	});

	it('übergeht ausgeblendete Elemente', () => {
		const navigation = [{ top: 0, bottom: 0, right: 0, width: 0, height: 0 }];
		const leisten = [{ top: 0, bottom: 0, right: 0, width: 0, height: 0 }];
		expect(meldungsPlatz({ ...lage, navigation, leisten })).toEqual({ links: 0, unten: RAND });
	});
});
