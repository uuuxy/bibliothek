import { describe, it, expect, beforeEach } from 'vitest';
import {
	KARTEN_HINTERGRUENDE,
	HINTERGRUND_VORDERSEITE,
	HINTERGRUND_RUECKSEITE,
	kartenStil,
	heileKartenHintergrund
} from './kartenFarben.js';
import { idStore, applyDesign, resetDesign } from './idDesignerStore.svelte.js';
import { WALDGRUEN_THEME } from './ausweisVorlagen.js';

// Die Klassenlisten, unter denen der Hintergrund einer Kartenseite gespeichert wurde, bevor
// er eine Kennung bekam. Ein zentral gespeicherter Entwurf kann jede davon noch tragen.
const ALTE_WERTE = {
	weiss: 'bg-white text-black border-slate-200',
	grau: 'bg-slate-100 text-slate-900 border-slate-300',
	smaragd: 'bg-linear-to-tr from-emerald-100 to-teal-100 text-emerald-950 border-emerald-300',
	blau: 'bg-linear-to-tr from-sky-100 to-indigo-100 text-indigo-950 border-sky-300',
	bernstein: 'bg-linear-to-tr from-amber-100 to-orange-100 text-amber-950 border-amber-300',
	waldgruen: 'bg-linear-to-tr from-[#16330a] via-[#2d5a12] to-[#3f7418] text-white border-[#0d2405]'
};

// Fassungen aus der Zeit davor: fast weiße Flächen, die es in der Auswahl nicht mehr gibt.
const AELTERE_WERTE = [
	'bg-slate-50 text-slate-900 border-slate-200',
	'bg-linear-to-tr from-emerald-50/40 to-teal-50/40 text-zinc-900 border-emerald-100',
	'bg-linear-to-tr from-blue-50/40 to-indigo-50/40 text-zinc-900 border-blue-100'
];

describe('heileKartenHintergrund', () => {
	it('übersetzt jede früher gespeicherte Klassenliste in ihre Kennung', () => {
		for (const [kennung, alt] of Object.entries(ALTE_WERTE)) {
			expect(heileKartenHintergrund(alt), alt).toBe(kennung);
		}
	});

	it('deckt mit den alten Werten jede Kennung der Liste ab', () => {
		expect(Object.keys(ALTE_WERTE).sort()).toEqual(KARTEN_HINTERGRUENDE.map((k) => k.value).sort());
	});

	it('lässt eine Kennung, wie sie ist', () => {
		for (const k of KARTEN_HINTERGRUENDE) expect(heileKartenHintergrund(k.value)).toBe(k.value);
	});

	it('liest die älteren, fast weißen Fassungen als weiße Karte', () => {
		for (const alt of AELTERE_WERTE) expect(heileKartenHintergrund(alt), alt).toBe('weiss');
	});

	it('liest Unbekanntes als weiße Karte', () => {
		expect(heileKartenHintergrund('x')).toBe('weiss');
		expect(heileKartenHintergrund('')).toBe('weiss');
		expect(heileKartenHintergrund(undefined)).toBe('weiss');
		expect(heileKartenHintergrund(7)).toBe('weiss');
	});
});

describe('kartenStil', () => {
	it('gibt Fläche und Schrift der Kennung', () => {
		expect(kartenStil('grau')).toBe('background: #f1f0f4; color: #1a1c1e;');
		expect(kartenStil('waldgruen')).toContain('#16330a');
	});

	it('zeichnet zu einer unbekannten Kennung die weiße Karte', () => {
		expect(kartenStil('bg-white text-black')).toBe(kartenStil('weiss'));
		expect(kartenStil(undefined)).toBe('background: #ffffff; color: #000000;');
	});
});

describe('Kennungen im Entwurf', () => {
	beforeEach(() => {
		resetDesign();
	});

	it('die Vorgaben und die Vorlage Waldgrün stehen in der Liste der Auswahl', () => {
		const kennungen = KARTEN_HINTERGRUENDE.map((k) => k.value);
		expect(kennungen).toContain(HINTERGRUND_VORDERSEITE);
		expect(kennungen).toContain(HINTERGRUND_RUECKSEITE);
		expect(kennungen).toContain(WALDGRUEN_THEME);
		expect(idStore.front.theme).toBe(HINTERGRUND_VORDERSEITE);
		expect(idStore.back.theme).toBe(HINTERGRUND_RUECKSEITE);
	});

	it('ein gespeicherter Entwurf mit Klassenlisten kommt mit Kennungen im Store an', () => {
		applyDesign({
			front: { elements: [], theme: ALTE_WERTE.bernstein },
			back: { elements: [], theme: ALTE_WERTE.waldgruen }
		});
		expect(idStore.front.theme).toBe('bernstein');
		expect(idStore.back.theme).toBe('waldgruen');
	});

	it('ein Entwurf ohne Hintergrund behält die Vorgabe der Seite', () => {
		applyDesign({ front: { elements: [], theme: '' }, back: { elements: [] } });
		expect(idStore.front.theme).toBe(HINTERGRUND_VORDERSEITE);
		expect(idStore.back.theme).toBe(HINTERGRUND_RUECKSEITE);
	});
});
