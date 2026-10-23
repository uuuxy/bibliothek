import { describe, it, expect } from 'vitest';
import { AUSWEIS_VORLAGEN, vorlage } from './ausweisVorlagen.js';
import { idStore, resetDesign } from './idDesignerStore.svelte.js';
import { SICHERHEITSRAND_MM, STRICHCODE_RAND_MM } from './kartenRand.js';
import { NUMMERNZEILE_MM } from '../strichcodeBild.js';

// ISO 7810 ID-1 — dieselben Maße, die der Designer und der Kartendruck verwenden.
const KARTE_BREITE = 85.6;
const KARTE_HOEHE = 53.98;

// Farbflächen und Motive über die ganze Breite reichen mit Absicht bis an den Rand.
const reichtBisAnDenRand = (/** @type {any} */ el) =>
	el.type === 'box' || (el.type === 'image' && el.width >= KARTE_BREITE);

// Auf hundertstel Millimeter: 85,6 − 27 − 55,6 ergibt in Gleitkomma 2,999…, nicht 3.
const mm = (/** @type {number} */ wert) => Math.round(wert * 100) / 100;

/**
 * Was zu nah am Rand steht, als „Seite Kennung: Abstand".
 * @param {string} seite
 * @param {any[]} elemente
 */
function zuNahAmRand(seite, elemente) {
	const funde = [];
	for (const el of elemente) {
		if (!el.show || reichtBisAnDenRand(el)) continue;
		const rechts = KARTE_BREITE - el.x - el.width;
		const unten = KARTE_HOEHE - el.y - el.height;
		const abstand = mm(Math.min(el.x, el.y, rechts, unten));
		if (abstand < SICHERHEITSRAND_MM) funde.push(`${seite} ${el.id}: ${abstand.toFixed(2)} mm`);
		// Unter den Strichen steht die Nummernzeile; die Striche enden um sie höher.
		const striche = mm(Math.min(el.x, el.y, rechts, unten + NUMMERNZEILE_MM));
		if (el.type === 'barcode' && striche < STRICHCODE_RAND_MM)
			funde.push(`${seite} ${el.id} (Striche): ${striche.toFixed(2)} mm`);
	}
	return funde;
}

/**
 * Abstand des Äußersten zum Kartenrand, links und rechts.
 * @param {any[]} elemente
 */
function aussenRand(elemente) {
	const gedruckt = elemente.filter((el) => el.show && !reichtBisAnDenRand(el));
	return {
		links: mm(Math.min(...gedruckt.map((el) => el.x))),
		rechts: mm(KARTE_BREITE - Math.max(...gedruckt.map((el) => el.x + el.width)))
	};
}

describe('Ausweis: alles Gedruckte innerhalb des sicheren Rands', () => {
	it('das Standard-Design', () => {
		resetDesign();
		expect([
			...zuNahAmRand('vorn', idStore.front.elements),
			...zuNahAmRand('hinten', idStore.back.elements)
		]).toEqual([]);
	});

	it.each(AUSWEIS_VORLAGEN.map((v) => v.value))('die Vorlage %s', (kennung) => {
		const v = /** @type {NonNullable<ReturnType<typeof vorlage>>} */ (vorlage(kennung));
		expect([
			...zuNahAmRand('vorn', v.front.elements),
			...zuNahAmRand('hinten', v.back.elements)
		]).toEqual([]);
	});
});

// Alle Designs haben denselben Rand: Was links und rechts außen steht, steht an der Linie.
describe('Ausweis: bündig an der Linie des sicheren Rands', () => {
	it.each(['standard', ...AUSWEIS_VORLAGEN.map((v) => v.value)])('%s', (kennung) => {
		resetDesign();
		const v = kennung === 'standard' ? idStore : vorlage(kennung);
		const rand = { links: SICHERHEITSRAND_MM, rechts: SICHERHEITSRAND_MM };
		expect({
			vorn: aussenRand(v?.front.elements ?? []),
			hinten: aussenRand(v?.back.elements ?? [])
		}).toEqual({ vorn: rand, hinten: rand });
	});
});
