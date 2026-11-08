import { describe, it, expect } from 'vitest';
import { AUSWEIS_VORLAGEN, vorlage } from './ausweisVorlagen.js';
import { idStore, resetDesign } from './idDesignerStore.svelte.js';
import {
	FANGWEITE_MM,
	KARTE_BREITE_MM as KARTE_BREITE,
	KARTE_HOEHE_MM as KARTE_HOEHE,
	SICHERHEITSRAND_MM,
	STRICHCODE_RAND_MM,
	rasteEin,
	reichtBisAnDenRand,
	sichererAbstand,
	stehtImRand
} from './kartenRand.js';
import { NUMMERNZEILE_MM } from '../utils/strichcodeBild.js';

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

// Farbflächen und Motive über die ganze Breite reichen mit Absicht bis an den Rand; alles andere
// hält 3 mm, die Striche eines Strichcodes 6,3 mm, unten zählt die Nummernzeile mit.
describe('sichererAbstand: was jede Kante halten muss', () => {
	const text = { type: 'text', width: 20 };
	it('Text, Bild und Logo: 3 mm an jeder Kante', () => {
		const r = SICHERHEITSRAND_MM;
		for (const el of [text, { type: 'image', width: 20 }, { type: 'logo', width: 15 }]) {
			expect(sichererAbstand(el)).toEqual({ links: r, oben: r, rechts: r, unten: r });
		}
	});
	it('Strichcode: 6,3 mm, unten um die Nummernzeile weniger', () => {
		const s = STRICHCODE_RAND_MM;
		expect(sichererAbstand({ type: 'barcode', width: 30 })).toEqual({
			links: s,
			oben: s,
			rechts: s,
			unten: s - NUMMERNZEILE_MM
		});
	});
	it('Farbfläche und Motiv über die ganze Breite: kein Abstand', () => {
		expect(reichtBisAnDenRand({ type: 'box', width: 10 })).toBe(true);
		expect(sichererAbstand({ type: 'box', width: 10 })).toBeNull();
		expect(sichererAbstand({ type: 'image', width: KARTE_BREITE })).toBeNull();
		expect(reichtBisAnDenRand(text)).toBe(false);
	});
});

describe('stehtImRand: zu nah am Kartenrand', () => {
	const el = (/** @type {object} */ lage) => ({
		type: 'text',
		x: 10,
		y: 10,
		width: 20,
		height: 5,
		...lage
	});
	it('an der Linie nicht, 0,1 mm darüber schon, an jeder Kante', () => {
		expect(stehtImRand(el({ x: SICHERHEITSRAND_MM }))).toBe(false);
		expect(stehtImRand(el({ x: 2.9 }))).toBe(true);
		expect(stehtImRand(el({ y: 2.9 }))).toBe(true);
		expect(stehtImRand(el({ x: KARTE_BREITE - 20 - 2.9 }))).toBe(true);
		expect(stehtImRand(el({ y: KARTE_HOEHE - 5 - 2.9 }))).toBe(true);
		// 85,6 − 3 − 20 ergibt in Gleitkomma nicht genau 62,6; an der Linie ist trotzdem nicht darüber.
		expect(stehtImRand(el({ x: KARTE_BREITE - SICHERHEITSRAND_MM - 20 }))).toBe(false);
	});
	it('ein Strichcode 4 mm vom Rand: innerhalb der Linie, für die Striche zu nah', () => {
		expect(stehtImRand({ type: 'barcode', x: 4, y: 10, width: 30, height: 11 })).toBe(true);
		expect(stehtImRand({ type: 'barcode', x: 6.3, y: 10, width: 30, height: 11 })).toBe(false);
	});
	it('eine Farbfläche bis an den Rand: nie', () => {
		expect(stehtImRand({ type: 'box', x: 0, y: 0, width: KARTE_BREITE, height: 8 })).toBe(false);
	});
});

// Nah an der Linie rastet die Kante ein, darüber hinaus läuft das Element frei weiter.
describe('rasteEin: Widerstand an der Linie', () => {
	const text = (/** @type {number} */ x, y = 20) => ({ type: 'text', x, y, width: 20, height: 5 });
	it('links: innerhalb der Fangweite genau auf der Linie, von beiden Seiten', () => {
		expect(rasteEin(text(3.8))).toEqual({ x: SICHERHEITSRAND_MM, y: 20, eingerastet: true });
		expect(rasteEin(text(2.2))).toEqual({ x: SICHERHEITSRAND_MM, y: 20, eingerastet: true });
		expect(rasteEin(text(SICHERHEITSRAND_MM - FANGWEITE_MM))).toMatchObject({
			x: SICHERHEITSRAND_MM
		});
	});
	it('weiter gezogen frei, auf ein Zehntel gerundet, bis an den Kartenrand', () => {
		expect(rasteEin(text(1.94))).toEqual({ x: 1.9, y: 20, eingerastet: false });
		expect(rasteEin(text(0))).toEqual({ x: 0, y: 20, eingerastet: false });
		expect(rasteEin(text(10.04))).toEqual({ x: 10, y: 20, eingerastet: false });
	});
	it('rechts und unten: die hintere Kante rastet ein', () => {
		const amRechtenRand = Math.round((KARTE_BREITE - SICHERHEITSRAND_MM - 20) * 100) / 100;
		expect(rasteEin(text(amRechtenRand + 0.6)).x).toBe(amRechtenRand);
		const amUnterenRand = Math.round((KARTE_HOEHE - SICHERHEITSRAND_MM - 5) * 100) / 100;
		expect(rasteEin(text(30, amUnterenRand - 0.7))).toMatchObject({
			y: amUnterenRand,
			eingerastet: true
		});
	});
	it('der Strichcode rastet an der Grenze seiner Striche ein, nicht an der gestrichelten Linie', () => {
		const code = { type: 'barcode', x: 5.8, y: 20, width: 30, height: 11 };
		expect(rasteEin(code).x).toBe(STRICHCODE_RAND_MM);
		expect(rasteEin({ ...code, x: 3.4 })).toMatchObject({ x: 3.4, eingerastet: false });
		// Die Unterkante: Unter den Strichen steht die Nummernzeile.
		const unten =
			Math.round((KARTE_HOEHE - (STRICHCODE_RAND_MM - NUMMERNZEILE_MM) - 11) * 100) / 100;
		expect(rasteEin({ ...code, x: 20, y: unten + 0.5 }).y).toBe(unten);
	});
	it('eine Farbfläche rastet nie ein', () => {
		expect(rasteEin({ type: 'box', x: 3.5, y: 2.6, width: 30, height: 8 })).toEqual({
			x: 3.5,
			y: 2.6,
			eingerastet: false
		});
	});
});
